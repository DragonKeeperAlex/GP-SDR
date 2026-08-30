package app

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testPCM16WAV(t *testing.T, path string) {
	t.Helper()
	const rate, frames = 8000, 800
	data := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(i%1000))
	}
	buf := make([]byte, 44+len(data))
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(buf)-8))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	binary.LittleEndian.PutUint16(buf[22:], 1)
	binary.LittleEndian.PutUint32(buf[24:], rate)
	binary.LittleEndian.PutUint32(buf[28:], rate*2)
	binary.LittleEndian.PutUint16(buf[32:], 2)
	binary.LittleEndian.PutUint16(buf[34:], 16)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(len(data)))
	copy(buf[44:], data)
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestModulateTransmitAudioProducesBoundedIQ(t *testing.T) {
	audio := []int16{0, 12000, -12000, 0}
	for _, mode := range []string{"am", "nfm", "wfm"} {
		iq, err := modulateTransmitAudio(audio, 8000, 2_000_000, 2000, mode)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if len(iq) != 4000 {
			t.Fatalf("%s: got %d IQ bytes, want 4000", mode, len(iq))
		}
		for _, sample := range iq {
			if sample > 255 {
				t.Fatalf("%s: sample overflow %d", mode, sample)
			}
		}
	}
}

func TestTransmitDryRunDoesNotRequireHardwareOutput(t *testing.T) {
	root := t.TempDir()
	wav := filepath.Join(root, "audio.wav")
	testPCM16WAV(t, wav)
	runtimeState, err := NewRuntime(root, "", false)
	if err != nil {
		t.Fatal(err)
	}
	// A dry run still validates WAV parsing and modulation, but it must not
	// invoke hackrf_transfer or require a real radio to be assigned.
	serial := "000000000000000024b862dc3140c5cb"
	runtimeState.devices = []SDRDevice{{ID: "hackrf-test", Kind: "HackRF", Serial: &serial, Connected: true, Available: true}}
	status, err := runtimeState.Transmit(TransmitRequest{DeviceID: "hackrf-test", FrequencyHz: 100.1e6, Mode: "nfm", AudioPath: wav, DurationSecond: .1, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "running" {
		t.Fatalf("unexpected initial transmit state: %+v", status)
	}
	runtimeState.StopTransmit()
	time.Sleep(20 * time.Millisecond)
	if final := runtimeState.TransmitStatus(); final.State != "stopped" && final.State != "complete" {
		t.Fatalf("unexpected final transmit state: %+v", final)
	}
}
