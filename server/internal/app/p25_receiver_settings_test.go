package app

import (
	"encoding/json"
	"testing"
)

func TestOP25ReceiverOverridesStayWithTheirPhysicalDevice(t *testing.T) {
	profile := sdrTrunkTestProfile()
	profile.SchemaVersion = 1
	profile.Name = "Receiver settings"
	profile.Settings.P25SampleRateHz = 2_400_000
	profile.Settings.P25Receivers = map[string]P25ReceiverSettings{
		"pluto":  {SampleRateHz: 2_000_000, GainDB: ptr(40.0), PPMCorrection: ptr(-2)},
		"hackrf": {SampleRateHz: 8_000_000, LNAGainDB: ptr(16), VGAGainDB: ptr(8), AmpEnabled: ptr(false)},
		"rtl":    {GainDB: ptr(0.0), PPMCorrection: ptr(3)},
	}
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Save(profile); err != nil {
		t.Fatal(err)
	}
	saved, ok := store.Get(profile.ID)
	if !ok {
		t.Fatal("saved profile missing")
	}
	// Reordering the receivers cannot move settings to a different radio.
	devices := []SDRDevice{{ID: "rtl", Kind: "RTL-SDR"}, {ID: "pluto", Kind: "PlutoSDR"}, {ID: "hackrf", Kind: "HackRF"}}
	raw, err := buildOP25ConfigurationWithPlan(saved, devices, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var config op25Configuration
	if err = json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	want := []op25Device{{Rate: 2_400_000, Gains: "LNA:0", PPM: 3}, {Rate: 2_000_000, Gains: "PGA:40", PPM: -2}, {Rate: 8_000_000, Gains: "LNA:16,VGA:8,AMP:0"}}
	for i, w := range want {
		got := config.Devices[i]
		if got.Rate != w.Rate || got.Gains != w.Gains || got.PPM != w.PPM {
			t.Fatalf("device %s got %+v; want %+v", devices[i].ID, got, w)
		}
	}
	if saved.Settings.P25SampleRateHz != 2_400_000 {
		t.Fatal("shared default was changed")
	}
}

func TestP25ReceiverOverridesRejectUnsupportedControls(t *testing.T) {
	for _, test := range []struct {
		kind     string
		settings P25ReceiverSettings
	}{
		{"HackRF", P25ReceiverSettings{SampleRateHz: 5_000_000}},
		{"RTL-SDR", P25ReceiverSettings{SampleRateHz: 8_000_000}},
		{"RTL-SDR", P25ReceiverSettings{LNAGainDB: ptr(16)}},
		{"PlutoSDR", P25ReceiverSettings{VGAGainDB: ptr(8)}},
		{"HackRF", P25ReceiverSettings{LNAGainDB: ptr(7)}},
		{"RTL-SDR", P25ReceiverSettings{GainDB: ptr(50.0)}},
		{"PlutoSDR", P25ReceiverSettings{PPMCorrection: ptr(201)}},
	} {
		config := op25Device{Rate: 2_400_000, Gains: "LNA:8,VGA:0,AMP:0"}
		if err := applyP25ReceiverSettings(&config, SDRDevice{Kind: test.kind}, test.settings); err == nil {
			t.Fatalf("accepted %s %+v", test.kind, test.settings)
		}
	}
}

func TestP25ReceiverRTLDiscreteRate(t *testing.T) {
	config := op25Device{}
	if err := applyP25ReceiverSettings(&config, SDRDevice{Kind: "RTL-SDR"}, P25ReceiverSettings{SampleRateHz: 1_024_000}); err != nil {
		t.Fatal(err)
	}
	if config.Rate != 1_024_000 {
		t.Fatal("RTL discrete rate lost")
	}
}
