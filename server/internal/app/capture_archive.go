package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Original captures are immutable and shared by all event records in a batch.
// The mutex makes capacity checks and manifest appends safe across receivers.
var archiveMu sync.Mutex
var archiveSizes = map[string]int64{}

type CaptureInterval struct {
	ReceiverSettings  CaptureSpec `json:"receiverSettings"`
	ID                string      `json:"id"`
	JobID             string      `json:"jobID"`
	DeviceID          string      `json:"deviceID"`
	RequestedAt       time.Time   `json:"requestedAt"`
	ReceivedAt        time.Time   `json:"receivedAt"`
	SampleSeconds     float64     `json:"sampleSeconds"`
	CenterFrequencyHz int64       `json:"centerFrequencyHz"`
	SampleRateHz      int         `json:"sampleRateHz"`
	SampleBytes       int         `json:"sampleBytes"`
	FrequenciesHz     []float64   `json:"frequenciesHz"`
	IQPath            string      `json:"iqPath,omitempty"`
	SHA256            string      `json:"sha256,omitempty"`
	Error             string      `json:"error,omitempty"`
}

func writeDurableFile(path string, data []byte) error {
	f, err := os.OpenFile(path+".part", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	} // Keep partial evidence for diagnosis/recovery.
	return os.Rename(path+".part", path)
}

func writeCaptureInterval(root string, interval *CaptureInterval, spec CaptureSpec, format SampleFormat, data []byte, archive bool, capBytes int64) error {
	archiveMu.Lock()
	defer archiveMu.Unlock()
	interval.ReceiverSettings = spec
	if archive && interval.Error == "" {
		archiveRoot := filepath.Join(root, "IQ", "Archive")
		budgetRoot := filepath.Join(root, "IQ")
		used, ok := archiveSizes[budgetRoot]
		if !ok {
			used = directoryBytes(budgetRoot)
			archiveSizes[budgetRoot] = used
		}
		if capBytes > 0 && used+int64(len(data))+4096 > capBytes {
			return errors.New("original IQ archive reached its capacity; capture stopped without deleting archived samples")
		}
		dir := filepath.Join(archiveRoot, interval.ReceivedAt.UTC().Format("2006-01-02"))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		ext := ".cs8"
		if format == ComplexUnsigned8 {
			ext = ".cu8"
		}
		path := filepath.Join(dir, interval.ID+ext)
		if err := writeDurableFile(path, data); err != nil {
			return err
		}
		archiveSizes[budgetRoot] += int64(len(data)) + 4096
		sum := sha256.Sum256(data)
		interval.IQPath, interval.SHA256 = path, hex.EncodeToString(sum[:])
		metadata := IQCaptureMetadata{FrequencyHz: float64(spec.CenterFrequencyHz), CenterFrequencyHz: spec.CenterFrequencyHz,
			SampleRateHz: spec.SampleRateHz, Format: format, CapturedAt: interval.ReceivedAt.Add(-time.Duration(interval.SampleSeconds * float64(time.Second))),
			SampleBytes: len(data), OriginalRateHz: spec.SampleRateHz, OriginalBytes: len(data), LifecycleStatus: "archived-original",
			CaptureID: interval.ID, SHA256: interval.SHA256, Valuable: true, ValueReasons: []string{"Original receiver bytes; shared batch capture"}}
		encoded, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return err
		}
		if err := writeDurableFile(iqMetadataPath(path), encoded); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "Data"), 0700); err != nil {
		return err
	}
	return appendDurableJSON(filepath.Join(root, "Data", "capture-intervals.jsonl"), interval)
}

func (r *Runtime) captureFailure(run *mapperRunContext, err error) bool {
	message := fmt.Sprintf("Capture stopped: %v", err)
	r.setRuntimeError(message)
	if run != nil && r.mapper != nil {
		r.mapper.SetJobError(run.JobID, message)
	}
	return false
}
