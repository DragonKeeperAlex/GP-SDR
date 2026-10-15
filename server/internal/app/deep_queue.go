package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DeepAnalysisJob struct {
	ID          string              `json:"id"`
	Event       TransmissionEvent   `json:"event"`
	Fingerprint string              `json:"fingerprint"`
	Status      string              `json:"status"`
	Attempts    int                 `json:"attempts"`
	LastError   string              `json:"lastError,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	Result      *SignalIntelligence `json:"result,omitempty"`
}

type DeepAnalysisQueue struct {
	mu       sync.Mutex
	path     string
	jobs     []DeepAnalysisJob
	analyzer *LocalAIAnalyzer
}

func NewDeepAnalysisQueue(dataDirectory string, analyzer *LocalAIAnalyzer) *DeepAnalysisQueue {
	q := &DeepAnalysisQueue{path: filepath.Join(dataDirectory, "Data", "deep-analysis-queue.json"), analyzer: analyzer}
	if data, err := os.ReadFile(q.path); err == nil {
		_ = json.Unmarshal(data, &q.jobs)
	}
	return q
}

func (q *DeepAnalysisQueue) Enqueue(event TransmissionEvent) (DeepAnalysisJob, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	fingerprint := event.ID
	for _, job := range q.jobs {
		if job.Fingerprint == fingerprint && job.Status != "discarded" {
			return job, false, nil
		}
	}
	now := time.Now().UTC()
	job := DeepAnalysisJob{ID: NewID(), Event: event, Fingerprint: fingerprint, Status: "pending", CreatedAt: now, UpdatedAt: now}
	q.jobs = append(q.jobs, job)
	return job, true, q.saveLocked()
}

func (q *DeepAnalysisQueue) List() []DeepAnalysisJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	// Keep the API contract array-shaped when no jobs have been queued.  A
	// JSON null response forces every UI client to special-case an otherwise
	// ordinary empty collection.
	return append([]DeepAnalysisJob{}, q.jobs...)
}

func (q *DeepAnalysisQueue) Process(ctx context.Context, config LocalAIConfig, limit int) []DeepAnalysisJob {
	if !config.RemoteEnabled || q.analyzer == nil {
		return nil
	}
	if validateLocalAIEndpoint(config.RemoteEndpoint) != nil {
		return nil
	}
	q.mu.Lock()
	selected := make([]int, 0)
	for i := range q.jobs {
		if q.jobs[i].Status == "pending" || q.jobs[i].Status == "retry" {
			q.jobs[i].Status = "running"
			q.jobs[i].Attempts++
			selected = append(selected, i)
			if limit > 0 && len(selected) >= limit {
				break
			}
		}
	}
	_ = q.saveLocked()
	q.mu.Unlock()
	for _, index := range selected {
		result, err := q.analyzer.analyzeWithConfig(ctx, q.jobs[index].Event, LocalAIConfig{Enabled: true, Endpoint: config.RemoteEndpoint, Model: config.RemoteModel, Profile: "deep", MinimumConfidence: config.RemoteMinConfidence, ContextLength: config.ContextLength})
		q.mu.Lock()
		if err != nil {
			q.jobs[index].Status = "retry"
			q.jobs[index].LastError = err.Error()
		} else {
			q.jobs[index].Status = "complete"
			q.jobs[index].Result = &result
			q.jobs[index].LastError = ""
		}
		q.jobs[index].UpdatedAt = time.Now().UTC()
		_ = q.saveLocked()
		q.mu.Unlock()
	}
	return q.List()
}

func (q *DeepAnalysisQueue) Discard(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].ID == id {
			q.jobs[i].Status = "discarded"
			q.jobs[i].UpdatedAt = time.Now().UTC()
			return q.saveLocked()
		}
	}
	return errors.New("deep-analysis job not found")
}

func (q *DeepAnalysisQueue) saveLocked() error {
	data, err := json.MarshalIndent(q.jobs, "", "  ")
	if err != nil {
		return err
	}
	return writeBytesAtomic(q.path, append(data, '\n'))
}
