package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLocalAIRejectsPublicEndpoint(t *testing.T) {
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	_, err := analyzer.Update(LocalAIConfig{Enabled: true, Endpoint: "http://example.com", Model: "test", Profile: "lightweight", MinimumConfidence: 50})
	if err == nil || !strings.Contains(err.Error(), "private-network") {
		t.Fatalf("expected private-network guard, got %v", err)
	}
}

func TestLocalAIAllowsPrivateOllamaServer(t *testing.T) {
	if err := validateLocalAIEndpoint("http://192.168.1.54:11434"); err != nil {
		t.Fatalf("private Ollama endpoint rejected: %v", err)
	}
	if err := validateLocalAIEndpoint("https://10.0.0.8"); err != nil {
		t.Fatalf("private HTTPS endpoint rejected: %v", err)
	}
}

func TestLocalAIContextCanUseConfiguredLongWindow(t *testing.T) {
	if got := localAIContext(LocalAIConfig{Profile: "deep", ContextLength: 262144}); got != 262144 {
		t.Fatalf("context = %d, want 262144", got)
	}
	if got := localAIContext(LocalAIConfig{Profile: "balanced"}); got != 4096 {
		t.Fatalf("automatic balanced context = %d, want 4096", got)
	}
}

func TestLocalAIReferenceMatchesRespectReceiveLocation(t *testing.T) {
	profiles := &ProfileStore{profiles: map[string]ScanProfile{
		"near": {ID: "near", Name: "Alameda County", Summary: "RadioReference import", ReferenceArea: &ProfileReferenceArea{Provider: "RadioReference", Latitude: 37.7, Longitude: -121.8, RadiusMiles: 25}, Channels: []ChannelDefinition{{Name: "Local dispatch", FrequencyHz: 774.5e6, Mode: "P25"}}},
		"far":  {ID: "far", Name: "Los Angeles", Summary: "RadioReference import", ReferenceArea: &ProfileReferenceArea{Provider: "RadioReference", Latitude: 34.05, Longitude: -118.24, RadiusMiles: 25}, Channels: []ChannelDefinition{{Name: "Distant reuse", FrequencyHz: 774.5e6, Mode: "P25"}}},
	}}
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.SetReferenceProfiles(profiles)
	matches := analyzer.localReferenceMatches(TransmissionEvent{FrequencyHz: 774.5e6, BandwidthHz: 12500, Location: &ObservationLocation{Latitude: 37.68, Longitude: -121.77, Label: "Livermore", Precision: "exact"}})
	if len(matches) != 1 || matches[0]["name"] != "Local dispatch" || matches[0]["locationVerified"] != true {
		t.Fatalf("unexpected location-filtered references: %#v", matches)
	}
}

func TestLocalAIStatusListsGenerationModelsOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3.5:9b","size":6000000000,"details":{"parameter_size":"9B","quantization_level":"Q4_K_M"}},{"name":"nomic-embed-text:v1.5","size":100}]}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "qwen3.5:9b", Profile: "balanced", MinimumConfidence: 55}
	status := analyzer.Status()
	if status.State != "ready" || len(status.Models) != 1 || status.Models[0].Name != "qwen3.5:9b" || status.Models[0].ParameterSize != "9B" {
		t.Fatalf("unexpected model inventory: %+v", status)
	}
}

