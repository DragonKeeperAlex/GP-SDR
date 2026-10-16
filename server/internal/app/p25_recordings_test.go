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
