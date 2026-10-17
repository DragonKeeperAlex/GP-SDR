package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

const computeFixture = `{"response":"{\"signalFamily\":\"Unknown\",\"modulation\":\"UNKNOWN\",\"summary\":\"Insufficient synthetic evidence\",\"confidence\":0.1,\"evidence\":[],\"callsigns\":[]}"}`

func TestComputeRoutingPrefersHomeAndFallsBack(t *testing.T) {
	for _, scenario := range []string{"ready", "paused", "offline", "missing-model", "fails-generation", "local", "deferred", "off"} {
		t.Run(scenario, func(t *testing.T) {
			var localCalls, homeCalls atomic.Int32
			local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					localCalls.Add(1)
				}
				w.Write([]byte(computeFixture))
			}))
			defer local.Close()
			home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/guard/status":
					if scenario == "offline" {
						http.Error(w, "offline", 503)
						return
					}
					json.NewEncoder(w).Encode(map[string]bool{"paused": scenario == "paused" || scenario == "deferred"})
				case "/api/tags":
					if scenario == "offline" {
						http.Error(w, "offline", 503)
						return
					}
					if scenario == "missing-model" {
						w.Write([]byte(`{"models":[]}`))
						return
					}
					w.Write([]byte(`{"models":[{"name":"home"}]}`))
				default:
					homeCalls.Add(1)
					if scenario == "fails-generation" {
						http.Error(w, "lost", 503)
						return
					}
					w.Write([]byte(computeFixture))
				}
			}))
			defer home.Close()
			a := NewLocalAIAnalyzer(t.TempDir())
			mode := "auto"
			if scenario == "local" || scenario == "deferred" || scenario == "off" {
				mode = scenario
			}
			a.config = normalizeLocalAIConfig(LocalAIConfig{ComputeMode: mode, Endpoint: local.URL, Model: "pi", RemoteEndpoint: home.URL, RemoteModel: "home"})
			queued := false
			a.deferEvidence = func(TransmissionEvent) error { queued = true; return nil }
			result, err := a.Analyze(context.Background(), TransmissionEvent{ID: "fixture", FrequencyHz: 462.55e6})
			switch scenario {
			case "ready":
				if err != nil || homeCalls.Load() != 1 || localCalls.Load() != 0 || !strings.Contains(result.Engine, "home model") {
					t.Fatalf("home preference failed: %v %s", err, result.Engine)
				}
			case "deferred":
				if err == nil || !queued || localCalls.Load() != 0 || homeCalls.Load() != 0 {
					t.Fatal("home-only must queue while paused")
				}
			case "off":
				if err == nil || localCalls.Load() != 0 || homeCalls.Load() != 0 {
					t.Fatal("off did not stop analysis")
				}
			default:
				if err != nil || localCalls.Load() != 1 {
					t.Fatalf("local fallback failed: %v", err)
				}
			}
		})
	}
}
func TestComputeQueueCanRunLocally(t *testing.T) {
	var calls atomic.Int32
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(computeFixture)) }))
	defer local.Close()
	a := NewLocalAIAnalyzer(t.TempDir())
	q := NewDeepAnalysisQueue(t.TempDir(), a)
	if _, _, err := q.Enqueue(TransmissionEvent{ID: "saved"}); err != nil {
		t.Fatal(err)
	}
	cfg := normalizeLocalAIConfig(LocalAIConfig{ComputeMode: "local", Endpoint: local.URL, Model: "pi"})
	jobs := q.Process(context.Background(), cfg, 1)
	if calls.Load() != 1 || len(jobs) != 1 || jobs[0].Status != "complete" {
		t.Fatal("queued work did not follow local selection")
	}
}
func TestComputeSelectionOverLANPreservesEndpoints(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	runtime.localAI.config = normalizeLocalAIConfig(LocalAIConfig{ComputeMode: "off", Endpoint: "http://127.0.0.1:1", Model: "pi", RemoteEndpoint: "http://127.0.0.1:2", RemoteModel: "home"})
	server := NewServer(runtime, fstest.MapFS{}, "127.0.0.1", 8073, "fixture-auth")
	request := httptest.NewRequest("PUT", "/api/local-ai/compute", bytes.NewBufferString(`{"mode":"local","remoteEndpoint":"http://1.2.3.4:1234"}`))
	request.RemoteAddr = "192.168.1.50:1234"
	request.Header.Set("X-GP-SDR-Token", "fixture-auth")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatal("mode endpoint must reject configuration fields", response.Code)
	}
	request = httptest.NewRequest("PUT", "/api/local-ai/compute", strings.NewReader(`{"mode":"local"}`))
	request.RemoteAddr = "192.168.1.50:1234"
	request.Header.Set("X-GP-SDR-Token", "fixture-auth")
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	cfg := runtime.localAI.config
	if cfg.ComputeMode != "local" || cfg.RemoteEndpoint != "http://127.0.0.1:2" || cfg.RemoteModel != "home" {
		t.Fatal("selector changed endpoint/model")
	}
	if loaded := NewLocalAIAnalyzer(filepath.Dir(filepath.Dir(runtime.localAI.path))); loaded.config.ComputeMode != "local" {
		t.Fatal("selection did not persist")
	}
	request = httptest.NewRequest("PUT", "/api/local-ai/compute", strings.NewReader(`{"mode":"auto"}`))
	request.RemoteAddr = "192.168.1.50:1234"
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatal("unauthenticated mode change allowed")
	}
	if _, err = runtime.localAI.SetComputeMode("invalid"); err == nil {
		t.Fatal("invalid mode allowed")
	}
}

func TestComputeTranscriptionQueuesWithoutStartingInference(t *testing.T) {
	var calls atomic.Int32
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(computeFixture)) }))
	defer model.Close()
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	runtime.localAI.mu.Lock()
	runtime.localAI.config = normalizeLocalAIConfig(LocalAIConfig{ComputeMode: "local", Endpoint: model.URL, Model: "pi"})
	runtime.localAI.mu.Unlock()
	event := TransmissionEvent{ID: "transcription-fixture", FrequencyHz: 462.55e6, Transcript: ptr("synthetic radio check")}
	if err := runtime.Events.Append(event); err != nil {
		t.Fatal(err)
	}
	runtime.p25Transcription.onComplete(event.ID)
	if calls.Load() != 0 || len(runtime.deepQueue.List()) != 1 {
		t.Fatal("transcription callback must durably enqueue without waiting on inference")
	}
}
