package app

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHackRFCaptureCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a Unix executable shim")
	}
	directory := t.TempDir()
	helper := filepath.Join(directory, "hackrf_transfer")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	serial := "abc"
	command, err := BuildCaptureCommand(SDRDevice{Kind: "HackRF", Serial: &serial}, CaptureSpec{CenterFrequencyHz: 462_600_000, SampleRateHz: 8_000_000, GainDB: 21})
	if err != nil {
		t.Fatal(err)
	}
	if command.Format != ComplexSigned8 {
		t.Fatalf("wrong format: %s", command.Format)
	}
	want := []string{"-r", "-", "-f", "462600000", "-s", "8000000", "-d", "abc", "-g", "20"}
	if len(command.Arguments) != len(want) {
		t.Fatalf("arguments: %#v", command.Arguments)
	}
	for i := range want {
		if command.Arguments[i] != want[i] {
			t.Fatalf("arguments: %#v", command.Arguments)
		}
	}
}

func TestHackRFAdvancedControls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a Unix executable shim")
	}
	directory := t.TempDir()
	helper := filepath.Join(directory, "hackrf_transfer")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	command, err := BuildCaptureCommand(SDRDevice{Kind: "HackRF"}, CaptureSpec{CenterFrequencyHz: 462_600_000, SampleRateHz: 8_000_000, LNAGainDB: 24, VGAGainDB: 22, AmpEnabled: true, AntennaPower: true, PPMCorrection: -3})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(command.Arguments, " ")
	for _, expected := range []string{"-l 24", "-g 22", "-a 1", "-p 1", "-C -3"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}
}

func TestRTLCaptureCommandCanForceConservativeManualGain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a Unix executable shim")
	}
	directory := t.TempDir()
	helper := filepath.Join(directory, "rtl_sdr")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	command, err := BuildCaptureCommand(SDRDevice{Kind: "RTL-SDR", ID: "rtlsdr-0"}, CaptureSpec{
		CenterFrequencyHz: 101_800_000, SampleRateHz: 2_400_000, GainDB: 1.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if command.Format != ComplexUnsigned8 {
		t.Fatalf("wrong format: %s", command.Format)
	}
	joined := strings.Join(command.Arguments, " ")
	if !strings.Contains(joined, "-g 1.5") {
		t.Fatalf("conservative RTL gain was not forwarded: %s", joined)
	}
}

func TestCaptureCommandRejectsUnsupportedDevice(t *testing.T) {
	_, err := BuildCaptureCommand(SDRDevice{Kind: "Other"}, CaptureSpec{CenterFrequencyHz: 100e6, SampleRateHz: 1e6})
	if err == nil {
		t.Fatal("expected unsupported device error")
	}
}

func TestPersistentRTLSessionOnConnectedHardware(t *testing.T) {
	if os.Getenv("GP_SDR_HARDWARE_TEST") != "1" {
		t.Skip("set GP_SDR_HARDWARE_TEST=1 with one idle RTL-SDR connected")
	}
	shutdownLocalRTLSessions()
	t.Cleanup(shutdownLocalRTLSessions)
	device := SDRDevice{ID: "rtlsdr-0", Kind: "RTL-SDR"}
	pids := make([]int, 0, 2)
	for _, frequency := range []int64{100_100_000, 162_550_000} {
		stream, err := StartIQStream(device, CaptureSpec{CenterFrequencyHz: frequency, SampleRateHz: 2_400_000, GainDB: 20})
		if err != nil {
			t.Fatal(err)
		}
		data := make([]byte, 64*1024)
		if _, err := io.ReadFull(stream.Reader, data); err != nil {
			t.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
		localRTLSessions.Lock()
		session := localRTLSessions.items[device.ID]
		localRTLSessions.Unlock()
		session.stateMu.Lock()
		if session.cmd == nil || session.cmd.Process == nil {
			session.stateMu.Unlock()
			t.Fatal("persistent rtl_tcp helper stopped between captures")
		}
		pids = append(pids, session.cmd.Process.Pid)
		session.stateMu.Unlock()
	}
	if pids[0] != pids[1] {
		t.Fatalf("RTL-SDR was reopened between captures: helper PIDs %v", pids)
	}
}
