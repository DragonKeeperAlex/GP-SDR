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
