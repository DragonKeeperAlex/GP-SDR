package app

import (
	"math"
	"path/filepath"
	"testing"
)

func TestCalibrationStorePersistsPerDevice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "calibrations.json")
	store, err := NewCalibrationStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := DeviceCalibration{DeviceID: "hackrf-serial", DeviceKind: "HackRF", PPMCorrection: -3, IQGain: 1.08, IQPhase: -2.5, DCRemoval: true}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewCalibrationStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reloaded.Get(want.DeviceID)
	if !ok || got.PPMCorrection != want.PPMCorrection || got.IQGain != want.IQGain || got.IQPhase != want.IQPhase {
		t.Fatalf("calibration did not round-trip: %#v", got)
	}
}

func TestAnalyzeCalibrationMeasuresIQGainMismatch(t *testing.T) {
	const rate = 2_400_000
	data := make([]byte, 32768)
	for sample := 0; sample < len(data)/2; sample++ {
		phase := 2 * math.Pi * 31_250 * float64(sample) / rate
		data[sample*2] = byte(int8(math.Round(math.Cos(phase) * 95)))
		data[sample*2+1] = byte(int8(math.Round(math.Sin(phase) * 70)))
	}
	calibration, err := analyzeCalibration(data, ComplexSigned8, rate, 100_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if calibration.IQGain < 1.25 || calibration.IQGain > 1.45 {
		t.Fatalf("expected Q correction near 1.36, got %.3f", calibration.IQGain)
	}
}
