package app

import (
	"fmt"
	"testing"
	"time"
)

func TestP25RecordingsFiltersBeforePagingAndIgnoresSurveyTraffic(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	group := uint32(2606)
	call := func(id string) TransmissionEvent {
		return TransmissionEvent{ID: id, StartedAt: time.Date(2026, 10, 15, 12, 0, 0, 0, time.Local), ProtocolName: ptr("P25 Phase 2"), TalkgroupID: &group, SystemName: ptr("Central"), Transcript: ptr("Dispatch test"), AudioPath: ptr("missing-test.wav")}
	}
	for _, id := range []string{"old", "middle", "new"} {
		if err := store.Append(call(id)); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"encrypted", "simulated", "guess"} {
		e := call(id)
		e.Encrypted = id == "encrypted"
		e.Simulated = id == "simulated"
		if id == "guess" {
			e.ProtocolName = ptr("P25 or land mobile")
		}
		if err := store.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	// More general events than the old activity page limit must not hide calls.
	for i := 0; i < 175; i++ {
		e := call(fmt.Sprint("survey-", i))
		e.ProtocolName = ptr("WFM")
		if err := store.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	page, err := store.P25Recordings("central 2606 2026-10-15", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Calls) != 2 || page.Calls[0].ID != "new" || page.Calls[1].ID != "middle" || page.NextBefore != "middle" {
		t.Fatalf("first page: %+v", page)
	}
	// New calls arriving between pages cannot cause a duplicate or shift the cursor.
	if err := store.Append(call("later")); err != nil {
		t.Fatal(err)
	}
	older, err := store.P25Recordings("dispatch", page.NextBefore, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(older.Calls) != 1 || older.Calls[0].ID != "old" || older.NextBefore != "" {
		t.Fatalf("older page: %+v", older)
	}
	empty, err := store.P25Recordings("absent", "", 50)
	if err != nil || len(empty.Calls) != 0 {
		t.Fatalf("empty page: %+v %v", empty, err)
	}
	if _, err := store.P25Recordings("", "missing", 50); err == nil {
		t.Fatal("invalid cursor accepted")
	}
}

func TestP25RecordingsPreserveUnavailableAudio(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	event := TransmissionEvent{ID: "missing", ProtocolName: ptr("P25"), TalkgroupID: ptr(uint32(3255)), MediaIssues: []string{"Missing recording: old.wav"}}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	page, err := store.P25Recordings("", "", 50)
	if err != nil || len(page.Calls) != 1 || page.Calls[0].AudioAvailable {
		t.Fatalf("missing metadata hidden: %+v %v", page, err)
	}
}

func TestP25ArchiveDerivesHistoricalDurationWithoutChangingJournal(t *testing.T) {
	root := t.TempDir()
	store, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	path := root + "/audio.wav"
	if err := WriteMonoWAV(path, make([]int16, 16000), 8000); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(TransmissionEvent{ID: "old-duration", ProtocolName: ptr("P25"), TalkgroupID: ptr(uint32(3255)), AudioPath: &path, DurationSeconds: 0.001}); err != nil {
		t.Fatal(err)
	}
	page, err := store.P25Recordings("", "", 50)
	if err != nil || !page.Calls[0].AudioAvailable || page.Calls[0].RecordedDurationSeconds == nil || *page.Calls[0].RecordedDurationSeconds != 2 {
		t.Fatal("incorrect actual WAV duration")
	}
	original, _ := store.Get("old-duration")
	if original.DurationSeconds != 0.001 {
		t.Fatal("historical journal changed during read")
	}
}
