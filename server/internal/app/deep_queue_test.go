package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeepAnalysisQueueEmptyListEncodesAsArray(t *testing.T) {
	queue := NewDeepAnalysisQueue(t.TempDir(), NewLocalAIAnalyzer(t.TempDir()))
	data, err := json.Marshal(queue.List())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("empty queue must encode as an array, got %s", data)
	}
}

func TestDeepAnalysisQueueDeduplicatesAndPersistsRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	q := NewDeepAnalysisQueue(t.TempDir(), analyzer)
	event := TransmissionEvent{ID: "queue-1", FrequencyHz: 145.5e6, Modulation: "NFM"}
	first, added, err := q.Enqueue(event)
	if err != nil || !added {
		t.Fatalf("enqueue failed: %+v %v", first, err)
	}
	second, added, err := q.Enqueue(event)
	if err != nil || added || second.ID != first.ID {
		t.Fatalf("deduplication failed: %+v %v %v", second, added, err)
	}
	q.Process(context.Background(), LocalAIConfig{RemoteEnabled: true, RemoteEndpoint: server.URL, RemoteModel: "test", RemoteMinConfidence: 55}, 1)
	if jobs := q.List(); len(jobs) != 1 || jobs[0].Status != "retry" || jobs[0].Attempts != 1 || jobs[0].LastError == "" {
		t.Fatalf("retry state not retained: %+v", jobs)
	}
	reloaded := NewDeepAnalysisQueue(filepath.Dir(filepath.Dir(q.path)), analyzer)
	if jobs := reloaded.List(); len(jobs) != 1 || jobs[0].Status != "retry" {
		t.Fatalf("queue did not persist: %+v", jobs)
	}
}

func TestDeepQueueRecoversInterruptedWorkAndPreservesCorruptSnapshots(t *testing.T) {
	root := t.TempDir()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	q := NewDeepAnalysisQueue(root, analyzer)
	q.jobs = []DeepAnalysisJob{{ID: "interrupted", Status: "running"}}
	if err := q.saveLocked(); err != nil {
		t.Fatal(err)
	}
	reloaded := NewDeepAnalysisQueue(root, analyzer)
	if reloaded.List()[0].Status != "retry" {
		t.Fatal("interrupted job stuck running")
	}
	if err := os.WriteFile(q.path, []byte("broken; preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	reloaded = NewDeepAnalysisQueue(root, analyzer)
	if _, _, err := reloaded.Enqueue(TransmissionEvent{ID: "new"}); err == nil {
		t.Fatal("corrupt queue overwritten")
	}
	if data, _ := os.ReadFile(q.path); string(data) != "broken; preserve" {
		t.Fatal("corrupt snapshot lost")
	}
	q.path = t.TempDir()
	q.jobs = nil
	if _, _, err := q.Enqueue(TransmissionEvent{ID: "failure"}); err == nil || len(q.List()) != 0 {
		t.Fatal("failed enqueue changed state")
	}
}

func TestDeepQueueSerializesRemoteWorkWithoutBlockingFirstPass(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; http.Error(w, "offline", 503) }))
	defer server.Close()
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	q := NewDeepAnalysisQueue(t.TempDir(), analyzer)
	if _, _, err := q.Enqueue(TransmissionEvent{ID: "one"}); err != nil {
		t.Fatal(err)
	}
	analyzer.semaphore <- struct{}{}
	defer func() { <-analyzer.semaphore }()
	cfg := LocalAIConfig{RemoteEnabled: true, RemoteEndpoint: server.URL, RemoteModel: "fixture"}
	done := make(chan struct{})
	go func() { q.Process(context.Background(), cfg, 1); close(done) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("remote worker blocked on Pi first-pass semaphore")
	}
	if jobs := q.Process(context.Background(), cfg, 1); jobs[0].Attempts != 1 {
		t.Fatal("duplicate remote processing")
	}
	close(release)
	<-done
	if jobs := q.List(); jobs[0].Status != "retry" || jobs[0].NextAttemptAt == nil {
		t.Fatal("failed remote job lacks backoff")
	}
	q.Process(context.Background(), cfg, 1)
	if q.List()[0].Attempts != 1 {
		t.Fatal("retry ignored backoff")
	}
}

func TestDeepQueueAppliesCompletedResultsOnlyOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"response": `{"signalFamily":"UNKNOWN","modulation":"UNKNOWN","summary":"Synthetic evidence only","confidence":0.2,"evidence":[],"callsigns":[]}`})
	}))
	defer server.Close()
	root := t.TempDir()
	events, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	event := TransmissionEvent{ID: "apply-once", FrequencyHz: 145500000, Modulation: "NFM"}
	if err := events.Append(event); err != nil {
		t.Fatal(err)
	}
	analyzer := NewLocalAIAnalyzer(t.TempDir())
	analyzer.config = LocalAIConfig{RemoteEnabled: true, RemoteEndpoint: server.URL, RemoteModel: "fixture"}
	q := NewDeepAnalysisQueue(t.TempDir(), analyzer)
	q.Enqueue(event)
	r := &Runtime{Events: events, localAI: analyzer, deepQueue: q}
	r.ProcessDeepAnalysis(context.Background(), 1)
	first, err := os.Stat(events.path)
	if err != nil {
		t.Fatal(err)
	}
	r.ProcessDeepAnalysis(context.Background(), 1)
	second, err := os.Stat(events.path)
	if err != nil {
		t.Fatal(err)
	}
	if !q.List()[0].ResultApplied || first.Size() != second.Size() {
		t.Fatal("completed result repeatedly rewrote journal")
	}
}
