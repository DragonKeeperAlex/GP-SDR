package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestP25TranscriptionQueueRetryRestartAndDeduplication(t *testing.T) {
	root := t.TempDir()
	events, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second", "encrypted"} {
		e := TransmissionEvent{ID: id, ProtocolName: ptr("P25"), TalkgroupID: ptr(uint32(3255)), AudioPath: ptr("fixture.wav"), Encrypted: id == "encrypted"}
		if err := events.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	failed := true
	transcribe := func(ctx context.Context, path string) (string, error) {
		if failed {
			return "", errors.New("model offline")
		}
		return "Test dispatch", nil
	}
	q := newP25TranscriptionQueue(root, events, transcribe)
	if q.enqueue("encrypted", false) == nil {
		t.Fatal("encrypted call queued")
	}
	for i := 0; i < 2; i++ {
		if err := q.enqueue("first", false); err != nil {
			t.Fatal(err)
		}
	}
	if len(q.snapshot()) != 1 {
		t.Fatal("duplicate job")
	}
	q.processOne(context.Background())
	if q.snapshot()[0].State != "failed" {
		t.Fatal("failure hidden")
	}
	q = newP25TranscriptionQueue(root, events, transcribe)
	if q.snapshot()[0].State != "failed" {
		t.Fatal("failure lost on restart")
	}
	failed = false
	notified := false
	q.onComplete = func(id string) {
		if id != "first" || q.snapshot()[0].State != "complete" {
			t.Error("completion callback ran before durable completion")
		}
		notified = true
	}
	if err := q.enqueue("first", true); err != nil {
		t.Fatal(err)
	}
	q.processOne(context.Background())
	event, _ := events.Get("first")
	if event.Transcript == nil || *event.Transcript != "Test dispatch" || q.snapshot()[0].State != "complete" || !notified {
		t.Fatal("retry failed")
	}
	if err := q.enqueue("second", false); err != nil {
		t.Fatal(err)
	}
	q.transcribe = func(context.Context, string) (string, error) { return "", context.Canceled }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q.processOne(ctx)
	q = newP25TranscriptionQueue(root, events, transcribe)
	if q.snapshot()[1].State != "pending" {
		t.Fatal("interrupted job not recovered")
	}
	q.processOne(context.Background())
	if q.snapshot()[1].State != "complete" {
		t.Fatal("restart did not resume")
	}
}
func TestP25TranscriptionQueuePreservesCorruptFileAndFailedPersistence(t *testing.T) {
	root := t.TempDir()
	events, _ := NewEventStore(filepath.Join(root, "Data"))
	events.Append(TransmissionEvent{ID: "call", ProtocolName: ptr("P25"), TalkgroupID: ptr(uint32(1)), AudioPath: ptr("fixture.wav")})
	q := newP25TranscriptionQueue(root, events, nil)
	original := []byte("broken queue; preserve this")
	if err := os.WriteFile(q.path, original, 0600); err != nil {
		t.Fatal(err)
	}
	q = newP25TranscriptionQueue(root, events, nil)
	if q.enqueue("call", false) == nil {
		t.Fatal("corrupt queue overwritten")
	}
	if data, _ := os.ReadFile(q.path); string(data) != string(original) {
		t.Fatal("corrupt evidence changed")
	}
	q.path = t.TempDir()
	q.loadError = nil
	if q.enqueue("call", false) == nil || len(q.snapshot()) != 0 {
		t.Fatal("failed enqueue changed memory")
	}
}
