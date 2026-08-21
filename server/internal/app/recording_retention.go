package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type IQCaptureMetadata struct {
	FrequencyHz       float64      `json:"frequencyHz"`
	CenterFrequencyHz int64        `json:"centerFrequencyHz"`
	SampleRateHz      int          `json:"sampleRateHz"`
	Format            SampleFormat `json:"format"`
	CapturedAt        time.Time    `json:"capturedAt"`
	SampleBytes       int          `json:"sampleBytes"`
}

func writeIQEvidence(dataDirectory string, frequencyHz float64, spec CaptureSpec, format SampleFormat, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("IQ evidence is empty")
	}
	now := time.Now().UTC()
	directory := filepath.Join(dataDirectory, "IQ", now.Format("2006-01-02"))
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
		Format: format, CapturedAt: now, SampleBytes: len(data)}
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

// pruneExpiredRecordings only considers strict YYYY-MM-DD children below the
// two GP-SDR-owned capture roots. It never traverses arbitrary user paths.
func pruneExpiredRecordings(dataDirectory string, maxDays int, now time.Time) (int, error) {
	if maxDays <= 0 {
		return 0, nil
	}
	cutoff := now.AddDate(0, 0, -maxDays)
	removed := 0
	var joined error
	for _, rootName := range []string{"Recordings", "IQ"} {
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