func TestLocalAIBenchmarkComparesInstalledModelsWithoutChangingSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[{"name":"fast:1b"},{"name":"careful:2b"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"Unknown\",\"modulation\":\"UNKNOWN\",\"summary\":\"Insufficient evidence\",\"confidence\":0.2,\"evidence\":[],\"callsigns\":[]}"}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "fast:1b", Profile: "lightweight", MinimumConfidence: 55}
	if _, err := analyzer.StartBenchmark([]string{"fast:1b", "careful:2b"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for analyzer.BenchmarkStatus().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	status := analyzer.BenchmarkStatus()
	if status.Running || len(status.Results) != 2 || status.Results[0].Cases != 5 || status.Results[0].StructuredPercent != 100 {
		t.Fatalf("unexpected benchmark result: %#v", status)
	}
	if analyzer.config.Model != "fast:1b" {
		t.Fatal("benchmark changed the configured model")
	}
}

func TestInstalledLocalAIEndToEnd(t *testing.T) {
	if os.Getenv("GPSDR_LOCAL_AI_SMOKE") != "1" {
		t.Skip("set GPSDR_LOCAL_AI_SMOKE=1 to exercise the installed local model")
	}
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: "http://127.0.0.1:11434", Model: "qwen2.5:1.5b", Profile: "lightweight", MinimumConfidence: 55}
	result, err := analyzer.Analyze(context.Background(), TransmissionEvent{FrequencyHz: 462.55e6, BandwidthHz: 20e3, Modulation: "NFM", Transcript: ptr("K6ABC radio check"), Analysis: &SignalIntelligence{Modulation: "NFM", SignalFamily: "Analog frequency", Confidence: .82}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Modulation != "NFM" || result.Engine == "" || result.Summary == "" {
		t.Fatalf("unexpected installed-model result: %+v", result)
	}
}

func TestLocalAIAnalyzesOnlyBoundedMetadata(t *testing.T) {
	var prompt string
	var think *bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[]}`))
			return
		}
		var body struct {
			Prompt string `json:"prompt"`
			Think  *bool  `json:"think"`
		}
		_ = decodeJSONRequest(r, &body)
		prompt = body.Prompt
		think = body.Think
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"Analog voice\",\"modulation\":\"NFM\",\"summary\":\"Likely local analog voice; no protocol frames decoded.\",\"confidence\":0.78,\"evidence\":[\"NFM waveform\"],\"callsigns\":[\"K6ABC\"]}"}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "lightweight", MinimumConfidence: 55}
	// httptest uses loopback but its URL passes the same production guard.
	result, err := analyzer.Analyze(context.Background(), TransmissionEvent{FrequencyHz: 462.55e6, Modulation: "NFM", Transcript: ptr("K6ABC testing")})
	if err != nil {
		t.Fatal(err)
	}
	if result.SignalFamily != "Analog voice" || result.Modulation != "NFM" || len(result.Callsigns) != 1 {
		t.Fatalf("unexpected analysis: %+v", result)
	}
	if strings.Contains(prompt, "IQPath") || strings.Contains(prompt, "AudioPath") {
		t.Fatal("model prompt included capture paths")
	}
	if think == nil || *think {
		t.Fatal("Ollama reasoning was not disabled for structured radio analysis")
	}
}

func TestLocalAICannotOverrideMeasuredModulation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"Digital candidate\",\"modulation\":\"DMR\",\"summary\":\"Possible digital signal.\",\"confidence\":0.94,\"evidence\":[],\"callsigns\":[]}"}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "lightweight", MinimumConfidence: 55}
	result, err := analyzer.Analyze(context.Background(), TransmissionEvent{Modulation: "NFM", Analysis: &SignalIntelligence{Modulation: "NFM", Confidence: .8}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Modulation != "NFM" || result.Confidence > .69 {
		t.Fatalf("model overrode DSP evidence: %+v", result)
	}
}

func TestLocalAICorrectsImpossibleFrequencyBand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"VHF analog\",\"modulation\":\"NFM\",\"summary\":\"Voice\",\"confidence\":0.9,\"evidence\":[],\"callsigns\":[]}"}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "lightweight", MinimumConfidence: 55}
	result, err := analyzer.Analyze(context.Background(), TransmissionEvent{FrequencyHz: 462.55e6, Modulation: "NFM"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(result.SignalFamily), "VHF") || !strings.Contains(strings.ToUpper(result.SignalFamily), "UHF") || result.Confidence > .69 {
		t.Fatalf("impossible model band survived: %+v", result)
	}
}

func TestLocalAIDropsPlaceholderCallsigns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"Unknown\",\"modulation\":\"UNKNOWN\",\"summary\":\"Insufficient evidence\",\"confidence\":0.2,\"evidence\":[],\"callsigns\":[\"UNKNOWN\",\"N/A\",\"K6ABC\"]}"}`))
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "lightweight", MinimumConfidence: 55}
	result, err := analyzer.Analyze(context.Background(), TransmissionEvent{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Callsigns) != 1 || result.Callsigns[0] != "K6ABC" {
		t.Fatalf("placeholder callsigns were retained: %#v", result.Callsigns)
	}
}

func decodeJSONRequest(request *http.Request, destination any) error {
	return json.NewDecoder(request.Body).Decode(destination)
}
