package app

import "testing"

func TestNoiseReductionSuppressesLowLevelStatic(t *testing.T) {
	samples := make([]int16, 800)
	for i := range samples {
		samples[i] = int16((i%11)-5) * 20
	}
	before := audioEnergy(samples)
	applyNoiseReduction(samples, 16000, "strong")
	if after := audioEnergy(samples); after >= before*.65 {
		t.Fatalf("expected static reduction, before=%f after=%f", before, after)
	}
}

func audioEnergy(samples []int16) float64 {
	total := 0.0
	for _, sample := range samples {
		v := float64(sample)
		total += v * v
	}
	return total / float64(len(samples))
}
