package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type DeferredAnalysisCurrent struct {
	FrequencyHz float64 `json:"frequencyHz"`
	Location    string  `json:"location"`
	File        int     `json:"file"`
	Files       int     `json:"files"`
	Stage       string  `json:"stage"`
	EventID     string  `json:"eventID"`
}

type DeferredAnalysisLogEntry struct {
	At          time.Time `json:"at"`
	Level       string    `json:"level"`
	FrequencyHz float64   `json:"frequencyHz,omitempty"`
	Stage       string    `json:"stage,omitempty"`
	Message     string    `json:"message"`
}

type DeferredAnalysisStatus struct {
	Running        bool                       `json:"running"`
	Pending        int                        `json:"pending"`
	Completed      int                        `json:"completed"`
	Failed         int                        `json:"failed"`
	StartedAt      *time.Time                 `json:"startedAt,omitempty"`
	LastError      string                     `json:"lastError,omitempty"`
	Total          int                        `json:"total"`
	Groups         int                        `json:"groups"`
	GroupsDone     int                        `json:"groupsDone"`
	Current        *DeferredAnalysisCurrent   `json:"current,omitempty"`
	Active         []DeferredAnalysisCurrent  `json:"active"`
	Concurrency    int                        `json:"concurrency"`
	ElapsedSeconds float64                    `json:"elapsedSeconds"`
	ETASeconds     *float64                   `json:"etaSeconds,omitempty"`
	Log            []DeferredAnalysisLogEntry `json:"log"`
}

func (r *Runtime) DeferredAnalysisStatus() DeferredAnalysisStatus {
	pending := len(r.Events.PendingAnalysis(0, ""))
	r.analysisMu.RLock()
	defer r.analysisMu.RUnlock()
	active := make([]DeferredAnalysisCurrent, 0, len(r.analysisActive))
	for _, current := range r.analysisActive {
		active = append(active, current)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].FrequencyHz < active[j].FrequencyHz })
	status := DeferredAnalysisStatus{Running: r.analysisRunning, Pending: pending, Completed: r.analysisCompleted,
		Failed: r.analysisFailed, StartedAt: r.analysisStartedAt, LastError: r.analysisLastError, Total: r.analysisTotal,
		Groups: r.analysisGroups, GroupsDone: r.analysisGroupsDone, Current: r.analysisCurrent, Active: active, Concurrency: r.analysisConcurrency,
		Log: append([]DeferredAnalysisLogEntry(nil), r.analysisLog...)}
	if r.analysisStartedAt != nil {
		status.ElapsedSeconds = time.Since(*r.analysisStartedAt).Seconds()
	}
	done := r.analysisCompleted + r.analysisFailed
	if r.analysisRunning && done > 0 && r.analysisTotal > done {
		eta := status.ElapsedSeconds / float64(done) * float64(r.analysisTotal-done)
		status.ETASeconds = &eta
	}
	return status
}

// StartDeferredAnalysis consumes captures without opening a receiver. This is
// intentionally useful after a mobile/field collection run when the computer
// is back on AC power. The queue is bounded by normal EventStore and IQ caps.
func (r *Runtime) StartDeferredAnalysis(jobID string, concurrency int) (DeferredAnalysisStatus, error) {
	if concurrency < 0 || concurrency > 16 {
		return r.DeferredAnalysisStatus(), errors.New("parallel frequency groups must be Auto or between 1 and 16")
	}
	r.analysisMu.Lock()
	if r.analysisRunning {
		r.analysisMu.Unlock()
		return r.DeferredAnalysisStatus(), errors.New("deferred analysis is already running")
	}
	events := r.Events.PendingAnalysis(0, strings.TrimSpace(jobID))
	if len(events) == 0 {
		r.analysisMu.Unlock()
		return r.DeferredAnalysisStatus(), nil
	}
	stop := make(chan struct{})
	now := time.Now().UTC()
	r.analysisRunning, r.analysisStop, r.analysisStartedAt = true, stop, &now
	groups := groupDeferredEvents(events)
	if concurrency == 0 {
		concurrency = maxInt(2, minInt(8, goruntime.NumCPU()/2))
	}
	if concurrency < 1 || concurrency > 16 {
		r.analysisMu.Unlock()
		return r.DeferredAnalysisStatus(), errors.New("parallel frequency groups must be Auto or between 1 and 16")
	}
	if concurrency > len(groups) {
		concurrency = len(groups)
	}
	r.analysisCompleted, r.analysisFailed, r.analysisLastError = 0, 0, ""
	r.analysisTotal, r.analysisGroups, r.analysisGroupsDone = len(events), len(groups), 0
	r.analysisCurrent, r.analysisActive, r.analysisLog, r.analysisConcurrency = nil, make(map[string]DeferredAnalysisCurrent), nil, concurrency
	r.appendAnalysisLogLocked("info", 0, "queue", fmt.Sprintf("Queued %d files in %d frequency/location groups · %d parallel", len(events), len(groups), concurrency))
	r.analysisMu.Unlock()
	go r.runDeferredAnalysis(groups, stop, concurrency)
	return r.DeferredAnalysisStatus(), nil
}

