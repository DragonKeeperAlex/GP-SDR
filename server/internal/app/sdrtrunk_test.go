package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sdrTrunkTestProfile() ScanProfile {
	return ScanProfile{ID: "p25-test", P25Systems: []P25SystemConfig{{
		ID: "system", Name: "County & City P25", ControlChannelsHz: []float64{851_012_500, 851_262_500}, Enabled: true,
		Talkgroups: []TalkgroupDefinition{{ID: 1201, Name: "Dispatch <East>", Mode: "D", Enabled: true},
			{ID: 1202, Name: "Encrypted", Mode: "DE", Encrypted: true, Enabled: true}},
	}}}
}

func TestBuildSDRTrunkPlaylistCreatesP25ControlAndEventLogs(t *testing.T) {
	data, err := BuildSDRTrunkPlaylist(sdrTrunkTestProfile(), "HackRF ONE 00000000-00000000-24B862DC-3140C5CB", nil)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{
		`decodeConfigP25Phase1`, `<frequency>851012500</frequency>`, `<frequency>851262500</frequency>`,
		`frequency_rotation_delay="1200"`,
		`preferred_tuner="HackRF ONE 00000000-00000000-24B862DC-3140C5CB"`,
		`<logger>DECODED_MESSAGE</logger>`, `<logger>TRAFFIC_CALL_EVENT</logger>`,
		`name="County &amp; City P25"`, `name="Dispatch &lt;East&gt;"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in generated playlist:\n%s", expected, text)
		}
	}
}

func TestInspectSDRTrunkEventsIgnoresPreviousSession(t *testing.T) {
	directory := t.TempDir()
	old := filepath.Join(directory, "old_decoded_messages.log")
	if err := os.WriteFile(old, []byte("20260820 011029,PASSED,NAC:501/x1F5 TSBK1 NET_STATUS_BCAST\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-time.Minute)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if locked, _ := inspectSDRTrunkEvents(directory, time.Now()); locked {
		t.Fatal("a prior session must not make the current P25 session appear locked")
	}
	fresh := filepath.Join(directory, "fresh_decoded_messages.log")
	if err := os.WriteFile(fresh, []byte("20260820 030134,PASSED,NAC:501/x1F5 TSBK1 NET_STATUS_BCAST\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if locked, _ := inspectSDRTrunkEvents(directory, time.Now().Add(-time.Second)); !locked {
		t.Fatal("fresh P25 framing should report a current control-channel lock")
	}
}

func TestOptimizeHackRFP25SampleRateReducesCompactSiteLoad(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	input := `{"tunerConfigurations":[{"type":"hackRFTunerConfiguration","sampleRate":"RATE_10_0","amplifierEnabled":true},{"type":"e4KTunerConfiguration","sampleRate":"RATE_2_048MHZ"}]}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := optimizeHackRFP25SampleRate(root, sdrTrunkTestProfile()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"sampleRate": "RATE_5_0"`) || !strings.Contains(text, `"amplifierEnabled": true`) || !strings.Contains(text, `"sampleRate": "RATE_2_048MHZ"`) {
		t.Fatalf("unexpected optimized tuner configuration:\n%s", text)
	}
}

func TestOptimizeHackRFP25SampleRatePreservesWideSystem(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	input := `{"tunerConfigurations":[{"type":"hackRFTunerConfiguration","sampleRate":"RATE_10_0"}]}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := sdrTrunkTestProfile()
	profile.P25Systems[0].ControlChannelsHz = []float64{770_000_000, 780_000_000}
	if err := optimizeHackRFP25SampleRate(root, profile); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"sampleRate":"RATE_10_0"`) {
		t.Fatalf("wide system sample rate was unexpectedly changed: %s", data)
	}
}

func TestBuildSDRTrunkPlaylistMakesEncryptedAndMutedTalkgroupsDoNotMonitor(t *testing.T) {
	data, err := BuildSDRTrunkPlaylist(sdrTrunkTestProfile(), "", map[uint32]bool{1201: true})
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(data), `type="priority" priority="-1"`); count != 2 {
		t.Fatalf("expected both muted and encrypted talkgroups to be disabled, got %d:\n%s", count, data)
	}
}

func TestPreferredSDRTrunkHackRFNameUsesHardwareSerial(t *testing.T) {
	serial := "000000000000000024b862dc3140c5cb"
	name := preferredSDRTrunkTuner([]p25AssignedDevice{{Device: SDRDevice{Kind: "HackRF", Serial: &serial}}})
	if name != "HackRF ONE 00000000-00000000-24B862DC-3140C5CB" {
		t.Fatalf("unexpected preferred name %q", name)
	}
}

func TestMacJMBEPreferenceUsesBundledHelperBeforeJavaFallback(t *testing.T) {
	data, err := os.ReadFile("sdrtrunk.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	helper := strings.Index(text, `findTool("gpsdr-mac-prefs")`)
	java := strings.Index(text, `findTool("java")`)
	if helper < 0 || java < 0 || helper >= java {
		t.Fatal("macOS JMBE setup must prefer the bundled native helper over a system JDK fallback")
	}
}

func TestBundledMacP25PathsMatchApplicationResourceLayout(t *testing.T) {
	checks := map[string][]string{
		"sdrtrunk.go":  {`filepath.Join(base, "..", "sdrtrunk-"+runtime.GOARCH`},
		"installer.go": {`filepath.Join(base, "..", "jmbe-creator-"+runtime.GOARCH`},
	}
	for path, required := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range required {
			if !strings.Contains(string(data), text) {
				t.Fatalf("%s does not match the packaged macOS Resources layout", path)
			}
		}
	}
}

func TestSDRTrunkLauncherOptionsPreservePathsWithSpaces(t *testing.T) {
	quoted := quoteLauncherOption("-Duser.home=/Users/Test User/Library/Application Support/GP-SDR")
	if !strings.Contains(quoted, "Test User") || !strings.HasPrefix(quoted, `'`) || !strings.HasSuffix(quoted, `'`) {
		t.Fatalf("launcher option is not safely quoted: %s", quoted)
	}
}
