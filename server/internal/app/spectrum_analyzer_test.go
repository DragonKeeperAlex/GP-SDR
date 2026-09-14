package app

import "testing"

func TestSpectrumAnalyzerClearPreservesRangeAndRemovesBins(t *testing.T) {
	runtime := &Runtime{spectrumAnalyzer: &spectrumAnalyzerState{SpectrumAnalyzerStatus: SpectrumAnalyzerStatus{StartHz: 88e6, EndHz: 108e6, BinsDBFS: []float64{-80, -20}, Sweeps: 3, Slices: 12}, stop: make(chan struct{})}}
	status := runtime.ClearSpectrumAnalyzer()
	if status.StartHz != 88e6 || status.EndHz != 108e6 || status.Sweeps != 0 || status.Slices != 0 {
		t.Fatalf("unexpected cleared status: %#v", status)
	}
	for _, level := range status.BinsDBFS {
		if level != -120 {
			t.Fatalf("expected cleared floor, got %v", level)
		}
	}
}

func TestSpectrumAnalyzerRejectsMissingReceiver(t *testing.T) {
	runtime := &Runtime{}
	if _, err := runtime.StartSpectrumAnalyzer(SpectrumAnalyzerRequest{DeviceIDs: []string{"missing"}, StartHz: 88e6, EndHz: 108e6}); err == nil {
		t.Fatal("expected a missing-receiver error")
	}
}
