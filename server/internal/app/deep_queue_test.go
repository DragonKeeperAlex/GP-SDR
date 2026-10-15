package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
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
