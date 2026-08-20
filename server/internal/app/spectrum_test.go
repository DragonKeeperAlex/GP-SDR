package app

import (
	"math"
	"testing"
)

func TestMeasureChannelSpectrumFindsOffsetCarrier(t *testing.T) {
	const rate = 1_000_000
	data := make([]byte, 32_768*2)
	seed := uint32(1)
	for index := 0; index < len(data)/2; index++ {
		seed = seed*1664525 + 1013904223
		noiseI := (float64(int((seed>>24)&255)-128) / 128) * .025
		seed = seed*1664525 + 1013904223
		noiseQ := (float64(int((seed>>24)&255)-128) / 128) * .025
		phase := 2 * math.Pi * 100_000 * float64(index) / rate
		i := .55*math.Cos(phase) + noiseI
		q := .55*math.Sin(phase) + noiseQ
		data[index*2] = byte(int8(math.Round(i * 120)))
		data[index*2+1] = byte(int8(math.Round(q * 120)))
	}
	channels := []ChannelDefinition{
		{ID: "signal", FrequencyHz: 100.1e6, BandwidthHz: 12_500},
		{ID: "empty", FrequencyHz: 100.3e6, BandwidthHz: 12_500},
	}
	levels, err := MeasureChannelSpectrum(data, ComplexSigned8, rate, 100e6, channels)
	if err != nil {
		t.Fatal(err)
	}
	if levels["signal"].SignalDB-levels["signal"].NoiseDB < 15 {
		t.Fatalf("carrier SNR was too low: %#v", levels["signal"])
	}
	if levels["signal"].SignalDB-levels["empty"].SignalDB < 15 {
		t.Fatalf("carrier was not separated from empty channel: %#v", levels)
	}
}

func TestBuildSpectrumSnapshotFindsCarrierAndFrequencySpan(t *testing.T) {
	const rate = 1_000_000
	const offset = 100_000
	data := make([]byte, 4096*2)
	for index := 0; index < len(data)/2; index++ {
		phase := 2 * math.Pi * offset * float64(index) / rate
		data[index*2] = byte(int8(math.Round(math.Cos(phase) * 120)))
		data[index*2+1] = byte(int8(math.Round(math.Sin(phase) * 120)))
	}
	snapshot, err := BuildSpectrumSnapshot(CaptureSpec{CenterFrequencyHz: 100_000_000, SampleRateHz: rate}, data, ComplexSigned8)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StartFrequencyHz != 99_500_000 || snapshot.EndFrequencyHz != 100_500_000 || len(snapshot.BinsDBFS) != 1024 {
		t.Fatalf("unexpected spectrum metadata: %#v", snapshot)
	}
	peak := 0
	for index := range snapshot.BinsDBFS {
		if snapshot.BinsDBFS[index] > snapshot.BinsDBFS[peak] {
			peak = index
		}
	}
	expected := int((offset + rate/2) * 1024 / rate)
	if peak < expected-2 || peak > expected+2 {
		t.Fatalf("expected peak near bin %d, got %d", expected, peak)
	}
}
