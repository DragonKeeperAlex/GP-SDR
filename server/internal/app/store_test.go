package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestEventStorePrunesOnlyLegacyMapperFalsePositives(t *testing.T) {
	directory := t.TempDir()
	bad := TransmissionEvent{ID: "bad", StartedAt: time.Now(), DurationSeconds: .2, FrequencyHz: 10e6, SignalDBFS: -26, NoiseDBFS: 0, Modulation: "NFM", Label: ptr("Mapper discovery"), Confidence: .72}
	good := TransmissionEvent{ID: "good", StartedAt: time.Now(), DurationSeconds: .2, FrequencyHz: 155.25e6, SignalDBFS: -30, NoiseDBFS: -78, Modulation: "NFM", Label: ptr("Measured"), Confidence: .72}
	badData, _ := json.Marshal(bad)
	goodData, _ := json.Marshal(good)
	path := filepath.Join(directory, "events.jsonl")
	if err := os.WriteFile(path, append(append(badData, '\n'), append(goodData, '\n')...), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewEventStore(directory)
	if err != nil {
		t.Fatal(err)
	}
	if store.Count() != 1 || store.events[0].ID != "good" || len(store.Signals(10)) != 1 {
		t.Fatalf("legacy event migration kept the wrong rows: %#v", store.events)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"id":"bad"`) || !strings.Contains(string(data), `"id":"good"`) {
		t.Fatalf("event file was not rewritten correctly: %s", data)
	}
}

func TestEventSearchIndexesTranscriptCallsignProtocolAndFrequency(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	event := TransmissionEvent{ID: "search-event", StartedAt: time.Now(), FrequencyHz: 462_562_500, Modulation: "NFM",
		ProtocolName: ptr("Analog FM"), Label: ptr("GMRS Trail Group"), Transcript: ptr("K6ABC arriving at camp"), Callsigns: []string{"K6ABC"}}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"K6ABC", "trail camp", "analog", "462.5625", "gmrs"} {
		results := store.Search(query, 10)
		if len(results) != 1 || results[0].ID != event.ID {
			t.Fatalf("query %q did not find the event: %#v", query, results)
		}
	}
	if results := store.Search("not-present", 10); len(results) != 0 {
		t.Fatalf("unexpected nonmatching results: %#v", results)
	}
	if tokens := strings.Join(searchTokens("P25 Phase-2"), ","); tokens != "p25,phase,2" {
		t.Fatalf("unexpected search tokens: %s", tokens)
	}
}
