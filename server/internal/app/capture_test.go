package app

import (
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

func TestCaptureCommandRejectsUnsupportedDevice(t *testing.T) {
	_, err := BuildCaptureCommand(SDRDevice{Kind: "Other"}, CaptureSpec{CenterFrequencyHz: 100e6, SampleRateHz: 1e6})
	if err == nil {
		t.Fatal("expected unsupported device error")
	}
}
