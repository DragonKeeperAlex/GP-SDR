package app

import (
	"math"
	"testing"
)

func TestNormalizeFPVRequestDefaultsAndCapsRate(t *testing.T) {
	limit := 8_000_000.0
	device := SDRDevice{Kind: "HackRF", Driver: "SoapySDR:HackRF", SampleRateLimit: &limit}
	request, err := normalizeFPVRequest(FPVReceiverRequest{FrequencyHz: 5_917_000_000}, device)
	if err != nil {
		t.Fatal(err)
	}
	if request.Standard != "ntsc" || request.SampleRateHz != 8_000_000 {
		t.Fatalf("unexpected normalized request: %+v", request)
	}
}

func TestNormalizeFPVRequestRejectsInvalidStandardAndNonFiniteGain(t *testing.T) {
	device := SDRDevice{Kind: "HackRF", DriverSampleRateMaximumHz: 16_000_000}
	if _, err := normalizeFPVRequest(FPVReceiverRequest{FrequencyHz: 5_800_000_000, Standard: "secam"}, device); err == nil {
		t.Fatal("expected unsupported video standard to be rejected")
	}
	if _, err := normalizeFPVRequest(FPVReceiverRequest{FrequencyHz: 5_800_000_000, GainDB: math.NaN()}, device); err == nil {
		t.Fatal("expected non-finite gain to be rejected")
	}
}
