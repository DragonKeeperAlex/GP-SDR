package app

import (
	"io"
	"os"
	"testing"
	"time"
)

// This checks actual IQ transport and retuning; it is not proof of modulation
// identification or intelligible audio. Run only with idle, attached receivers.
func TestConnectedReceiverCapture(t *testing.T) {
	if os.Getenv("GPSDR_CAPTURE_TEST") != "1" {
		t.Skip("set GPSDR_CAPTURE_TEST=1 for real receiver capture")
	}
	kind := os.Getenv("GPSDR_TEST_DEVICE_KIND")
	if kind == "" {
		kind = "HackRF"
	}
	defer shutdownLocalRTLSessions()
	var device SDRDevice
	for _, d := range DiscoverDevices(false) {
		if d.Connected && d.Available && d.Kind == kind {
			device = d
			break
		}
	}
	if device.ID == "" {
		t.Fatalf("no available %s", kind)
	}
	rates := []int{2_400_000}
	if kind == "HackRF" {
		rates = []int{5_000_000, 10_000_000, 20_000_000}
	}
	for _, rate := range rates {
		for _, frequency := range []int64{100_100_000, 462_600_000, 772_768_750} {
			stream, err := StartIQStream(device, CaptureSpec{CenterFrequencyHz: frequency, SampleRateHz: rate, GainDB: 20, LNAGainDB: 16, VGAGainDB: 20})
			if err != nil {
				t.Fatal(err)
			}
			data := make([]byte, rate) // half a second of interleaved eight-bit IQ
			done := make(chan error, 1)
			go func() { _, err := io.ReadFull(stream.Reader, data); done <- err }()
			select {
			case err = <-done:
			case <-time.After(10 * time.Second):
				_ = stream.Close()
				t.Fatal("capture timed out")
			}
			_ = stream.Close()
			if err != nil {
				t.Fatal(err)
			}
			var occupied [256]bool
			var clips int
			for _, b := range data {
				occupied[b] = true
				if (stream.Format == ComplexSigned8 && (b == 127 || b == 128)) || (stream.Format == ComplexUnsigned8 && (b == 0 || b == 255)) {
					clips++
				}
			}
			levels := 0
			for _, present := range occupied {
				if present {
					levels++
				}
			}
			if levels < 3 {
				t.Fatalf("stuck IQ stream at %d Hz: %d distinct byte values", frequency, levels)
			}
			t.Logf("%s %.6f MHz %.1f MS/s: %d bytes, %d levels, %.3f%% rail samples", device.ID, float64(frequency)/1e6, float64(rate)/1e6, len(data), levels, 100*float64(clips)/float64(len(data)))
		}
	}
}
