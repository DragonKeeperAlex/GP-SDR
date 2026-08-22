package app

import (
	"strings"
	"testing"
	"time"
)

func TestCharacterizationValidatesRangeAndResolution(t *testing.T) {
	if _, err := validateCharacterizationRequest(CharacterizationRequest{DeviceIDs: []string{"a"}, RangeMode: "custom", StartHz: 100e6, EndHz: 90e6, Points: 96, DwellMilliseconds: 180}); err == nil {
		t.Fatal("reversed custom range should be rejected")
	}
	request, err := validateCharacterizationRequest(CharacterizationRequest{DeviceIDs: []string{"a"}, RangeMode: "antenna", AntennaMinimumHz: 40e6, AntennaMaximumHz: 860e6})
	if err != nil {
		t.Fatal(err)
	}
	if request.StartHz != 40e6 || request.EndHz != 860e6 || request.Points != 96 || request.DwellMilliseconds != 180 {
		t.Fatalf("unexpected defaults: %+v", request)
	}
}

func TestCharacterizationSummaryFindsBestObservedPointAndOverload(t *testing.T) {
	result := DeviceCharacterization{Points: []CharacterizationPoint{
		{FrequencyHz: 100e6, NoiseDBFS: -105, PeakDBFS: -70, SNRDB: 8, RelativeScore: 30, SignalDetected: true},
		{FrequencyHz: 450e6, NoiseDBFS: -98, PeakDBFS: -45, SNRDB: 22, RelativeScore: 95, SignalDetected: true},
		{FrequencyHz: 850e6, NoiseDBFS: -90, PeakDBFS: -20, SNRDB: 30, RelativeScore: 20, ClippedPercent: 1.2},
	}}
	updateCharacterizationSummary(&result)
	if result.BestObservedFrequencyHz != 450e6 || result.QuietestFrequencyHz != 100e6 || result.StrongestFrequencyHz != 850e6 || result.OverloadedPoints != 1 {
		t.Fatalf("unexpected characterization summary: %+v", result)
	}
	if !strings.Contains(result.Recommendation, "Overload") {
		t.Fatalf("expected overload advice, got %q", result.Recommendation)
	}
}

func TestCharacterizationRunsTwoReceiversInParallelAndExports(t *testing.T) {
	manager := NewCharacterizationManager(t.TempDir())
	devices := []SDRDevice{
		{ID: "simulator-0", Name: "Receiver A", Kind: "Simulator", Connected: true, Available: true, FrequencyMinimumHz: 1e6, FrequencyMaximumHz: 6e9},
		{ID: "simulator-1", Name: "Receiver B", Kind: "Simulator", Connected: true, Available: true, FrequencyMinimumHz: 24e6, FrequencyMaximumHz: 1.766e9},
	}
	status, err := manager.Start(devices, CharacterizationRequest{DeviceIDs: []string{"simulator-0", "simulator-1"}, RangeMode: "antenna", AntennaLabel: "40-860 MHz", AntennaMinimumHz: 40e6, AntennaMaximumHz: 860e6, Points: 8, DwellMilliseconds: 80}, nil)
	if err != nil || !status.Running {
		t.Fatalf("characterization did not start: status=%+v error=%v", status, err)
	}
	deadline := time.Now().Add(time.Second)
	for manager.Status().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status = manager.Status()
	if status.Running || len(status.Results) != 2 {
		t.Fatalf("two-receiver characterization did not finish: %+v", status)
	}
	for _, result := range status.Results {
		if result.CompletedPoints != 8 || len(result.Points) != 8 || result.TestedMinimumHz != 40e6 || result.TestedMaximumHz != 860e6 {
			t.Fatalf("unexpected receiver result: %+v", result)
		}
	}
	data, err := manager.CSV()
	if err != nil || !strings.Contains(string(data), "Receiver A") || !strings.Contains(string(data), "40-860 MHz") {
		t.Fatalf("unexpected characterization export: %v %s", err, data)
	}
}
