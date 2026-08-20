package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestNFMDemodulatesToneWithFrequencyOffset(t *testing.T) {
	const inputRate = 256_000
	data := syntheticFM(inputRate, 0.35, 31_250, 1_000, 5_000)
	result, err := DemodulateIQ(data, ComplexSigned8, inputRate, 31_250, "nfm")
	if err != nil {
		t.Fatal(err)
	}
	if frequency := zeroCrossingFrequency(result.Audio[len(result.Audio)/4:], result.AudioRateHz); frequency < 900 || frequency > 1100 {
		t.Fatalf("expected 1 kHz audio, got %.1f Hz", frequency)
	}
}

func TestWFMDemodulatesBroadcastToneAtHackRFRate(t *testing.T) {
	const inputRate = 2_000_000
	data := syntheticFM(inputRate, 0.2, 0, 1_000, 75_000)
	result, err := DemodulateIQ(data, ComplexSigned8, inputRate, 0, "wfm")
	if err != nil {
		t.Fatal(err)
	}
	if frequency := zeroCrossingFrequency(result.Audio[len(result.Audio)/4:], result.AudioRateHz); frequency < 850 || frequency > 1150 {
		t.Fatalf("expected 1 kHz WFM audio, got %.1f Hz", frequency)
	}
}

func TestAMDemodulatesTone(t *testing.T) {
	const inputRate = 192_000
	samples := int(0.35 * inputRate)
	data := make([]byte, samples*2)
	for n := 0; n < samples; n++ {
		time := float64(n) / inputRate
		amplitude := 0.55 + 0.35*math.Sin(2*math.Pi*700*time)
		phase := 2 * math.Pi * 20_000 * time
		data[n*2] = byte(int8(math.Round(amplitude * math.Cos(phase) * 120)))
		data[n*2+1] = byte(int8(math.Round(amplitude * math.Sin(phase) * 120)))
	}
	result, err := DemodulateIQ(data, ComplexSigned8, inputRate, 20_000, "am")
	if err != nil {
		t.Fatal(err)
	}
	if frequency := zeroCrossingFrequency(result.Audio[len(result.Audio)/3:], result.AudioRateHz); frequency < 620 || frequency > 780 {
		t.Fatalf("expected 700 Hz audio, got %.1f Hz", frequency)
	}
}

func TestWriteMonoWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := WriteMonoWAV(path, []int16{0, 100, -100}, 16_000); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 50 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("invalid WAV: %d bytes", len(data))
	}
}

func syntheticFM(rate int, seconds, offset, tone, deviation float64) []byte {
	samples := int(seconds * float64(rate))
	data := make([]byte, samples*2)
	for n := 0; n < samples; n++ {
		time := float64(n) / float64(rate)
		phase := 2*math.Pi*offset*time - (deviation/tone)*math.Cos(2*math.Pi*tone*time)
		data[n*2] = byte(int8(math.Round(math.Cos(phase) * 120)))
		data[n*2+1] = byte(int8(math.Round(math.Sin(phase) * 120)))
	}
	return data
}

func zeroCrossingFrequency(samples []int16, rate int) float64 {
	crossings := 0
	for index := 1; index < len(samples); index++ {
		if samples[index-1] <= 0 && samples[index] > 0 {
			crossings++
		}
	}
	return float64(crossings) * float64(rate) / float64(len(samples))
}
