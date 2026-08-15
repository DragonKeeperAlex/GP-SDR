package app

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProfileImportAndDuplicate(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile := ScanProfile{SchemaVersion: 1, ID: NewID(), Name: "Test", Ranges: []ScanRange{{ID: NewID(), Name: "Range", StartHz: 100e6, EndHz: 101e6, StepHz: 12500, DwellMilliseconds: 100, PreferredMode: "auto", Enabled: true}}, DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "discovery"}}, Settings: defaultSettings()}
	data, _ := json.Marshal(profile)
	imported, err := store.Import(data)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.Duplicate(imported.ID)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID == imported.ID || duplicate.Name != "Test Copy" {
		t.Fatalf("unexpected duplicate: %#v", duplicate)
	}
}

func TestProfileValidationRejectsBadRange(t *testing.T) {
	profile := ScanProfile{SchemaVersion: 1, ID: NewID(), Name: "Bad", Ranges: []ScanRange{{ID: NewID(), Name: "Backwards", StartHz: 200, EndHz: 100, StepHz: 1, DwellMilliseconds: 100}}}
	if validateProfile(profile) == nil {
		t.Fatal("expected invalid range to be rejected")
	}
}

func TestEventAggregation(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		event := TransmissionEvent{ID: NewID(), StartedAt: time.Now().Add(time.Duration(i) * time.Second), DurationSeconds: 2, FrequencyHz: 162.55e6, BandwidthHz: 12500, SignalDBFS: -30 + float64(i), NoiseDBFS: -70, Modulation: "NFM", DeviceID: "test", Confidence: .8}
		if err := store.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	signals := store.Signals(10)
	if len(signals) != 1 || signals[0].EventCount != 2 {
		t.Fatalf("unexpected aggregate: %#v", signals)
	}
}
