package app

import (
	"context"
	"os"
	"testing"
	"time"
)

// Opt-in connection check: synthetic metadata only, no saved media or location.
func TestConfiguredRemoteEvidenceAnalysis(t *testing.T) {
	endpoint := os.Getenv("GPSDR_REMOTE_AI_TEST_ENDPOINT")
	model := os.Getenv("GPSDR_REMOTE_AI_TEST_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("set explicit private endpoint/model for the bounded synthetic acceptance check")
	}
	if err := validateLocalAIEndpoint(endpoint); err != nil {
		t.Fatal(err)
	}
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	event := TransmissionEvent{ID: "synthetic-remote-acceptance", FrequencyHz: 145500000, BandwidthHz: 12500, Modulation: "Unknown", SignalDBFS: -50, NoiseDBFS: -80, Simulated: true}
	result, err := analyzer.analyzeWithConfig(ctx, event, LocalAIConfig{Enabled: true, Endpoint: endpoint, Model: model, Profile: "deep", MinimumConfidence: 55, ContextLength: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary == "" {
		t.Fatal("remote model returned no usable evidence summary")
	}
	t.Log("Synthetic structured response accepted; no real RF/media acceptance implied")
}
