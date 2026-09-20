package app

import (
	"bytes"
	"math"
	"testing"
)

func TestDecodeNOAAAPTDeterministicFixture(t *testing.T) {
	const lines = 3
	pcm := make([]int16, noaaAPTLineSamples*lines)
	for line := 0; line < lines; line++ {
		for sample := 0; sample < noaaAPTLineSamples; sample++ {
			phase := 2 * math.Pi * float64(sample%32) / 32
			envelope := 5000.0 + float64((sample+line*17)%6000)
			pcm[line*noaaAPTLineSamples+sample] = int16(envelope * (0.6 + 0.4*math.Sin(phase)))
		}
	}
	first, err := DecodeNOAAAPT(pcm, 4160)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecodeNOAAAPT(pcm, 4160)
	if err != nil || !bytes.Equal(first.PNG, second.PNG) {
		t.Fatalf("fixture was not deterministic: err=%v", err)
	}
	if first.Width != 1818 || first.Height != lines || first.Lines != lines || !first.Complete || first.Quality <= 0 {
		t.Fatalf("unexpected image result: %+v", first)
	}
	if err := imageArtifactPNG(first.PNG); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeNOAAAPTRejectsShortAndMarksPartialInput(t *testing.T) {
	if _, err := DecodeNOAAAPT(make([]int16, noaaAPTLineSamples-1), 4160); err == nil {
		t.Fatal("expected short APT input to be rejected")
	}
	result, err := DecodeNOAAAPT(make([]int16, noaaAPTLineSamples*2+1), 4160)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.ProcessingError == "" {
		t.Fatalf("partial APT input was accepted without an explicit limitation: %+v", result)
	}
}
