package app

import (
	"math"
	"testing"
)

func TestDetectCTCSSStandardTone(t *testing.T) {
	const rate = 48_000
	samples := make([]int16, rate/2)
	for index := range samples {
		voice := 1200 * math.Sin(2*math.Pi*900*float64(index)/rate)
		tone := 5000 * math.Sin(2*math.Pi*123*float64(index)/rate)
		samples[index] = int16(math.Round(voice + tone))
	}
	tone, confidence, ok := DetectCTCSS(samples, rate)
	if !ok || tone != 123 || confidence < .7 {
		t.Fatalf("expected 123.0 Hz CTCSS, got %.1f confidence %.2f ok=%t", tone, confidence, ok)
	}
}

func TestDetectCTCSSRejectsVoiceOnly(t *testing.T) {
	const rate = 48_000
	samples := make([]int16, rate/2)
	for index := range samples {
		samples[index] = int16(5000 * math.Sin(2*math.Pi*900*float64(index)/rate))
	}
	if tone, confidence, ok := DetectCTCSS(samples, rate); ok {
		t.Fatalf("unexpected CTCSS %.1f confidence %.2f", tone, confidence)
	}
}
