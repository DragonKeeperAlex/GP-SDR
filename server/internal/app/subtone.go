package app

import (
	"math"
	"sort"
)

var standardCTCSSTones = []float64{
	67.0, 69.3, 71.9, 74.4, 77.0, 79.7, 82.5, 85.4, 88.5, 91.5, 94.8, 97.4,
	100.0, 103.5, 107.2, 110.9, 114.8, 118.8, 123.0, 127.3, 131.8, 136.5, 141.3,
	146.2, 151.4, 156.7, 159.8, 162.2, 165.5, 167.9, 171.3, 173.8, 177.3, 179.9,
	183.5, 186.2, 189.9, 192.8, 196.6, 199.5, 203.5, 206.5, 210.7, 218.1, 225.7,
	229.1, 233.6, 241.8, 250.3, 254.1,
}

// DetectCTCSS returns a standard analog subtone only when its narrow spectral
// peak is well above both the audio energy and neighboring frequencies.
func DetectCTCSS(samples []int16, sampleRate int) (toneHz, confidence float64, ok bool) {
	if sampleRate < 8000 || len(samples) < sampleRate/5 {
		return 0, 0, false
	}
	maximum := sampleRate / 2
	if maximum > len(samples) {
		maximum = len(samples)
	}
	window := samples[len(samples)-maximum:]
	mean := 0.0
	for _, sample := range window {
		mean += float64(sample)
	}
	mean /= float64(len(window))
	rmsPower := 0.0
	for _, sample := range window {
		value := float64(sample) - mean
		rmsPower += value * value
	}
	rmsPower /= float64(len(window))
	if rmsPower < 16 {
		return 0, 0, false
	}
	type candidate struct{ tone, score, share float64 }
	candidates := make([]candidate, 0, len(standardCTCSSTones))
	for _, tone := range standardCTCSSTones {
		power := goertzelPower(window, sampleRate, tone, mean)
		neighbor := math.Max(goertzelPower(window, sampleRate, tone-4, mean), goertzelPower(window, sampleRate, tone+4, mean))
		candidates = append(candidates, candidate{tone: tone, score: power / math.Max(neighbor, 1), share: power / rmsPower})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].share > candidates[j].share })
	best := candidates[0]
	second := candidates[1]
	if best.share < .025 || best.score < 4 || best.share < second.share*1.35 {
		return 0, 0, false
	}
	confidence = math.Min(.99, .55+math.Min(.25, best.share)+math.Min(.19, (best.score-4)/20))
	return best.tone, confidence, true
}

func goertzelPower(samples []int16, sampleRate int, frequency, mean float64) float64 {
	coefficient := 2 * math.Cos(2*math.Pi*frequency/float64(sampleRate))
	previous, previous2 := 0.0, 0.0
	denominator := math.Max(1, float64(len(samples)-1))
	for index, sample := range samples {
		window := .5 - .5*math.Cos(2*math.Pi*float64(index)/denominator)
		value := (float64(sample) - mean) * window
		current := value + coefficient*previous - previous2
		previous2, previous = previous, current
	}
	power := previous2*previous2 + previous*previous - coefficient*previous*previous2
	return power / (float64(len(samples)) * float64(len(samples)))
}
