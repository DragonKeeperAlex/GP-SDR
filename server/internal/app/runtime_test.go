package app

import "testing"

func TestReceiverPlanUsesDistinctDevices(t *testing.T) {
	runtime := &Runtime{devices: []SDRDevice{{ID: "one", Name: "Control radio", Connected: true, Available: true}, {ID: "two", Name: "Voice radio", Connected: true, Available: true}}}
	profile := ScanProfile{DeviceAssignments: []DeviceAssignment{{ID: "a", Role: "control"}, {ID: "b", Role: "voice"}}}
	plan := runtime.buildPlan(profile)
	if len(plan) != 2 || plan[0].DeviceID == nil || plan[1].DeviceID == nil {
		t.Fatalf("assignments missing: %#v", plan)
	}
	if *plan[0].DeviceID == *plan[1].DeviceID {
		t.Fatalf("roles share one device: %#v", plan)
	}
}

func TestPinnedReceiverRole(t *testing.T) {
	id := "voice-radio"
	runtime := &Runtime{devices: []SDRDevice{{ID: id, Name: "Pinned", Connected: true, Available: true}}}
	plan := runtime.buildPlan(ScanProfile{DeviceAssignments: []DeviceAssignment{{ID: "a", Role: "voice", DeviceID: &id}}})
	if plan[0].State != "assigned" || plan[0].DeviceID == nil || *plan[0].DeviceID != id {
		t.Fatalf("pin not honored: %#v", plan)
	}
}

func TestTunerValidatesInputBeforeOpeningHardware(t *testing.T) {
	runtime := &Runtime{}
	for _, request := range []TunerRequest{
		{FrequencyHz: 462_675_000, Mode: "digital"},
		{FrequencyHz: -1, Mode: "nfm"},
		{FrequencyHz: 462_675_000, Mode: "nfm", GainDB: 80},
	} {
		if err := runtime.Tune(request); err == nil {
			t.Fatalf("expected request to be rejected: %#v", request)
		}
	}
}

func TestLockedCenterMovesSoftwareVFOWithoutRestartingReceiver(t *testing.T) {
	updates := make(chan TunerRequest, 1)
	activeRequest := TunerRequest{DeviceID: "hackrf", FrequencyHz: 450e6, Mode: "nfm", BandwidthHz: 12_500,
		SampleRateHz: 8e6, GainDB: 20, IQGain: 1, LockCenter: true}
	active := ScanProfile{ID: "quick-tune", Channels: []ChannelDefinition{{ID: "quick-tune-channel", FrequencyHz: 450e6, Mode: "nfm", BandwidthHz: 12_500}}}
	runtime := &Runtime{running: true, tuning: true, active: &active, tunerUpdates: updates, tunerHardware: &activeRequest,
		spectrum: SpectrumSnapshot{CenterFrequencyHz: 450e6, SampleRateHz: 8e6}, mixer: []MixerChannel{{Channel: active.Channels[0]}}}

	next := activeRequest
	next.FrequencyHz = 452e6
	next.HardwareCenterHz = 450e6
	if err := runtime.Tune(next); err != nil {
		t.Fatal(err)
	}
	select {
	case update := <-updates:
		if update.FrequencyHz != 452e6 || update.HardwareCenterHz != 450e6 {
			t.Fatalf("unexpected software VFO update: %+v", update)
		}
	default:
		t.Fatal("software VFO update was not delivered")
	}
	if runtime.active.Channels[0].FrequencyHz != 452e6 || runtime.mixer[0].Channel.FrequencyHz != 452e6 {
		t.Fatal("active tuner state did not follow the software VFO")
	}

	next.FrequencyHz = 454e6
	if err := runtime.Tune(next); err == nil {
		t.Fatal("frequency outside the locked usable passband should be rejected")
	}
}

func TestLockedCenterRestartsWhenHardwareControlsChange(t *testing.T) {
	active := TunerRequest{DeviceID: "hackrf", SampleRateHz: 8e6, GainDB: 20, IQGain: 1}
	next := active
	next.GainDB = 24
	if sameTunerHardware(active, next) {
		t.Fatal("hardware gain change must not be treated as a software-only VFO update")
	}
}

func TestPartitionScanProfileSplitsConventionalWorkWithoutOverlap(t *testing.T) {
	profile := ScanProfile{ID: "multi", Channels: []ChannelDefinition{
		{ID: "a", FrequencyHz: 100e6, Enabled: true}, {ID: "b", FrequencyHz: 101e6, Enabled: true}, {ID: "c", FrequencyHz: 102e6, Enabled: true}, {ID: "d", FrequencyHz: 103e6, Enabled: true},
	}, Ranges: []ScanRange{{ID: "range", StartHz: 200e6, EndHz: 200.0875e6, StepHz: 12_500, Enabled: true}}}
	first, second := partitionScanProfile(profile, 0, 2), partitionScanProfile(profile, 1, 2)
	if len(first.Channels) != 2 || len(second.Channels) != 2 || first.Channels[1].ID != "b" || second.Channels[0].ID != "c" {
		t.Fatalf("unexpected channel partitions: %#v %#v", first.Channels, second.Channels)
	}
	seen := make(map[float64]bool)
	for _, target := range append(surveyTargets(first), surveyTargets(second)...) {
		if seen[target.FrequencyHz] {
			t.Fatalf("frequency %.0f assigned to more than one receiver", target.FrequencyHz)
		}
		seen[target.FrequencyHz] = true
	}
	if len(seen) != 12 {
		t.Fatalf("expected four channels plus eight range steps, got %d", len(seen))
	}
}
