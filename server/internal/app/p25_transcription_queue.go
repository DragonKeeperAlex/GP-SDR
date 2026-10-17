package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type P25TranscriptionJob struct {
	EventID  string `json:"eventID"`
	State    string `json:"state"`
	Error    string `json:"error,omitempty"`
	Attempts int    `json:"attempts"`
}

// Queue metadata is durable; audio stays at its original journal path. One
// worker shares the global transcription semaphore and survives monitoring Stop.
type P25TranscriptionQueue struct {
	onComplete       func(string)
	mu               sync.Mutex
	path             string
	jobs             []P25TranscriptionJob
	loadError        error
	persistenceError string
	wake             chan struct{}
	events           *EventStore
	transcribe       func(context.Context, string) (string, error)
}

func newP25TranscriptionQueue(root string, events *EventStore, transcribe func(context.Context, string) (string, error)) *P25TranscriptionQueue {
	q := &P25TranscriptionQueue{path: filepath.Join(root, "Data", "p25-transcription-queue.json"), wake: make(chan struct{}, 1), events: events, transcribe: transcribe}
	data, err := os.ReadFile(q.path)
	if err == nil {
		err = json.Unmarshal(data, &q.jobs)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		q.loadError = fmt.Errorf("P25 transcription queue: %w", err)
		return q
	}
	for i := range q.jobs {
		if q.jobs[i].State == "running" {
			q.jobs[i].State = "pending"
			q.jobs[i].Error = "Interrupted; queued after restart"
		}
	}
	return q
}
func (q *P25TranscriptionQueue) snapshot() []P25TranscriptionJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]P25TranscriptionJob{}, q.jobs...)
}
func (q *P25TranscriptionQueue) enqueue(eventID string, retry bool) error {
	event, ok := q.events.Get(eventID)
	if !ok || !p25TranscriptionEligible(event) {
		return fmt.Errorf("call is not eligible for transcription")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.loadError != nil {
		return q.loadError
	}
	next := append([]P25TranscriptionJob{}, q.jobs...)
	found := false
	for i := range next {
		if next[i].EventID == eventID {
			found = true
			if !retry || next[i].State == "running" || next[i].State == "pending" {
				if next[i].State == "pending" {
					select {
					case q.wake <- struct{}{}:
					default:
					}
				}
				return nil
			}
			next[i].State = "pending"
			next[i].Error = ""
			break
		}
	}
	if !found {
		if len(next) >= 1000 {
			for i, job := range next {
				if job.State == "complete" || job.State == "no_speech" {
					next = append(next[:i], next[i+1:]...)
					break
				}
			}
		}
		if len(next) >= 1000 {
			return fmt.Errorf("transcription queue is full; resolve failed jobs before adding calls")
		}
		next = append(next, P25TranscriptionJob{EventID: eventID, State: "pending"})
	}
	if err := writeJSONAtomic(q.path, next); err != nil {
		return err
	}
	q.jobs = next
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return nil
}
func (q *P25TranscriptionQueue) run(ctx context.Context) {
	if q.loadError != nil {
		return
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
		}
		for ctx.Err() == nil && q.processOne(ctx) {
		}
	}
}
func (q *P25TranscriptionQueue) processOne(ctx context.Context) bool {
	q.mu.Lock()
	index := -1
	for i, job := range q.jobs {
		if job.State == "pending" {
			index = i
			break
		}
	}
	if index < 0 {
		q.mu.Unlock()
		return false
	}
	next := append([]P25TranscriptionJob{}, q.jobs...)
	next[index].State = "running"
	next[index].Error = ""
	next[index].Attempts++
	if err := writeJSONAtomic(q.path, next); err != nil {
		q.persistenceError = err.Error()
		q.mu.Unlock()
		return false
	}
	q.jobs = next
	job := next[index]
	q.mu.Unlock()
	event, ok := q.events.Get(job.EventID)
	text := ""
	var err error
	if !ok || !p25TranscriptionEligible(event) {
		err = fmt.Errorf("call audio is unavailable or ineligible")
	} else {
		text, err = q.transcribe(ctx, *event.AudioPath)
	}
	if err == nil && text != "" {
		err = q.events.UpdateTranscript(job.EventID, text)
	}
	q.mu.Lock()
	next = append([]P25TranscriptionJob{}, q.jobs...)
	// Find by identity: enqueue may have pruned a completed job during execution.
	for i := range next {
		if next[i].EventID == job.EventID {
			next[i].State = "complete"
			if text == "" {
				next[i].State = "no_speech"
			}
			if err != nil {
				next[i].State = "failed"
				next[i].Error = truncateText(err.Error(), 1000)
			}
			if ctx.Err() != nil {
				next[i].State = "pending"
				next[i].Error = "Interrupted; waiting for restart"
			}
			break
		}
	}
	if err := writeJSONAtomic(q.path, next); err != nil {
		q.persistenceError = err.Error()
		for i := range q.jobs {
			if q.jobs[i].EventID == job.EventID {
				q.jobs[i].State = "failed"
				q.jobs[i].Error = "Could not persist transcription result: " + err.Error()
			}
		}
		q.mu.Unlock()
		return false
	}
	q.persistenceError = ""
	q.jobs = next
	q.mu.Unlock()
	if err == nil && text != "" && ctx.Err() == nil && q.onComplete != nil {
		q.onComplete(job.EventID)
	}
	return ctx.Err() == nil
}

func (q *P25TranscriptionQueue) decorate(page *P25RecordingsPage) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	states := map[string]P25TranscriptionJob{}
	for _, job := range q.jobs {
		states[job.EventID] = job
	}
	for i := range page.Calls {
		if job, ok := states[page.Calls[i].ID]; ok {
			page.Calls[i].TranscriptionState = job.State
			page.Calls[i].TranscriptionError = job.Error
		}
	}
	page.QueueError = q.persistenceError
	if q.loadError != nil {
		page.QueueError = q.loadError.Error()
	}
}

func p25TranscriptionEligible(event TransmissionEvent) bool {
	protocol := strings.ToLower(stringValue(event.ProtocolName))
	return (protocol == "p25" || protocol == "p25 phase 1" || protocol == "p25 phase 2") && !event.Encrypted && !event.Simulated && event.TalkgroupID != nil && *event.TalkgroupID > 0 && event.AudioPath != nil && *event.AudioPath != ""
}
