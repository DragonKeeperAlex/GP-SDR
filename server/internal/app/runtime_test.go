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
