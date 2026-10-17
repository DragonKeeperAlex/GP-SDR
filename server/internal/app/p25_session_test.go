package app

import (
	"encoding/json"
	"testing"
	"time"
)

func sessionUpdates(t *testing.T, value string) []map[string]json.RawMessage {
	t.Helper()
	var updates []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &updates); err != nil {
		t.Fatal(err)
	}
	return updates
}

func TestP25SessionAttributesControlToOnlyItsReceiver(t *testing.T) {
	now := time.Unix(2000, 0)
	configured := []P25ReceiverStatus{
		{DeviceID: "hackrf", StreamIndex: 0, System: "site", ConfiguredRateHz: 8_000_000},
		{DeviceID: "rtl", StreamIndex: 1, System: "site", ConfiguredRateHz: 2_400_000},
		{DeviceID: "pluto", StreamIndex: 2, System: "site", ConfiguredRateHz: 8_000_000},
	}
	updates := sessionUpdates(t, `[
	 {"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":1999.5}},
	 {"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":774143750},"1":{"system":"site","tag":"Idle","freq":800000000},"2":{"system":"site","tag":"Dispatch","freq":772356250,"tgid":7955,"tdma":1,"encrypted":0}}
	]`)
	status := P25Status{Reception: "searching", CaptureRateHz: 8_000_000}
	summarizeOP25Session(&status, configured, updates, now.Add(-time.Minute), now)
	if status.Reception != "locked" || status.CaptureRateHz != 0 || status.TelemetryState != "available" {
		t.Fatalf("bad summary: %#v", status)
	}
	if status.Receivers[0].State != "control_locked" || status.Receivers[0].LastControlAt == nil {
		t.Fatalf("missing control evidence: %#v", status.Receivers[0])
	}
	if status.Receivers[1].State != "idle" || status.Receivers[1].LastControlAt != nil {
		t.Fatalf("idle receiver inherited lock: %#v", status.Receivers[1])
	}
	voice := status.Receivers[2]
	if voice.State != "following" || voice.TDMASlot == nil || *voice.TDMASlot != 1 || voice.LastControlAt != nil {
		t.Fatalf("bad voice attribution: %#v", voice)
	}
	if configured[0].State != "" {
		t.Fatal("projection mutated immutable session config")
	}
}

func TestP25SessionRejectsStaleForeignAndMissingEvidence(t *testing.T) {
	now := time.Unix(2000, 0)
	configured := []P25ReceiverStatus{{StreamIndex: 0, System: "site"}}
	for _, test := range []struct{ name, raw, state string }{
		{"old", `[{"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":1980}},{"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":774143750}}]`, "searching"},
		{"previous_session", `[{"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":1994}},{"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":774143750}}]`, "searching"},
		{"future", `[{"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":2001}},{"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":774143750}}]`, "searching"},
		{"wrong_frequency", `[{"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":1999}},{"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":771000000}}]`, "searching"},
		{"foreign_system", `[{"json_type":"channel_update","0":{"system":"other","tag":"Control Channel","freq":774143750}}]`, "telemetry_unavailable"},
		{"missing_channel", `[{"json_type":"channel_update","channels":[]}]`, "telemetry_unavailable"},
		{"held_group", `[{"json_type":"channel_update","0":{"system":"site","freq":774143750,"tgid":42,"hold_tgid":42}}]`, "unconfirmed"},
		{"encrypted", `[{"json_type":"channel_update","0":{"system":"site","freq":774143750,"tgid":42,"encrypted":1}}]`, "encrypted"},
		{"empty", `[]`, "telemetry_unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status := P25Status{Reception: "searching"}
			summarizeOP25Session(&status, configured, sessionUpdates(t, test.raw), now.Add(-5*time.Second), now)
			if status.Receivers[0].State != test.state || status.Reception == "locked" {
				t.Fatalf("unearned evidence: %#v", status)
			}
			if test.state == "telemetry_unavailable" && (status.TelemetryState != "unavailable" || status.Reception != "unknown") {
				t.Fatalf("missing telemetry presented as reception: %#v", status)
			}
		})
	}
}

