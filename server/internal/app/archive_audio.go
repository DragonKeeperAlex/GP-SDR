package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Derivatives never replace the shared original IQ. Expensive per-channel work
// happens here for manual/after-job archives, after the receiver has moved on.
func deriveStoredAudio(root string, event TransmissionEvent) (string, SignalIntelligence, error) {
	var metadata IQCaptureMetadata
	encoded, err := os.ReadFile(iqMetadataPath(stringValue(event.IQPath)))
	if err != nil {
		return "", SignalIntelligence{}, err
	}
	if err = json.Unmarshal(encoded, &metadata); err != nil {
		return "", SignalIntelligence{}, err
	}
	data, err := os.ReadFile(stringValue(event.IQPath))
	if err != nil {
		return "", SignalIntelligence{}, err
	}
	if len(data) != metadata.SampleBytes || metadata.SampleRateHz <= 0 {
		return "", SignalIntelligence{}, fmt.Errorf("IQ sample length or rate does not match its metadata")
	}
	bandwidth := event.BandwidthHz
	if bandwidth <= 0 {
		bandwidth = 12500
	}
	spec := CaptureSpec{SampleRateHz: metadata.SampleRateHz, CenterFrequencyHz: metadata.CenterFrequencyHz}
	analysis := AnalyzeSignalIQ(data, metadata.Format, spec.SampleRateHz, event.FrequencyHz-float64(spec.CenterFrequencyHz), bandwidth)
	mode := strings.ToLower(event.Modulation)
	if mode != "am" && mode != "nfm" && mode != "wfm" && mode != "fm" {
		mode = strings.ToLower(analysis.Modulation)
	}
	if mode != "am" && mode != "wfm" && mode != "fm" {
		mode = "nfm"
	}
	if mode == "wfm" || mode == "fm" {
		bandwidth = 200000
	}
	filtered, channelSpec, format := compactIQEvidence(data, metadata.Format, spec, event.FrequencyHz, bandwidth)
	result, err := DemodulateIQ(filtered, format, channelSpec.SampleRateHz, event.FrequencyHz-float64(channelSpec.CenterFrequencyHz), mode)
	if err != nil {
		return "", analysis, err
	}
	at := event.StartedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	// IDs are not used as paths; a fresh generated identifier prevents traversal
	// from imported history and avoids collisions across concurrent workers.
	path := filepath.Join(root, "Recordings", at.Format("2006-01-02"), fmt.Sprintf("%s-%.0f-%s-%s.wav", at.Format("20060102T150405.000Z"), event.FrequencyHz, mode, NewID()))
	if err := WriteMonoWAV(path, result.Audio, result.AudioRateHz); err != nil {
		return "", analysis, err
	}
	return path, analysis, nil
}
