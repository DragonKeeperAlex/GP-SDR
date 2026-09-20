package app

import (
	"math"
	"os"
	"testing"
	"time"
)

// Opt-in physical acceptance test. It transmits only when explicitly enabled
// on the Pi, keeping ordinary regression runs receive-only.
func TestRelayHardwareGeneratedTone(t *testing.T) {
	if os.Getenv("GP_SDR_RELAY_HARDWARE") != "1" {
		t.Skip("set GP_SDR_RELAY_HARDWARE=1 on the Pi for the controlled hardware test")
	}
	hub := NewAudioHub()
	relay := NewRelayRuntime(hub)
	status := relay.Start()
	if !status.Active {
		t.Fatalf("relay did not start: %+v", status)
	}
	samples := make([]int16, 8000*2)
	for i := range samples {
		samples[i] = int16(math.Sin(2*math.Pi*1000*float64(i)/8000) * 1800)
	}
	time.Sleep(100 * time.Millisecond)
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: samples})
	time.Sleep(2500 * time.Millisecond)
	status = relay.Stop()
	if status.Active {
		t.Fatal("relay remained active after stop")
	}
}
