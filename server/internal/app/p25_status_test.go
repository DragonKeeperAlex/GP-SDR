package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestP25IdleStatusUsesDefaultOP25Engine(t *testing.T) {
	t.Setenv("GPSDR_P25_ENGINE", "")
	manager := &OP25Manager{}
	status := manager.Status()
	if status.Engine != "OP25" {
		t.Fatalf("default idle P25 engine = %q, want OP25", status.Engine)
	}
}

func TestOP25StatusReportsAppliedCaptureRate(t *testing.T) {
	done := make(chan struct{})
	manager := &OP25Manager{command: &exec.Cmd{Path: "op25"}, done: done, captureRateHz: 2_400_000}
	status := manager.op25Status()
	if status.State != "running" || status.CaptureRateHz != 2_400_000 {
		t.Fatalf("P25 status did not report the applied OP25 rate: %#v", status)
	}
}

func TestEffectiveOP25DeviceRateDoesNotReuseIncompatibleSharedProfileRate(t *testing.T) {
	profile := ScanProfile{Settings: SurveySettings{P25SampleRateHz: 2_400_000}}
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "HackRF"}); got != 8_000_000 {
		t.Fatalf("HackRF shared 2.4 MS/s profile rate = %d, want 8000000", got)
	}
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "RTL-SDR"}); got != 2_400_000 {
		t.Fatalf("RTL-SDR shared profile rate = %d, want 2400000", got)
	}
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "PlutoSDR"}); got != 2_400_000 {
		t.Fatalf("Pluto shared profile rate = %d, want 2400000", got)
	}
	profile.Settings.P25SampleRateHz = 20_000_000
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "HackRF"}); got != 20_000_000 {
		t.Fatalf("HackRF explicit rate = %d, want 20000000", got)
	}
}

func TestOP25ExitDiagnosticKeepsUsefulTailWithoutFullLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "op25.log")
	if err := os.WriteFile(path, []byte("Starting OP25 (pid = 7)\nfirst detail\nreceiver disconnected\nlast useful failure\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	message := op25ExitDiagnostic(path, errors.New("exit status 1"))
	for _, want := range []string{"exit status 1", "receiver disconnected", "last useful failure"} {
		if !strings.Contains(message, want) {
			t.Fatalf("diagnostic %q does not contain %q", message, want)
		}
	}
	if strings.Contains(message, "Starting OP25") {
		t.Fatalf("diagnostic included boilerplate: %q", message)
	}
}

func TestP25FrontendStateMatchesConservativeHackRFDefaults(t *testing.T) {
	id := "hackrf-test"
	profile := ScanProfile{}
	plan := []ReceiverPlanItem{{DeviceID: &id}}
	devices := []SDRDevice{{ID: id, Kind: "HackRF", Connected: true, Available: true}}
	if got, want := p25FrontendState(&profile, plan, devices), "LNA 8 dB · VGA 0 dB · RF amp off"; got != want {
		t.Fatalf("frontend state = %q, want %q", got, want)
	}
}

func TestP25FrontendStateReportsAppliedHackRFGains(t *testing.T) {
	id := "hackrf-test"
	lna, vga := 8, 0
	profile := ScanProfile{Settings: SurveySettings{P25AmpMode: "on", P25LNAGainDB: &lna, P25VGAGainDB: &vga}}
	plan := []ReceiverPlanItem{{DeviceID: &id}}
	devices := []SDRDevice{{ID: id, Kind: "HackRF", Connected: true, Available: true}}
	if got, want := p25FrontendState(&profile, plan, devices), "LNA 8 dB · VGA 0 dB · RF amp on"; got != want {
		t.Fatalf("frontend state = %q, want %q", got, want)
	}
}

// A decoder can fail before the first status poll (for example a missing USB
// receiver). It must release audio and retain its useful startup diagnostic.
func TestOP25ImmediateStartupFailureClearsAudioSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper fixture")
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "multi_rx.py"), []byte("#!/bin/sh\necho 'receiver startup failed' >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GPSDR_HELPERS", directory)
	t.Setenv("GPSDR_P25_TAGGED_AUDIO", "1")
	deviceID := "fixture-radio"
	profile := ScanProfile{ID: "startup-fixture", P25Systems: []P25SystemConfig{{ID: "test", Name: "Test", Enabled: true, ControlChannelsHz: []float64{774456250}}}}
	manager := &OP25Manager{audioHub: NewAudioHub()}
	defer manager.stopProcess()
	if err := manager.startOP25(profile, []ReceiverPlanItem{{DeviceID: &deviceID}}, []SDRDevice{{ID: deviceID, Kind: "RTL-SDR", Connected: true, Available: true}}, directory); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	done := manager.done
	manager.mu.Unlock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("startup fixture did not exit")
	}
	manager.mu.Lock()
	session, tagged, sockets, note := manager.audioSession, manager.taggedTalkgroupAudio, len(manager.audioSockets), manager.lastError
	manager.mu.Unlock()
	if session != "" || tagged || sockets != 0 {
		t.Fatalf("failed child retained audio state: session=%q tagged=%v sockets=%d", session, tagged, sockets)
	}
	if note == nil || !strings.Contains(*note, "receiver startup failed") || !strings.Contains(*note, "exit status 7") {
		t.Fatalf("startup diagnostic lost: %v", note)
	}
	status := manager.op25Status()
	if status.State != "error" || !strings.Contains(status.Note, "receiver startup failed") {
		t.Fatalf("startup failure not visible: %+v", status)
	}
}