func TestP25ReceiverContractUsesGeneratedConfiguration(t *testing.T) {
	profile := ScanProfile{P25Systems: []P25SystemConfig{{Name: "site", Enabled: true, ControlChannelsHz: []float64{774143750}}}, Settings: SurveySettings{P25SampleRateHz: 8_000_000}}
	devices := []SDRDevice{{ID: "hackrf", Name: "Good HackRF", Kind: "HackRF"}, {ID: "rtl", Name: "RTL", Kind: "RTL-SDR"}, {ID: "pluto", Name: "Pluto", Kind: "PlutoSDR", Transport: "network"}}
	data, err := BuildOP25Configuration(profile, devices, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var config op25Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	rows := op25ReceiverConfiguration(config, devices)
	if len(rows) != 3 {
		t.Fatalf("missing receivers: %#v", rows)
	}
	for index, row := range rows {
		if row.DeviceID != devices[index].ID || row.ConfiguredRateHz != config.Devices[index].Rate || row.ConfiguredGains != config.Devices[index].Gains || row.SettingsSource != "decoder_configuration" {
			t.Fatalf("profile/config mismatch: %#v", row)
		}
	}
	if rows[1].ConfiguredRateHz != 2_400_000 || rows[0].ConfiguredRateHz != 8_000_000 || rows[2].Transport != "network" {
		t.Fatalf("mixed receiver contract lost: %#v", rows)
	}
}

func TestP25PreferredControlReceiverPrecedesVoiceReceivers(t *testing.T) {
	devices := []SDRDevice{{ID: "hackrf", Connected: true, Available: true}, {ID: "rtl", Connected: true, Available: true}, {ID: "pluto", Connected: true, Available: true}}
	plan := []ReceiverPlanItem{{DeviceID: ptr("hackrf"), Role: "voice-1"}, {DeviceID: ptr("rtl"), Role: "voice-2"}, {DeviceID: ptr("pluto"), Role: "control"}}
	ordered := assignedDevices(plan, devices)
	if len(ordered) != 3 || ordered[0].ID != "pluto" || ordered[1].ID != "hackrf" || ordered[2].ID != "rtl" {
		t.Fatalf("control preference lost: %#v", ordered)
	}
	if plan[0].Role != "voice-1" || devices[0].ID != "hackrf" {
		t.Fatal("ordering mutated caller state")
	}
	devices[2].Available = false
	ordered = assignedDevices(plan, devices)
	if len(ordered) != 2 || ordered[0].ID != "hackrf" {
		t.Fatalf("unavailable preferred receiver included: %#v", ordered)
	}
}

func TestP25DiagnosticsKeepsStaleSystemEvidenceSeparateFromReceiver(t *testing.T) {
	now := time.Unix(2000, 0)
	updates := sessionUpdates(t, `[{"json_type":"trunk_update","0":{"system":"site","network_active":1,"rxchan":774143750,"last_tsbk":1980,"frequency_data":{"774143750":{"type":"control","counter":471},"772356250":{"type":"voice","counter":20},"772656250":{"type":"voice","counter":38}}}},{"json_type":"channel_update","0":{"system":"site","tag":"Control Channel","freq":774143750,"error":26},"1":{"system":"site","tag":"Idle","freq":772656250,"error":null}}]`)
	status := P25Status{Reception: "searching"}
	summarizeOP25Session(&status, []P25ReceiverStatus{{StreamIndex: 0, System: "site"}, {StreamIndex: 1, System: "site"}}, updates, now.Add(-time.Minute), now)
	if len(status.Systems) != 1 || status.Systems[0].LastControlAt == nil || *status.Systems[0].ControlMessages != 471 || *status.Systems[0].VoiceUpdates != 58 {
		t.Fatalf("missing system history: %#v", status.Systems)
	}
	if status.Receivers[0].LastControlAt != nil || status.Reception == "locked" || status.Receivers[0].FrequencyErrorHz == nil || *status.Receivers[0].FrequencyErrorHz != 26 || status.Receivers[1].FrequencyErrorHz != nil {
		t.Fatalf("wrong receiver evidence: %#v", status.Receivers)
	}
	for _, last := range []string{"1900", "2001"} {
		rows := projectOP25Systems(sessionUpdates(t, `[{"json_type":"trunk_update","0":{"system":"site","last_tsbk":`+last+`}}]`), now.Add(-time.Minute), now)
		if rows[0].LastControlAt != nil || rows[0].ControlMessages != nil || rows[0].VoiceUpdates != nil {
			t.Fatal("old/future or missing evidence was promoted")
		}
	}
}
