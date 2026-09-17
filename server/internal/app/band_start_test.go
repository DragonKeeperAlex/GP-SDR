package app

import (
	"strings"
	"testing"
)

func TestUnfittableBandDoesNotStopExistingSession(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	limit := 3200000.0
	runtime.devices = []SDRDevice{{ID: "narrow-rtl", Name: "RTL-SDR 1", Kind: "RTL-SDR", SampleRateLimit: &limit, Connected: true, Available: true}}
	runtime.running = true
	err = runtime.StartOnDevice("be8e8ba2-ef4d-47f4-875f-f489bc8d894b", "narrow-rtl", nil)
	if err == nil || !strings.Contains(err.Error(), "simultaneously") {
		t.Fatalf("expected actionable bandwidth error, got %v", err)
	}
	if !runtime.running {
		t.Fatal("rejected bank stopped existing session")
	}
}
