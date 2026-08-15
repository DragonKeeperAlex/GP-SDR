package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveStartWithoutConnectedSDRReturnsImmediately(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	runtime.devices = []SDRDevice{{ID: "driver", Kind: "RTL-SDR", Available: true, Connected: false}}
	profile := builtInProfiles()[1]
	runtime.Profiles.profiles[profile.ID] = profile
	if err := runtime.Start(profile.ID); err == nil || !strings.Contains(err.Error(), "No SDR") {
		t.Fatalf("expected a useful no-SDR error, got %v", err)
	}
	if runtime.Status().Running {
		t.Fatal("runtime was left running after a failed start")
	}
}

func TestBuildOP25ConfigurationUsesDistinctReceiversAndSilencesEncryptedTalkgroups(t *testing.T) {
	profile := ScanProfile{SchemaVersion: 1, ID: "profile", Name: "P25", P25Systems: []P25SystemConfig{{
		ID: "system", Name: "Test system", ControlChannelsHz: []float64{773_843_750}, NAC: "4a6", WACN: "bee00", SystemID: "4a2", Enabled: true,
		Talkgroups: []TalkgroupDefinition{{ID: 100, Name: "Dispatch", Enabled: true}, {ID: 200, Name: "Encrypted", Enabled: true, Encrypted: true}},
	}}}
	directory := t.TempDir()
	data, err := BuildOP25Configuration(profile, []SDRDevice{{ID: "rtlsdr-0", Kind: "RTL-SDR"}, {ID: "rtlsdr-1", Kind: "RTL-SDR"}}, directory)
	if err != nil {
		t.Fatal(err)
	}
	var configuration op25Configuration
	if err := json.Unmarshal(data, &configuration); err != nil {
		t.Fatal(err)
	}
	if len(configuration.Devices) != 2 || configuration.Devices[0].Name == configuration.Devices[1].Name {
		t.Fatalf("receivers were not distinct: %#v", configuration.Devices)
	}
	if len(configuration.Channels) != 2 || configuration.Trunking.Channels[0].CryptBehavior != 2 {
		t.Fatalf("unexpected OP25 configuration: %#v", configuration)
	}
	whitelist, err := os.ReadFile(filepath.Join(directory, configuration.Channels[0].Whitelist))
	if err != nil {
		t.Fatal(err)
	}
	if string(whitelist) != "100\n" {
		t.Fatalf("encrypted talkgroup was not filtered: %q", whitelist)
	}
}

func TestRadioReferenceSOAPEscapesCredentials(t *testing.T) {
	client := &radioReferenceClient{username: "a&b", password: "<secret>", appKey: "key", endpoint: radioReferenceEndpoint}
	body := soapEnvelope("getZipcodeInfo", []soapValue{{"zipcode", "94107", "xsd:int"}}, client)
	for _, expected := range []string{"a&amp;b", "&lt;secret&gt;", "<zipcode xsi:type=\"xsd:int\">94107</zipcode>"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("SOAP body missing %q: %s", expected, body)
		}
	}
}

func TestHaversineRange(t *testing.T) {
	distance := haversineMiles(37.7749, -122.4194, 37.8044, -122.2712)
	if distance < 7 || distance > 9 {
		t.Fatalf("unexpected SF to Oakland distance: %.2f", distance)
	}
}

func TestSurveyTargetsSkipDigitalChannels(t *testing.T) {
	profile := ScanProfile{Channels: []ChannelDefinition{
		{FrequencyHz: 100e6, BandwidthHz: 12_500, Mode: "nfm", Enabled: true},
		{FrequencyHz: 101e6, BandwidthHz: 12_500, Mode: "p25", Enabled: true},
	}}
	targets := surveyTargets(profile)
	if len(targets) != 1 || targets[0].FrequencyHz != 100e6 {
		t.Fatalf("unexpected analog targets: %#v", targets)
	}
}
