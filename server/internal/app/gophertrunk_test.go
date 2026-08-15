package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func p25TestProfile() ScanProfile {
	return ScanProfile{ID: "p25-test", Settings: SurveySettings{MaxRecordingDays: 7}, P25Systems: []P25SystemConfig{{
		ID: "system", Name: "County P25", ControlChannelsHz: []float64{851_012_500, 851_262_500}, Enabled: true,
		Talkgroups: []TalkgroupDefinition{{ID: 101, Name: "Dispatch", Mode: "D", Enabled: true},
			{ID: 102, Name: "Encrypted", Mode: "DE", Enabled: true, Encrypted: true}},
	}}}
}

func TestGopherTrunkSingleHackRFUsesWidebandVoiceTaps(t *testing.T) {
	serial := "hackrf-serial"
	directory := t.TempDir()
	configuration, err := BuildGopherTrunkConfiguration(p25TestProfile(), []p25AssignedDevice{{
		Device: SDRDevice{Kind: "HackRF", Serial: &serial}, Role: "wideband",
	}}, directory, 8079, 8080)
	if err != nil {
		t.Fatal(err)
	}
	text := string(configuration)
	for _, expected := range []string{"role: wideband", "sample_rate: 8000000", "gain: \"0\"", "voice_taps: 6", "signalling_taps: 4", "protocol: p25"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated config is missing %q:\n%s", expected, text)
		}
	}
	csvData, err := os.ReadFile(filepath.Join(directory, "talkgroups-01.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(csvData), "101,Dispatch") || !strings.Contains(string(csvData), "102,Encrypted") {
		t.Fatalf("talkgroup CSV is incomplete: %s", csvData)
	}
}

func TestGopherTrunkMultipleReceiversKeepControlAndVoiceRoles(t *testing.T) {
	hackRFSerial, rtlSerial := "hackrf-serial", "00000001"
	configuration, err := BuildGopherTrunkConfiguration(p25TestProfile(), []p25AssignedDevice{
		{Device: SDRDevice{Kind: "HackRF", Serial: &hackRFSerial}, Role: "control"},
		{Device: SDRDevice{Kind: "RTL-SDR", Serial: &rtlSerial}, Role: "voice"},
	}, t.TempDir(), 8079, 8080)
	if err != nil {
		t.Fatal(err)
	}
	text := string(configuration)
	for _, expected := range []string{"serial: \"hackrf-serial\"\n      role: control", "serial: \"00000001\"\n      role: voice"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated config is missing %q:\n%s", expected, text)
		}
	}
}

func TestGopherTrunkRejectsWideControlSpan(t *testing.T) {
	profile := p25TestProfile()
	profile.P25Systems[0].ControlChannelsHz = []float64{851e6, 861e6}
	_, err := BuildGopherTrunkConfiguration(profile, []p25AssignedDevice{{
		Device: SDRDevice{ID: "hackrf-a", Kind: "HackRF", Serial: ptr("abc")}, Role: "wideband",
	}}, t.TempDir(), 18081, 18082)
	if err == nil || !strings.Contains(err.Error(), "wider") {
		t.Fatalf("expected a wideband span error, got %v", err)
	}
}
