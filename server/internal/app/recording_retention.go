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
	CaptureID         string       `json:"captureID,omitempty"`
	SHA256            string       `json:"sha256,omitempty"`
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
	archiveMu.Lock()
	defer archiveMu.Unlock()
	if len(data) == 0 {
		return "", errors.New("IQ evidence is empty")
	}
	now := time.Now().UTC()
	directory := filepath.Join(dataDirectory, "IQ", "Pending", now.Format("2006-01-02"))
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s-%.0f-%s", now.Format("20060102T150405.000Z"), frequencyHz, NewID())
	extension := ".cs8"
	if format == ComplexUnsigned8 {
		extension = ".cu8"
	}
	path := filepath.Join(directory, base+extension)
	if err := writeDurableFile(path, data); err != nil {
		return "", err
	}
	metadata := IQCaptureMetadata{FrequencyHz: frequencyHz, CenterFrequencyHz: spec.CenterFrequencyHz, SampleRateHz: spec.SampleRateHz,
		Format: format, CapturedAt: now, SampleBytes: len(data), LifecycleStatus: "pending-analysis"}
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := writeDurableFile(filepath.Join(directory, base+".json"), encoded); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if _, ok := archiveSizes[filepath.Join(dataDirectory, "IQ")]; ok {
		archiveSizes[filepath.Join(dataDirectory, "IQ")] += int64(len(data)) + 4096
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
	// Windowed-sinc FIR before decimation. A box average has weak rejection
	// and folds adjacent signals into the retained channel. Filter at the
	// output instants (polyphase decimation), retaining the full duration.
	output := make([]byte, outputSamples*2)
	half := 16 * decimation
	taps := 2*half + 1
	coefficients := make([]complex128, taps)
	weights := make([]float64, taps)
	cutoff := .4 / float64(decimation)
	offset := frequencyHz - float64(spec.CenterFrequencyHz)
	sum := 0.0
	for k := range weights {
		x := float64(k - half)
		h := 2 * cutoff
		if x != 0 {
			h = math.Sin(2*math.Pi*cutoff*x) / (math.Pi * x)
		}
		w := .42 - .5*math.Cos(2*math.Pi*float64(k)/float64(taps-1)) + .08*math.Cos(4*math.Pi*float64(k)/float64(taps-1))
		weights[k] = h * w
		sum += weights[k]
	}
	for k, weight := range weights {
		phase := -2 * math.Pi * offset * float64(k-half) / float64(spec.SampleRateHz)
		coefficients[k] = complex(math.Cos(phase), math.Sin(phase)) * complex(weight/sum, 0)
	}
	stepPhase := -2 * math.Pi * offset * float64(decimation) / float64(spec.SampleRateHz)
	rotation, step := complex(1.0, 0), complex(math.Cos(stepPhase), math.Sin(stepPhase))
	for n := 0; n < outputSamples; n++ {
		center := n * decimation
		value := complex(0.0, 0)
		for k, coefficient := range coefficients {
			index := center + k - half
			if index < 0 || index >= inputSamples {
				continue
			}
			i, q := decoderIQPair(data, index, format)
			value += complex(i, q) * coefficient
		}
		value *= rotation
		output[2*n] = byte(clampInt(int(math.Round(real(value)+127.5)), 0, 255))
		output[2*n+1] = byte(clampInt(int(math.Round(imag(value)+127.5)), 0, 255))
		rotation *= step
		if n&4095 == 0 {
			norm := math.Hypot(real(rotation), imag(rotation))
			if norm > 0 {
				rotation /= complex(norm, 0)
			}
		}
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
	if metadata.LifecycleStatus == "archived-original" {
		return iqPath, metadata, nil
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
	if !valuable && strings.EqualFold(event.IQRetentionPolicy, "delete") {
		if err := os.Remove(iqPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return iqPath, metadata, err
		}
		if err := os.Remove(metadataPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return iqPath, metadata, err
		}
		metadata.LifecycleStatus = "deleted-low-value"
		metadata.AnalysisCompleted, metadata.Valuable, metadata.ValueReasons, metadata.EventID = ptr(time.Now().UTC()), false, reasons, event.ID
		return "", metadata, nil
	}
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
	for _, rootName := range []string{"Recordings", "IQ", filepath.Join("IQ", "Retained"), filepath.Join("IQ", "Quarantine")} {
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