func (r *Runtime) StopDeferredAnalysis() DeferredAnalysisStatus {
	r.analysisMu.Lock()
	if r.analysisRunning && r.analysisStop != nil {
		close(r.analysisStop)
		r.analysisStop = nil
	}
	r.analysisMu.Unlock()
	return r.DeferredAnalysisStatus()
}

func (r *Runtime) runDeferredAnalysis(groups [][]TransmissionEvent, stop <-chan struct{}, concurrency int) {
	defer func() {
		r.analysisMu.Lock()
		r.analysisRunning, r.analysisStop = false, nil
		r.analysisCurrent = nil
		r.analysisActive = nil
		r.analysisMu.Unlock()
	}()
	work := make(chan []TransmissionEvent)
	var workers sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for events := range work {
				select {
				case <-stop:
					return
				default:
					r.processDeferredGroup(events, stop)
				}
			}
		}()
	}
	for _, events := range groups {
		select {
		case <-stop:
			close(work)
			workers.Wait()
			return
		case work <- events:
		}
	}
	close(work)
	workers.Wait()
}

func (r *Runtime) processDeferredGroup(events []TransmissionEvent, stop <-chan struct{}) {
	frequency := events[0].FrequencyHz
	location := deferredLocationLabel(events[0])
	r.analysisMu.Lock()
	r.appendAnalysisLogLocked("info", frequency, "group", fmt.Sprintf("Analyzing %d files · %s", len(events), location))
	r.analysisMu.Unlock()
	for index, event := range events {
		select {
		case <-stop:
			r.analysisMu.Lock()
			r.appendAnalysisLogLocked("warning", frequency, "stopped", "Analysis stopped; unprocessed files remain queued")
			r.analysisMu.Unlock()
			return
		default:
		}
		r.setAnalysisCurrent(event, location, index+1, len(events), "loading")
		_ = r.Events.UpdateAnalysisStatus(event.ID, "running", "")
		if err := r.analyzeStoredEvent(event, stop); err != nil {
			r.clearAnalysisCurrent(event.ID)
			if errors.Is(err, context.Canceled) {
				_ = r.Events.UpdateAnalysisStatus(event.ID, "pending", "")
				return
			}
			_ = r.Events.UpdateAnalysisStatus(event.ID, "error", err.Error())
			r.analysisMu.Lock()
			r.analysisFailed++
			r.analysisLastError = err.Error()
			r.appendAnalysisLogLocked("error", event.FrequencyHz, "file", err.Error())
			r.analysisMu.Unlock()
			continue
		}
		_ = r.Events.UpdateAnalysisStatus(event.ID, "complete", "")
		resultMessage := fmt.Sprintf("File %d of %d processed", index+1, len(events))
		if current, ok := r.Events.Get(event.ID); ok {
			parts := []string{}
			if len(current.DecoderMessages) > 0 {
				parts = append(parts, fmt.Sprintf("%d decoded messages", len(current.DecoderMessages)))
			}
			if current.Transcript != nil && strings.TrimSpace(*current.Transcript) != "" {
				parts = append(parts, "transcript captured")
			}
			if current.Analysis != nil && strings.TrimSpace(current.Analysis.SignalFamily) != "" {
				parts = append(parts, fmt.Sprintf("%s · %.0f%%", current.Analysis.SignalFamily, current.Analysis.Confidence*100))
			}
			if len(parts) > 0 {
				resultMessage += " · " + strings.Join(parts, " · ")
			}
		}
		r.analysisMu.Lock()
		r.analysisCompleted++
		r.appendAnalysisLogLocked("success", event.FrequencyHz, "file", resultMessage)
		r.analysisMu.Unlock()
		r.clearAnalysisCurrent(event.ID)
	}
	r.combineDeferredGroup(events, stop)
	r.clearAnalysisCurrent(events[len(events)-1].ID)
	r.analysisMu.Lock()
	r.analysisGroupsDone++
	r.appendAnalysisLogLocked("success", frequency, "group", "Combined group evidence")
	r.analysisMu.Unlock()
}

