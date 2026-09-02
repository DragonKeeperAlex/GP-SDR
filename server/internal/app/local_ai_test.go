package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLocalAIRejectsRemoteEndpoint(t *testing.T) {
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	_, err := analyzer.Update(LocalAIConfig{Enabled: true, Endpoint: "http://example.com", Model: "test", Profile: "lightweight", MinimumConfidence: 50})
	if err == nil || !strings.Contains(err.Error(), "localhost") {
		t.Fatalf("expected localhost guard, got %v", err)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[]}`))
			return
		}
		var body struct {
			Prompt string `json:"prompt"`
		}
		_ = decodeJSONRequest(r, &body)
		prompt = body.Prompt
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

func decodeJSONRequest(request *http.Request, destination any) error {
	return json.NewDecoder(request.Body).Decode(destination)
}
