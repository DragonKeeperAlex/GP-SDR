package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type IQCaptureMetadata struct {
	FrequencyHz       float64      `json:"frequencyHz"`
	CenterFrequencyHz int64        `json:"centerFrequencyHz"`
	SampleRateHz      int          `json:"sampleRateHz"`
	Format            SampleFormat `json:"format"`
	CapturedAt        time.Time    `json:"capturedAt"`
	SampleBytes       int          `json:"sampleBytes"`
	OriginalRateHz    int          `json:"originalRateHz,omitempty"`
	OriginalBytes     int          `json:"originalBytes,omitempty"`
	LifecycleStatus   string       `json:"lifecycleStatus,omitempty"`
	AnalysisCompleted *time.Time   `json:"analysisCompletedAt,omitempty"`
	Valuable          bool         `json:"valuable"`
	ValueReasons      []string     `json:"valueReasons,omitempty"`
	EventID           string       `json:"eventID,omitempty"`
}

func writeIQEvidence(dataDirectory string, frequencyHz float64, spec CaptureSpec, format SampleFormat, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("IQ evidence is empty")
	}
	now := time.Now().UTC()
	directory := filepath.Join(dataDirectory, "IQ", "Pending", now.Format("2006-01-02"))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s-%.0f", now.Format("20060102T150405.000Z"), frequencyHz)
	extension := ".cs8"
	if format == ComplexUnsigned8 {
		extension = ".cu8"
	}
	path := filepath.Join(directory, base+extension)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	metadata := IQCaptureMetadata{FrequencyHz: frequencyHz, CenterFrequencyHz: spec.CenterFrequencyHz, SampleRateHz: spec.SampleRateHz,
		Format: format, CapturedAt: now, SampleBytes: len(data), LifecycleStatus: "pending-analysis"}
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(directory, base+".json"), encoded, 0o644); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// compactIQEvidence shifts one channel to baseband and resamples it before it
// is written. Mapper may capture 8-20 MS/s to inspect many software VFOs at
// once, but an individual narrowband channel does not need that full rate in
// its evidence file. The output is decoder-friendly unsigned complex IQ.
func compactIQEvidence(data []byte, format SampleFormat, spec CaptureSpec, frequencyHz, bandwidthHz float64) ([]byte, CaptureSpec, SampleFormat) {
	if len(data) < 2 || spec.SampleRateHz <= 0 {
		return data, spec, format
	}
	targetRate := int(math.Ceil(math.Max(250_000, bandwidthHz*8)/50_000) * 50_000)
	if targetRate > 2_400_000 {
		targetRate = 2_400_000
	}
	if targetRate >= spec.SampleRateHz {
		return data, spec, format
	}
	inputSamples := len(data) / 2
	decimation := maxInt(1, spec.SampleRateHz/targetRate)
	actualRate := spec.SampleRateHz / decimation
	outputSamples := inputSamples / decimation
	if outputSamples < 1 {
		return data, spec, format
	}
	output := make([]byte, outputSamples*2)
	offsetHz := frequencyHz - float64(spec.CenterFrequencyHz)
	phaseStep := -2 * math.Pi * offsetHz / float64(spec.SampleRateHz)
	oscillatorI, oscillatorQ := 1.0, 0.0
	stepI, stepQ := math.Cos(phaseStep), math.Sin(phaseStep)
	accumulatorI, accumulatorQ, count, outputIndex := 0.0, 0.0, 0, 0
	for sampleIndex := 0; sampleIndex < inputSamples && outputIndex < outputSamples; sampleIndex++ {
		iValue, qValue := decoderIQPair(data, sampleIndex, format)
		accumulatorI += iValue*oscillatorI - qValue*oscillatorQ
		accumulatorQ += iValue*oscillatorQ + qValue*oscillatorI
		nextI := oscillatorI*stepI - oscillatorQ*stepQ
		oscillatorQ = oscillatorI*stepQ + oscillatorQ*stepI
		oscillatorI = nextI
		if sampleIndex&4095 == 0 {
			norm := math.Hypot(oscillatorI, oscillatorQ)
			if norm > 0 {
				oscillatorI, oscillatorQ = oscillatorI/norm, oscillatorQ/norm
			}
		}
		count++
		if count < decimation {
			continue
		}
		output[outputIndex*2] = byte(clampInt(int(math.Round(accumulatorI/float64(count)+127.5)), 0, 255))
		output[outputIndex*2+1] = byte(clampInt(int(math.Round(accumulatorQ/float64(count)+127.5)), 0, 255))
		outputIndex++
		accumulatorI, accumulatorQ, count = 0, 0, 0
	}
	compacted := spec
	compacted.CenterFrequencyHz = int64(math.Round(frequencyHz))
	compacted.SampleRateHz = actualRate
	return output, compacted, ComplexUnsigned8
}

