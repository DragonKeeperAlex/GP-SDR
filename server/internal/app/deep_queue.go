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
	ResultApplied bool                `json:"resultApplied,omitempty"`
	ID            string              `json:"id"`
	Event         TransmissionEvent   `json:"event"`
	Fingerprint   string              `json:"fingerprint"`
	Status        string              `json:"status"`
	Attempts      int                 `json:"attempts"`
	NextAttemptAt *time.Time          `json:"nextAttemptAt,omitempty"`
	LastError     string              `json:"lastError,omitempty"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
	Result        *SignalIntelligence `json:"result,omitempty"`
}

type DeepAnalysisQueue struct {
	processMu sync.Mutex
	loadError error
	mu        sync.Mutex
	path      string
	jobs      []DeepAnalysisJob
	analyzer  *LocalAIAnalyzer
}

func NewDeepAnalysisQueue(dataDirectory string, analyzer *LocalAIAnalyzer) *DeepAnalysisQueue {
	if analyzer != nil {
		// Home-server stalls must not occupy the Pi first-pass semaphore.
		analyzer = &LocalAIAnalyzer{client: analyzer.client, semaphore: analyzer.semaphore, remoteSemaphore: make(chan struct{}, 1), learning: analyzer.learning, profiles: analyzer.profiles}
	}
	q := &DeepAnalysisQueue{path: filepath.Join(dataDirectory, "Data", "deep-analysis-queue.json"), analyzer: analyzer}
	if data, err := os.ReadFile(q.path); err == nil {
		q.loadError = json.Unmarshal(data, &q.jobs)
	} else if !errors.Is(err, os.ErrNotExist) {
		q.loadError = err
	}
	if q.loadError != nil {
		q.jobs = nil
	}
	if q.loadError == nil {
		for i := range q.jobs {
			if q.jobs[i].Status == "running" {
				q.jobs[i].Status = "retry"
				q.jobs[i].LastError = "Interrupted; queued after restart"
				q.jobs[i].NextAttemptAt = nil
			}
		}
	}
	return q
}

func (q *DeepAnalysisQueue) Enqueue(event TransmissionEvent) (DeepAnalysisJob, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.loadError != nil {
		return DeepAnalysisJob{}, false, q.loadError
	}
	fingerprint := event.ID
	for _, job := range q.jobs {
		if job.Fingerprint == fingerprint && job.Status != "discarded" {
			return job, false, nil
		}
	}
	previous := append([]DeepAnalysisJob(nil), q.jobs...)
	if len(q.jobs) >= 1000 {
		if !q.processMu.TryLock() {
			return DeepAnalysisJob{}, false, errors.New("deep-analysis queue is full and processing; try again later")
		}
		retained := make([]DeepAnalysisJob, 0, len(q.jobs))
		for _, job := range q.jobs {
			if job.Status != "complete" && job.Status != "discarded" {
				retained = append(retained, job)
			}
		}
		q.jobs = retained
		q.processMu.Unlock()
		if len(q.jobs) >= 1000 {
			q.jobs = previous
			return DeepAnalysisJob{}, false, errors.New("deep-analysis queue is full; resolve or discard failed jobs")
		}
	}

	now := time.Now().UTC()
	job := DeepAnalysisJob{ID: NewID(), Event: event, Fingerprint: fingerprint, Status: "pending", CreatedAt: now, UpdatedAt: now}
	q.jobs = append(q.jobs, job)
	if err := q.saveLocked(); err != nil {
		q.jobs = previous
		return DeepAnalysisJob{}, false, err
	}
	return job, true, nil
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
	if config.ComputeMode == "off" || (!config.Enabled && !config.RemoteEnabled) || q.analyzer == nil {
		return nil
	}
	if config.ComputeMode == "" && validateLocalAIEndpoint(config.RemoteEndpoint) != nil {
		return nil
	}
	if q.loadError != nil || !q.processMu.TryLock() {
		return q.List()
	}
	defer q.processMu.Unlock()
	if limit <= 0 || limit > 8 {
		limit = 1
	}
	q.mu.Lock()
	previous := append([]DeepAnalysisJob(nil), q.jobs...)
	selected := make([]int, 0)
	for i := range q.jobs {
		if (q.jobs[i].Status == "pending" || q.jobs[i].Status == "retry") && (q.jobs[i].NextAttemptAt == nil || !time.Now().Before(*q.jobs[i].NextAttemptAt)) {
			q.jobs[i].Status = "running"
			q.jobs[i].Attempts++
			selected = append(selected, i)
			if limit > 0 && len(selected) >= limit {
				break
			}
		}
	}
	if len(selected) == 0 {
		q.mu.Unlock()
		return q.List()
	}
	if err := q.saveLocked(); err != nil {
		q.jobs = previous
		q.mu.Unlock()
		return q.List()
	}
	q.mu.Unlock()
	for _, index := range selected {
		q.mu.Lock()
		event := q.jobs[index].Event
		discarded := q.jobs[index].Status == "discarded"
		q.mu.Unlock()
		if discarded {
			continue
		}
		var result SignalIntelligence
		var err error
		if config.ComputeMode == "" {
			remote := &LocalAIAnalyzer{client: q.analyzer.client, semaphore: q.analyzer.remoteSemaphore, learning: q.analyzer.learning, profiles: q.analyzer.profiles}
			result, err = remote.analyzeWithConfig(ctx, event, LocalAIConfig{Enabled: true, Endpoint: config.RemoteEndpoint, Model: config.RemoteModel, Profile: "deep", MinimumConfidence: config.RemoteMinConfidence, ContextLength: config.ContextLength})
		} else {
			result, err = q.analyzer.analyzePreferred(ctx, event, normalizeLocalAIConfig(config))
		}
		q.mu.Lock()
		if q.jobs[index].Status == "discarded" {
			q.mu.Unlock()
			continue
		}
		q.jobs[index].NextAttemptAt = nil
		if err != nil {
			q.jobs[index].Status = "retry"
			q.jobs[index].LastError = truncateText(err.Error(), 1000)
			next := time.Now().Add(time.Duration(min(q.jobs[index].Attempts, 10)) * 30 * time.Second)
			q.jobs[index].NextAttemptAt = &next
		} else {
			q.jobs[index].Status = "complete"
			q.jobs[index].Result = &result
			q.jobs[index].LastError = ""
		}
		q.jobs[index].UpdatedAt = time.Now().UTC()
		if err := q.saveLocked(); err != nil {
			q.jobs[index].Status = "retry"
			q.jobs[index].LastError = "Could not persist deep analysis: " + err.Error()
			q.mu.Unlock()
			return q.List()
		}
		q.mu.Unlock()
	}
	return q.List()
}

func (q *DeepAnalysisQueue) Discard(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].ID == id {
			previous := q.jobs[i]
			q.jobs[i].Status = "discarded"
			q.jobs[i].UpdatedAt = time.Now().UTC()
			if err := q.saveLocked(); err != nil {
				q.jobs[i] = previous
				return err
			}
			return nil
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

func (q *DeepAnalysisQueue) markApplied(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.jobs {
		if q.jobs[i].ID == id {
			previous := q.jobs[i]
			q.jobs[i].ResultApplied = true
			if err := q.saveLocked(); err != nil {
				q.jobs[i] = previous
				return err
			}
			return nil
		}
	}
	return errors.New("deep-analysis job not found")
}

type DeepAnalysisQueueSummary struct {
	Pending   int    `json:"pending"`
	Running   int    `json:"running"`
	Retry     int    `json:"retry"`
	Complete  int    `json:"complete"`
	LastError string `json:"lastError,omitempty"`
}

func (q *DeepAnalysisQueue) summary() *DeepAnalysisQueueSummary {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	result := &DeepAnalysisQueueSummary{}
	if q.loadError != nil {
		result.LastError = q.loadError.Error()
	}
	var last time.Time
	for _, job := range q.jobs {
		switch job.Status {
		case "pending":
			result.Pending++
		case "running":
			result.Running++
		case "retry":
			result.Retry++
			if job.UpdatedAt.After(last) {
				last = job.UpdatedAt
				result.LastError = job.LastError
			}
		case "complete":
			result.Complete++
		}
	}
	return result
}
