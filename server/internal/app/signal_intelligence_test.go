package app

import (
	"math"
	"testing"
)

func TestExtractCallsignsFromTextAndNATOPhonetics(t *testing.T) {
	calls := ExtractCallsigns("Net control is W6ABC. Mobile is kilo seven charlie delta echo.")
	if len(calls) != 2 || calls[0] != "K7CDE" || calls[1] != "W6ABC" {
		t.Fatalf("unexpected callsigns: %#v", calls)
	}
}

func TestAnalyzeSignalIQClassifiesAMAndFM(t *testing.T) {
	const rate = 192_000
	am := syntheticIQ(rate, .3, func(index int) (float64, float64) {
		time := float64(index) / rate
		amplitude := .55 + .25*math.Sin(2*math.Pi*1000*time)
		phase := 2 * math.Pi * 12_000 * time
		return amplitude * math.Cos(phase), amplitude * math.Sin(phase)
	})
	amResult := AnalyzeSignalIQ(am, ComplexSigned8, rate, 12_000, 12_500)
	if amResult.Modulation != "AM" {
		t.Fatalf("expected AM, got %+v", amResult)
	}
	fm := syntheticIQ(rate, .3, func(index int) (float64, float64) {
		time := float64(index) / rate
		phase := 2*math.Pi*12_000*time + 4*math.Sin(2*math.Pi*1000*time)
		return .75 * math.Cos(phase), .75 * math.Sin(phase)
	})
	fmResult := AnalyzeSignalIQ(fm, ComplexSigned8, rate, 12_000, 12_500)
	if fmResult.Modulation != "NFM" {
		t.Fatalf("expected NFM, got %+v", fmResult)
	}
}

func syntheticIQ(rate int, seconds float64, sample func(int) (float64, float64)) []byte {
	count := int(float64(rate) * seconds)
	result := make([]byte, count*2)
	for index := 0; index < count; index++ {
		i, q := sample(index)
		result[index*2] = byte(int8(math.Round(clamp(i, -.99, .99) * 127)))
		result[index*2+1] = byte(int8(math.Round(clamp(q, -.99, .99) * 127)))
	}
	return result
}