func iqMetadataPath(iqPath string) string {
	extension := filepath.Ext(iqPath)
	return strings.TrimSuffix(iqPath, extension) + ".json"
}

func setIQCaptureOrigin(iqPath string, originalRateHz, originalBytes int) error {
	metadataPath := iqMetadataPath(iqPath)
	encoded, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}
	var metadata IQCaptureMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return err
	}
	metadata.OriginalRateHz, metadata.OriginalBytes = originalRateHz, originalBytes
	updated, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath, updated, 0o644)
}

// finalizeIQEvidence runs only after local classification, optional protocol
// decoding, and optional transcription have finished. Useful evidence moves to
// Retained; low-value captures move to a recoverable Quarantine where storage
// policy can remove them later.
func finalizeIQEvidence(iqPath string, event TransmissionEvent) (string, IQCaptureMetadata, error) {
	metadataPath := iqMetadataPath(iqPath)
	encoded, err := os.ReadFile(metadataPath)
	if err != nil {
		return iqPath, IQCaptureMetadata{}, err
	}
	var metadata IQCaptureMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return iqPath, metadata, err
	}
	reasons := make([]string, 0, 4)
	if len(event.DecoderMessages) > 0 {
		reasons = append(reasons, "valid decoder output")
	}
	if event.Transcript != nil && len(strings.TrimSpace(*event.Transcript)) >= 8 {
		reasons = append(reasons, "speech transcript")
	}
	if len(event.Callsigns) > 0 {
		reasons = append(reasons, "callsign evidence")
	}
	protocol := strings.ToLower(strings.TrimSpace(stringValue(event.ProtocolName)))
	if protocol != "" && !strings.Contains(protocol, "candidate") && protocol != "unknown" {
		reasons = append(reasons, "identified protocol")
	}
	if event.Analysis != nil && event.Analysis.Confidence >= .70 && event.Analysis.Modulation != "UNKNOWN" && event.Analysis.Modulation != "CARRIER" {
		reasons = append(reasons, "confident local waveform classification")
	}
	if event.SignalDBFS-event.NoiseDBFS >= 10 {
		reasons = append(reasons, "strong signal evidence")
	}
	valuable := len(reasons) > 0
	destinationClass := "Quarantine"
	metadata.LifecycleStatus = "quarantined-low-value"
	if valuable {
		destinationClass = "Retained"
		metadata.LifecycleStatus = "retained"
	}
	now := time.Now().UTC()
	metadata.AnalysisCompleted, metadata.Valuable, metadata.ValueReasons, metadata.EventID = &now, valuable, reasons, event.ID
	destinationDirectory := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(iqPath))), destinationClass, metadata.CapturedAt.Format("2006-01-02"))
	if err := os.MkdirAll(destinationDirectory, 0o755); err != nil {
		return iqPath, metadata, err
	}
	destinationIQ := filepath.Join(destinationDirectory, filepath.Base(iqPath))
	destinationMetadata := filepath.Join(destinationDirectory, filepath.Base(metadataPath))
	updated, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return iqPath, metadata, err
	}
	if err := os.WriteFile(metadataPath, updated, 0o644); err != nil {
		return iqPath, metadata, err
	}
	if err := os.Rename(iqPath, destinationIQ); err != nil {
		return iqPath, metadata, err
	}
	if err := os.Rename(metadataPath, destinationMetadata); err != nil {
		_ = os.Rename(destinationIQ, iqPath)
		return iqPath, metadata, err
	}
	return destinationIQ, metadata, nil
}

// pruneExpiredRecordings only considers strict YYYY-MM-DD children below the
// two GP-SDR-owned capture roots. It never traverses arbitrary user paths.
func pruneExpiredRecordings(dataDirectory string, maxDays int, now time.Time) (int, error) {
	if maxDays <= 0 {
		return 0, nil
	}
	cutoff := now.AddDate(0, 0, -maxDays)
	removed := 0
	var joined error
	for _, rootName := range []string{"Recordings", "IQ", filepath.Join("IQ", "Pending"), filepath.Join("IQ", "Retained"), filepath.Join("IQ", "Quarantine")} {
		root := filepath.Join(dataDirectory, rootName)
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			date, err := time.ParseInLocation("2006-01-02", entry.Name(), now.Location())
			if err != nil || !date.Before(time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, cutoff.Location())) {
				continue
			}
			if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
				joined = errors.Join(joined, err)
			} else {
				removed++
			}
		}
	}
	return removed, joined
}
