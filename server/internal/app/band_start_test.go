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

func TestStartOnDeviceUsesRuntimeOnlyOpenMonitor(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	profile, ok := runtime.Profiles.Get("be8e8ba2-ef4d-47f4-875f-f489bc8d894b")
	if !ok {
		t.Fatal("built-in GMRS profile missing")
	}
	if profile.Settings.MonitorOpen {
		t.Fatal("saved profile unexpectedly has a runtime monitor flag")
	}
	if err := runtime.StartOnDevice(profile.ID, "demo-hackrf", nil); err != nil {
		t.Fatal(err)
	}
	runtime.mu.RLock()
	active := runtime.active
	monitorOpen := active != nil && active.Settings.MonitorOpen
	runtime.mu.RUnlock()
	if !monitorOpen {
		t.Fatal("band start did not enable the runtime-only monitor")
	}
	persisted, ok := runtime.Profiles.Get(profile.ID)
	if !ok || persisted.Settings.MonitorOpen {
		t.Fatal("band start persisted the runtime-only monitor flag")
	}
}

func TestStartOnDevicePinsOneReceiverForP25(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	var profile ScanProfile
	for _, candidate := range runtime.Profiles.All() {
		if candidate.Name == "San Ramon · EBRCS P25" {
			profile = candidate
			break
		}
	}
	if profile.ID == "" {
		t.Fatal("built-in EBRCS profile missing")
	}
	if err := runtime.StartOnDevice(profile.ID, "demo-hackrf", nil); err != nil {
		t.Fatalf("selected single-receiver P25 start failed: %v", err)
	}
	runtime.mu.RLock()
	plan := append([]ReceiverPlanItem(nil), runtime.plan...)
	runtime.mu.RUnlock()
	if len(plan) != 1 || plan[0].Role != "control" || plan[0].DeviceID == nil || *plan[0].DeviceID != "demo-hackrf" {
		t.Fatalf("P25 selection did not reserve one control receiver: %#v", plan)
	}
}
