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

func TestLongP25ProfileNamesCanBeSavedAndDuplicated(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile := ScanProfile{SchemaVersion: 1, ID: NewID(), Name: "East Bay Regional Communications System (EBRCS) · Contra Costa · Site: 006 CCCO East Simulcast", Settings: defaultSettings()}
	if _, err := store.Save(profile); err != nil {
		t.Fatal(err)
	}
	profile.Name = strings.Repeat("é", 160)
	if _, err := store.Save(profile); err != nil {
		t.Fatal(err)
	}
	duplicate, err := store.Duplicate(profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(duplicate); err != nil {
		t.Fatal(err)
	}
	profile.Name = strings.Repeat("x", 161)
	if _, err := store.Save(profile); err == nil {
		t.Fatal("over-limit name accepted")
	}
	profile.Name = "   "
	if _, err := store.Save(profile); err == nil {
		t.Fatal("blank name accepted")
	}
}

func TestFailedProfileSavePreservesMemory(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile := ScanProfile{SchemaVersion: 1, ID: NewID(), Name: "Before", Settings: defaultSettings()}
	if _, err := store.Save(profile); err != nil {
		t.Fatal(err)
	}
	badDirectory := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badDirectory, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	store.dir = badDirectory
	profile.Name = "After"
	if _, err := store.Save(profile); err == nil {
		t.Fatal("save should fail")
	}
	saved, _ := store.Get(profile.ID)
	if saved.Name != "Before" {
		t.Fatal("failed save changed in-memory profile")
	}
	count := len(store.All())
	if _, err := store.Duplicate(profile.ID); err == nil {
		t.Fatal("duplicate should fail")
	}
	if len(store.All()) != count {
		t.Fatal("failed duplicate created phantom profile")
	}
	if err := store.Delete(profile.ID); err == nil {
		t.Fatal("delete should fail")
	}
	if _, exists := store.Get(profile.ID); !exists {
		t.Fatal("failed delete hid profile")
	}
}

func TestProfilePathsCannotEscapeStore(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../escape", "/tmp/escape", `..\escape`, ".", "..", "bad\x00id"} {
		profile := ScanProfile{SchemaVersion: 1, ID: id, Name: "Test", Settings: defaultSettings()}
		if _, err := store.Save(profile); err == nil {
			t.Fatalf("unsafe save ID accepted: %q", id)
		}
		data, _ := json.Marshal(profile)
		if _, err := store.Import(data); err == nil {
			t.Fatalf("unsafe import ID accepted: %q", id)
		}
	}
}

func TestProfileValidationRejectsBadRange(t *testing.T) {
	profile := ScanProfile{SchemaVersion: 1, ID: NewID(), Name: "Bad", Ranges: []ScanRange{{ID: NewID(), Name: "Backwards", StartHz: 200, EndHz: 100, StepHz: 1, DwellMilliseconds: 100}}}
	if validateProfile(profile) == nil {
		t.Fatal("expected invalid range to be rejected")
	}
}

func TestProfileValidationRejectsUnsupportedP25CaptureRate(t *testing.T) {
	profile := sdrTrunkTestProfile()
	profile.SchemaVersion = 1
	profile.Name = "Invalid capture rate"
	profile.Settings.P25SampleRateHz = 5_500_000
	err := validateProfile(profile)
	if err == nil || !strings.Contains(err.Error(), "unsupported P25 capture rate") {
		t.Fatalf("expected unsupported P25 capture rate error, got %v", err)
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

func TestDeferredAnalysisQueueAndStatusUpdates(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	queued := TransmissionEvent{ID: "queued", StartedAt: time.Now(), FrequencyHz: 155.25e6, MapperJobID: "job-a", AnalysisPolicy: "manual", AnalysisStatus: "pending"}
	complete := TransmissionEvent{ID: "complete", StartedAt: time.Now(), FrequencyHz: 155.26e6, MapperJobID: "job-a", AnalysisPolicy: "manual", AnalysisStatus: "complete"}
	for _, event := range []TransmissionEvent{queued, complete} {
		if err := store.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	if pending := store.PendingAnalysis(10, "job-a"); len(pending) != 1 || pending[0].ID != queued.ID {
		t.Fatalf("unexpected deferred queue: %#v", pending)
	}
	if err := store.UpdateAnalysisStatus(queued.ID, "complete", ""); err != nil {
		t.Fatal(err)
	}
	if pending := store.PendingAnalysis(10, "job-a"); len(pending) != 0 {
		t.Fatalf("completed event remained queued: %#v", pending)
	}
	updated, ok := store.Get(queued.ID)
	if !ok || updated.AnalysisStatus != "complete" || updated.AnalysisCompletedAt == nil {
		t.Fatalf("analysis completion was not recorded: %#v", updated)
	}
}
