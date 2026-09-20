package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "HackRF"}); got != 5_000_000 {
		t.Fatalf("HackRF shared 2.4 MS/s profile rate = %d, want 5000000", got)
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
