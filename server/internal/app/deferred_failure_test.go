package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeferredModelFailureRetainsIQ(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(root, "audio.wav")
	if err := WriteMonoWAV(audio, make([]int16, 8000), 8000); err != nil {
		t.Fatal(err)
	}
	iq := filepath.Join(root, "capture.cs8")
	if err := os.WriteFile(iq, []byte{1, 2, 3, 4}, 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }))
	defer server.Close()
	ai := NewLocalAIAnalyzer(root)
	ai.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "balanced"}
	event := TransmissionEvent{ID: "failed-model", StartedAt: time.Now(), FrequencyHz: 98.1e6, AudioPath: &audio, IQPath: &iq}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Events: store, localAI: ai, transcriber: NewTranscriber(root)}
	if err := runtime.analyzeStoredEvent(event, make(chan struct{})); err == nil || !strings.Contains(err.Error(), "local model analysis") {
		t.Fatalf("model failure hidden: %v", err)
	}
	if _, err := os.Stat(iq); err != nil {
		t.Fatal("failed model removed IQ", err)
	}
}

func TestDeferredCorruptAudioReportsFailure(t *testing.T) {
	root := t.TempDir()
	audio := filepath.Join(root, "corrupt.wav")
	if err := os.WriteFile(audio, []byte("not a WAV"), 0600); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{}
	err := runtime.analyzeStoredEvent(TransmissionEvent{ID: "corrupt", AudioPath: &audio}, make(chan struct{}))
	if err == nil || !strings.Contains(err.Error(), "read saved audio") {
		t.Fatalf("invalid audio hidden: %v", err)
	}
}

func TestDeferredMissingConfiguredTranscriptionRetainsIQ(t *testing.T) {
	root := t.TempDir()
	audio, iq := filepath.Join(root, "audio.wav"), filepath.Join(root, "capture.cs8")
	if err := WriteMonoWAV(audio, make([]int16, 8000), 8000); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iq, []byte{1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{transcriber: &Transcriber{executable: "whisper", model: filepath.Join(root, "missing-model")}}
	err := runtime.analyzeStoredEvent(TransmissionEvent{ID: "missing-model", AudioPath: &audio, IQPath: &iq}, make(chan struct{}))
	if err == nil || !strings.Contains(err.Error(), "transcription") {
		t.Fatalf("configured model failure hidden: %v", err)
	}
	if _, err := os.Stat(iq); err != nil {
		t.Fatal("transcription failure removed IQ")
	}
}

func TestDeferredGroupModelFailureIsReported(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }))
	defer server.Close()
	ai := NewLocalAIAnalyzer(root)
	ai.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "balanced"}
	runtime := &Runtime{Events: store, localAI: ai}
	err = runtime.combineDeferredGroup([]TransmissionEvent{{ID: "group", FrequencyHz: 98.1e6}}, make(chan struct{}))
	if err == nil || !strings.Contains(err.Error(), "combine group evidence") {
		t.Fatalf("group failure hidden: %v", err)
	}
	ai.config.Enabled = false
	if err := runtime.combineDeferredGroup([]TransmissionEvent{{ID: "group"}}, make(chan struct{})); err != nil {
		t.Fatal("disabled optional AI failed", err)
	}
}

func TestDeferredGroupFailurePreservesPreviouslyAnalyzedIQ(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	audio, iq := filepath.Join(root, "audio.wav"), filepath.Join(root, "capture.cs8")
	if err := WriteMonoWAV(audio, make([]int16, 8000), 8000); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iq, []byte{1, 2, 3, 4}, 0600); err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 {
			http.Error(w, "offline", 503)
			return
		}
		_, _ = w.Write([]byte(`{"response":"{\"signalFamily\":\"Unknown\",\"modulation\":\"UNKNOWN\",\"summary\":\"Insufficient evidence\",\"confidence\":0.2,\"evidence\":[],\"callsigns\":[]}"}`))
	}))
	defer server.Close()
	ai := NewLocalAIAnalyzer(root)
	ai.config = LocalAIConfig{Enabled: true, Endpoint: server.URL, Model: "test", Profile: "balanced"}
	event := TransmissionEvent{ID: "group-failure", StartedAt: time.Now(), FrequencyHz: 98.1e6, AudioPath: &audio, IQPath: &iq, AnalysisStatus: "pending"}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Events: store, localAI: ai}
	runtime.processDeferredGroup([]TransmissionEvent{event}, make(chan struct{}))
	current, _ := store.Get(event.ID)
	if current.AnalysisStatus != "error" {
		t.Fatalf("group failure marked %s", current.AnalysisStatus)
	}
	if _, err := os.Stat(iq); err != nil {
		t.Fatal("group failure lost IQ", err)
	}
	if runtime.analysisCompleted != 0 || runtime.analysisFailed != 1 {
		t.Fatal("misleading group counters")
	}
}
