package app

import (
	"os"
	"strings"
	"testing"
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
		`preferred_tuner="HackRF ONE 00000000-00000000-24B862DC-3140C5CB"`,
		`<logger>DECODED_MESSAGE</logger>`, `<logger>TRAFFIC_CALL_EVENT</logger>`,
		`name="County &amp; City P25"`, `name="Dispatch &lt;East&gt;"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in generated playlist:\n%s", expected, text)
		}
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
