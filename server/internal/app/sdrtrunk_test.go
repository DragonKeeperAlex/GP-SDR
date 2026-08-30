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
		`<id type="record"/>`, `<id type="talkgroupRange" protocol="APCO25" min="1" max="65535"/>`,
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

func TestInspectSDRTrunkControlFrequencyUsesCurrentBandPlan(t *testing.T) {
	directory := t.TempDir()
	logPath := filepath.Join(directory, "20260820_224128.539_0_Hz_EBRCS_decoded_messages.log")
	contents := strings.Join([]string{
		"20260820 224130,PASSED,NAC:501 TSBK1 NET_STATUS_BCAST WACN:BEE00 SYSTEM:2B1 CHAN:1-1992",
		"20260820 224129,PASSED,NAC:501 TSBK1 IDEN_UPDATE ID:1 BW:12500 SPACING:6250 BASE:762006250",
	}, "\n")
	if err := os.WriteFile(logPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := inspectSDRTrunkControlFrequency(directory, time.Now().Add(-time.Second)); got != 774_456_250 {
		t.Fatalf("unexpected control frequency %.0f", got)
	}
	if got := inspectSDRTrunkControlFrequency(directory, time.Now().Add(time.Second)); got != 0 {
		t.Fatalf("prior-session log must not report a current control frequency, got %.0f", got)
	}
}

func hackRFP25Assignment() []p25AssignedDevice {
	return []p25AssignedDevice{{Device: SDRDevice{Kind: "HackRF"}}}
}

func rtlSDRP25Assignment() []p25AssignedDevice {
	return []p25AssignedDevice{{Device: SDRDevice{Kind: "RTL-SDR"}}}
}

func TestOptimizeP25SampleRatesConfiguresAssignedHackRFOnly(t *testing.T) {
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
	if err := optimizeP25SampleRates(root, sdrTrunkTestProfile(), hackRFP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"sampleRate": "RATE_10_0"`) || !strings.Contains(text, `"amplifierEnabled": false`) || !strings.Contains(text, `"sampleRate": "RATE_2_048MHZ"`) {
		t.Fatalf("unexpected optimized tuner configuration:\n%s", text)
	}
}

func TestOptimizeP25SampleRatesDisablesHackRFAmplifierAtSafeRate(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	input := `{"tunerConfigurations":[{"type":"hackRFTunerConfiguration","sampleRate":"RATE_5_0","amplifierEnabled":true}]}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := optimizeP25SampleRates(root, sdrTrunkTestProfile(), hackRFP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"amplifierEnabled": false`) {
		t.Fatalf("HackRF amplifier remained enabled: %s", data)
	}
}

func TestOptimizeP25SampleRatesHonorsExplicitHackRFWideRateAndFallback(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	if err := os.WriteFile(path, []byte(`{"tunerConfigurations":[{"type":"hackRFTunerConfiguration","sampleRate":"RATE_5_0"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := sdrTrunkTestProfile()
	profile.Settings.P25SampleRateHz = 20_000_000
	if err := optimizeP25SampleRates(root, profile, hackRFP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"RATE_20_0"`) {
		t.Fatalf("explicit wide rate was not applied: %s", data)
	}
	if err := optimizeP25SampleRates(root, profile, hackRFP25Assignment(), true); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), `"RATE_5_0"`) {
		t.Fatalf("transport fallback was not applied: %s", data)
	}
}

func TestOptimizeP25SampleRatesPreservesWideHackRFSystem(t *testing.T) {
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
	if err := optimizeP25SampleRates(root, profile, hackRFP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"sampleRate": "RATE_10_0"`) {
		t.Fatalf("wide system sample rate was unexpectedly changed: %s", data)
	}
}

func TestOptimizeP25SampleRatesUsesRTLSDRSafeDefault(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	input := `{"tunerConfigurations":[{"type":"e4KTunerConfiguration","sampleRate":"RATE_2_048MHZ"},{"type":"hackRFTunerConfiguration","sampleRate":"RATE_10_0","amplifierEnabled":true}]}`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := optimizeP25SampleRates(root, sdrTrunkTestProfile(), rtlSDRP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"sampleRate": "RATE_2_400MHZ"`) {
		t.Fatalf("RTL-SDR safe default was not applied: %s", text)
	}
	if !strings.Contains(text, `"sampleRate": "RATE_10_0"`) || !strings.Contains(text, `"amplifierEnabled": true`) {
		t.Fatalf("unassigned HackRF settings were unexpectedly changed: %s", text)
	}
}

func TestOptimizeP25SampleRatesHonorsExplicitRTLSDRRate(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "tuner_configuration.json")
	if err := os.WriteFile(path, []byte(`{"tunerConfigurations":[{"type":"r820TTunerConfiguration","sampleRate":"RATE_2_400MHZ"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := sdrTrunkTestProfile()
	profile.Settings.P25SampleRateHz = 1_024_000
	if err := optimizeP25SampleRates(root, profile, rtlSDRP25Assignment(), false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"RATE_1_024MHZ"`) {
		t.Fatalf("explicit RTL-SDR rate was not applied: %s", data)
	}
}

func TestEffectiveP25CaptureRateMatchesAssignedReceiver(t *testing.T) {
	profile := sdrTrunkTestProfile()
	if got := effectiveP25CaptureRate(profile, rtlSDRP25Assignment(), false); got != 2_400_000 {
		t.Fatalf("RTL-SDR auto rate = %d, want 2400000", got)
	}
	profile.Settings.P25SampleRateHz = 2_048_000
	if got := effectiveP25CaptureRate(profile, rtlSDRP25Assignment(), false); got != 2_048_000 {
		t.Fatalf("RTL-SDR explicit rate = %d, want 2048000", got)
	}
	profile.Settings.P25SampleRateHz = 0
	if got := effectiveP25CaptureRate(profile, hackRFP25Assignment(), false); got != 10_000_000 {
		t.Fatalf("HackRF auto rate = %d, want 10000000", got)
	}
	if got := effectiveP25CaptureRate(profile, hackRFP25Assignment(), true); got != 5_000_000 {
		t.Fatalf("HackRF fallback rate = %d, want 5000000", got)
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
	name := preferredSDRTrunkTuner([]p25AssignedDevice{{Device: SDRDevice{Kind: "HackRF", Serial: &serial}}}, t.TempDir())
	if name != "HackRF ONE 00000000-00000000-24B862DC-3140C5CB" {
		t.Fatalf("unexpected preferred name %q", name)
	}
}

func TestPreferredSDRTrunkHackRFUsesConfiguredUSBID(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	configuration := `{"tunerConfigurations":[{"type":"hackRFTunerConfiguration","uniqueID":"HackRF USB Bus:2 Port:1.1"},{"type":"hackRFTunerConfiguration","uniqueID":"HackRF USB Bus:2 Port:1.2"}]}`
	if err := os.WriteFile(filepath.Join(directory, "tuner_configuration.json"), []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	serial := "000000000000000024b862dc3140c5cb"
	name := preferredSDRTrunkTuner([]p25AssignedDevice{{Device: SDRDevice{Kind: "HackRF", Serial: &serial}}}, root)
	if name != "" {
		t.Fatalf("multiple configured HackRFs must not receive an invented preference, got %q", name)
	}
}

func TestPreferredSDRTrunkRTLTunerUsesConfiguredUniqueID(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "configuration")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	configuration := `{"tunerConfigurations":[{"type":"e4KTunerConfiguration","uniqueID":"RTL-2832 USB Bus:2 Port:1","sampleRate":"RATE_2_400MHZ"}]}`
	if err := os.WriteFile(filepath.Join(directory, "tuner_configuration.json"), []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	name := preferredSDRTrunkTuner(rtlSDRP25Assignment(), root)
	if name != "RTL-2832 USB Bus:2 Port:1" {
		t.Fatalf("unexpected RTL-SDR preferred name %q", name)
	}
}

func TestP25AssignmentsSkipUnavailableSelfTestFailure(t *testing.T) {
	badID, goodID := "hackrf-bad", "hackrf-good"
	plan := []ReceiverPlanItem{{DeviceID: &badID, Role: "control"}, {DeviceID: &goodID, Role: "voice"}}
	devices := []SDRDevice{{ID: badID, Kind: "HackRF", Connected: true, Available: false}, {ID: goodID, Kind: "HackRF", Connected: true, Available: true}}
	assigned := p25DeviceAssignments(plan, devices)
	if len(assigned) != 1 || assigned[0].Device.ID != goodID {
		t.Fatalf("unavailable HackRF was assigned to P25: %#v", assigned)
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
