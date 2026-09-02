package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ConfirmedSignalSample struct {
	ID              string              `json:"id"`
	EventID         string              `json:"eventID"`
	ConfirmedAt     time.Time           `json:"confirmedAt"`
	FrequencyHz     float64             `json:"frequencyHz"`
	BandwidthHz     float64             `json:"bandwidthHz"`
	Modulation      string              `json:"modulation"`
	Protocol        string              `json:"protocol"`
	Notes           string              `json:"notes,omitempty"`
	SignalDBFS      float64             `json:"signalDBFS"`
	NoiseDBFS       float64             `json:"noiseDBFS"`
	Analysis        *SignalIntelligence `json:"analysis,omitempty"`
	DecoderMessages []DecoderMessage    `json:"decoderMessages,omitempty"`
	Transcript      string              `json:"transcript,omitempty"`
	Callsigns       []string            `json:"callsigns,omitempty"`
	IQPath          string              `json:"iqPath,omitempty"`
	AudioPath       string              `json:"audioPath,omitempty"`
}

type LearningLibraryStatus struct {
	Count     int        `json:"count"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type SignalLearningLibrary struct {
	mu      sync.RWMutex
	path    string
	samples []ConfirmedSignalSample
}

func NewSignalLearningLibrary(dataDirectory string) *SignalLearningLibrary {
	l := &SignalLearningLibrary{path: filepath.Join(dataDirectory, "Data", "confirmed-signal-samples.json")}
	if data, err := os.ReadFile(l.path); err == nil {
		_ = json.Unmarshal(data, &l.samples)
	}
	return l
}

func (l *SignalLearningLibrary) Status() LearningLibraryStatus {
	l.mu.RLock()
	defer l.mu.RUnlock()
	status := LearningLibraryStatus{Count: len(l.samples)}
	if len(l.samples) > 0 {
		latest := l.samples[len(l.samples)-1].ConfirmedAt
		status.UpdatedAt = &latest
	}
	return status
}

func (l *SignalLearningLibrary) Confirm(event TransmissionEvent, modulation, protocol, notes string, retainCaptures bool) (ConfirmedSignalSample, error) {
	if event.Simulated {
		return ConfirmedSignalSample{}, errors.New("simulated events cannot be added to the confirmed learning library")
	}
	modulation, protocol = strings.ToUpper(strings.TrimSpace(modulation)), strings.TrimSpace(protocol)
	if modulation == "" || modulation == "AUTO" || modulation == "UNKNOWN" {
		return ConfirmedSignalSample{}, errors.New("choose a confirmed modulation")
	}
	if protocol == "" {
		protocol = modulation
	}
	sample := ConfirmedSignalSample{ID: NewID(), EventID: event.ID, ConfirmedAt: time.Now(), FrequencyHz: event.FrequencyHz, BandwidthHz: event.BandwidthHz,
		Modulation: modulation, Protocol: protocol, Notes: strings.TrimSpace(notes), SignalDBFS: event.SignalDBFS, NoiseDBFS: event.NoiseDBFS,
		Analysis: event.Analysis, DecoderMessages: event.DecoderMessages, Transcript: stringValue(event.Transcript), Callsigns: append([]string(nil), event.Callsigns...)}
	if retainCaptures {
		sample.IQPath, sample.AudioPath = stringValue(event.IQPath), stringValue(event.AudioPath)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for index := range l.samples {
		if l.samples[index].EventID == event.ID {
			sample.ID = l.samples[index].ID
			l.samples[index] = sample
			return sample, l.saveLocked()
		}
	}
	l.samples = append(l.samples, sample)
	return sample, l.saveLocked()
}

func (l *SignalLearningLibrary) Similar(event TransmissionEvent, limit int) []ConfirmedSignalSample {
	l.mu.RLock()
	result := append([]ConfirmedSignalSample(nil), l.samples...)
	l.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return learningDistance(result[i], event) < learningDistance(result[j], event) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func learningDistance(sample ConfirmedSignalSample, event TransmissionEvent) float64 {
	frequencyScale := 25e6
	if event.BandwidthHz > 0 {
		frequencyScale = maxFloat(event.BandwidthHz*100, 1e6)
	}
	distance := absFloat(sample.FrequencyHz-event.FrequencyHz) / frequencyScale
	if strings.EqualFold(sample.Modulation, event.Modulation) {
		distance -= 1
	}
	if event.Analysis != nil && sample.Analysis != nil && strings.EqualFold(sample.Analysis.SignalFamily, event.Analysis.SignalFamily) {
		distance -= .75
	}
	return distance
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func (l *SignalLearningLibrary) ExportJSONL() []byte {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var output strings.Builder
	for _, sample := range l.samples {
		data, _ := json.Marshal(map[string]any{"input": map[string]any{"frequencyHz": sample.FrequencyHz, "bandwidthHz": sample.BandwidthHz, "signalDBFS": sample.SignalDBFS, "noiseDBFS": sample.NoiseDBFS, "analysis": sample.Analysis, "decoderMessages": sample.DecoderMessages, "transcript": sample.Transcript}, "label": map[string]any{"modulation": sample.Modulation, "protocol": sample.Protocol, "notes": sample.Notes}})
		output.Write(data)
		output.WriteByte('\n')
	}
	return []byte(output.String())
}

func (l *SignalLearningLibrary) saveLocked() error {
	data, err := json.MarshalIndent(l.samples, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.path, append(data, '\n'), 0o600)
}