func groupDeferredEvents(events []TransmissionEvent) [][]TransmissionEvent {
	groups := map[string][]TransmissionEvent{}
	for _, event := range events {
		key := fmt.Sprintf("%.0f|%s", event.FrequencyHz, deferredLocationKey(event))
		groups[key] = append(groups[key], event)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([][]TransmissionEvent, 0, len(keys))
	for _, key := range keys {
		sort.Slice(groups[key], func(i, j int) bool { return groups[key][i].StartedAt.Before(groups[key][j].StartedAt) })
		result = append(result, groups[key])
	}
	return result
}

func deferredLocationKey(event TransmissionEvent) string {
	if event.Location == nil {
		return "unknown"
	}
	// A tenth of a degree keeps nearby captures together while separating distinct receive areas.
	return fmt.Sprintf("%.1f,%.1f", event.Location.Latitude, event.Location.Longitude)
}
func deferredLocationLabel(event TransmissionEvent) string {
	if event.Location == nil {
		return "location not recorded"
	}
	if strings.TrimSpace(event.Location.Label) != "" {
		return event.Location.Label
	}
	return deferredLocationKey(event)
}
func (r *Runtime) appendAnalysisLogLocked(level string, frequency float64, stage, message string) {
	r.analysisLog = append(r.analysisLog, DeferredAnalysisLogEntry{At: time.Now().UTC(), Level: level, FrequencyHz: frequency, Stage: stage, Message: message})
	if len(r.analysisLog) > 300 {
		r.analysisLog = append([]DeferredAnalysisLogEntry(nil), r.analysisLog[len(r.analysisLog)-300:]...)
	}
}
func (r *Runtime) setAnalysisCurrent(event TransmissionEvent, location string, file, files int, stage string) {
	r.analysisMu.Lock()
	defer r.analysisMu.Unlock()
	if r.analysisActive == nil {
		r.analysisActive = make(map[string]DeferredAnalysisCurrent)
	}
	if file == 0 {
		if previous, ok := r.analysisActive[event.ID]; ok {
			file, files = previous.File, previous.Files
		}
	}
	current := DeferredAnalysisCurrent{FrequencyHz: event.FrequencyHz, Location: location, File: file, Files: files, Stage: stage, EventID: event.ID}
	r.analysisActive[event.ID] = current
	r.analysisCurrent = &current
}
func (r *Runtime) clearAnalysisCurrent(eventID string) {
	r.analysisMu.Lock()
	delete(r.analysisActive, eventID)
	r.analysisMu.Unlock()
}

func (r *Runtime) combineDeferredGroup(events []TransmissionEvent, stop <-chan struct{}) {
	if r.localAI == nil || len(events) == 0 {
		return
	}
	combined := events[len(events)-1]
	combined.DecoderMessages = nil
	combined.Callsigns = nil
	combined.Transcript = nil
	transcripts := []string{}
	for _, original := range events {
		if current, ok := r.Events.Get(original.ID); ok {
			combined.DecoderMessages = append(combined.DecoderMessages, current.DecoderMessages...)
			combined.Callsigns = mergeUniqueStrings(combined.Callsigns, current.Callsigns)
			if current.Transcript != nil && strings.TrimSpace(*current.Transcript) != "" {
				transcripts = append(transcripts, strings.TrimSpace(*current.Transcript))
			}
			if current.Analysis != nil && (combined.Analysis == nil || current.Analysis.Confidence > combined.Analysis.Confidence) {
				copy := *current.Analysis
				combined.Analysis = &copy
			}
		}
	}
	if len(transcripts) > 0 {
		text := strings.Join(transcripts, "\n--- next recording ---\n")
		combined.Transcript = &text
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	r.setAnalysisCurrent(combined, deferredLocationLabel(combined), len(events), len(events), "combining evidence")
	analysis, err := r.localAI.Analyze(ctx, combined)
	if err != nil {
		return
	}
	for _, event := range events {
		_ = r.Events.UpdateAnalysis(event.ID, analysis)
	}
	if r.mapper != nil {
		r.mapper.SetSignalIntelligence(combined.FrequencyHz, analysis)
	}
}

func (r *Runtime) analyzeStoredEvent(event TransmissionEvent, stop <-chan struct{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	if event.AudioPath == nil && event.IQPath != nil {
		path, analysis, err := deriveStoredAudio(r.dataDirectory, event)
		if err != nil {
			return err
		}
		if err := r.Events.UpdateAudioPath(event.ID, path); err != nil {
			return err
		}
		event.AudioPath = ptr(path)
		if err := r.Events.UpdateAnalysis(event.ID, analysis); err != nil {
			return err
		}
	}
	var audio []int16
	audioRate := 0
	if event.IQPath != nil && !fileExists(*event.IQPath) {
		return fmt.Errorf("IQ recording is missing: %s", *event.IQPath)
	}
	if event.AudioPath != nil && !fileExists(*event.AudioPath) {
		return fmt.Errorf("audio recording is missing: %s", *event.AudioPath)
	}
	if event.AudioPath == nil && event.IQPath == nil {
		return errors.New("no saved audio or IQ is available for analysis")
	}
	if event.AudioPath != nil {
		r.setAnalysisCurrent(event, deferredLocationLabel(event), 0, 0, "reading audio")
		if decoded, rate, err := readPCM16WAV(*event.AudioPath); err == nil {
			audio, audioRate = decoded, rate
		}
	}
	spec := CaptureSpec{CenterFrequencyHz: int64(event.FrequencyHz), SampleRateHz: 250_000}
	if event.IQPath != nil {
		if encoded, err := os.ReadFile(iqMetadataPath(*event.IQPath)); err == nil {
			var metadata IQCaptureMetadata
			if json.Unmarshal(encoded, &metadata) == nil {
				spec.CenterFrequencyHz, spec.SampleRateHz = metadata.CenterFrequencyHz, metadata.SampleRateHz
			}
		}
	}
	if candidate, ok := decoderCandidate(event.FrequencyHz, event.RequestedDecoder); ok && r.decoderReady(candidate.DecoderID) {
		r.setAnalysisCurrent(event, deferredLocationLabel(event), 0, 0, "decoder · "+candidate.DecoderID)
		messages, _ := runCandidateDecoder(ctx, candidate.DecoderID, audio, audioRate, stringValue(event.IQPath), event.FrequencyHz, spec)
		if len(messages) > 0 {
			_ = r.Events.UpdateDecoderMessages(event.ID, messages)
			if r.mapper != nil {
				r.mapper.SetDecodedMessages(event.FrequencyHz, candidate.DecoderID, messages)
			}
		}
	}
	if event.AudioPath != nil && ctx.Err() == nil {
		r.setAnalysisCurrent(event, deferredLocationLabel(event), 0, 0, "transcription")
		if transcript, err := r.transcriber.Transcribe(ctx, *event.AudioPath); err == nil && strings.TrimSpace(transcript) != "" {
			_ = r.Events.UpdateTranscript(event.ID, transcript)
			if r.mapper != nil {
				r.mapper.SetTranscript(event.FrequencyHz, transcript)
			}
		}
	}
	if ctx.Err() == nil && r.localAI != nil {
		r.setAnalysisCurrent(event, deferredLocationLabel(event), 0, 0, "local model")
		if current, ok := r.Events.Get(event.ID); ok {
			if analysis, err := r.localAI.Analyze(ctx, current); err == nil {
				_ = r.Events.UpdateAnalysis(event.ID, analysis)
				if r.mapper != nil {
					r.mapper.SetSignalIntelligence(event.FrequencyHz, analysis)
				}
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	current, ok := r.Events.Get(event.ID)
	if ok && current.IQPath != nil {
		r.setAnalysisCurrent(event, deferredLocationLabel(event), 0, 0, "retention cleanup")
		path, _, err := finalizeIQEvidence(*current.IQPath, current)
		if err != nil {
			return err
		}
		_ = r.Events.UpdateIQPath(event.ID, path)
	}
	return nil
}
