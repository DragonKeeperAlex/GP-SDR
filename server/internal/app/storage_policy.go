package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const gibibyte = int64(1024 * 1024 * 1024)

const defaultCaptureJournalCapBytes = int64(128 * 1024 * 1024)
const maxCaptureJournalCapBytes = int64(1024 * 1024 * 1024)

type StoragePolicy struct {
	AutoCleanup       bool  `json:"autoCleanup"`
	MaxCaptureDays    int   `json:"maxCaptureDays"`
	RecordingCapBytes int64 `json:"recordingCapBytes"`
	IQCapBytes        int64 `json:"iqCapBytes"`
	// CaptureJournalCapBytes bounds raw capture timing diagnostics. It never
	// applies to Mapper findings, event history, profiles, or calibration.
	CaptureJournalCapBytes   int64 `json:"captureJournalCapBytes"`
	AutoRemoveQuarantine     bool  `json:"autoRemoveQuarantine"`
	QuarantineRetentionHours int   `json:"quarantineRetentionHours"`
}

type StorageCleanupResult struct {
	FilesRemoved            int       `json:"filesRemoved"`
	CaptureIntervalsRemoved int       `json:"captureIntervalsRemoved,omitempty"`
	BytesFreed              int64     `json:"bytesFreed"`
	CompletedAt             time.Time `json:"completedAt"`
	LastError               string    `json:"lastError,omitempty"`
}

func defaultStoragePolicy() StoragePolicy {
	return StoragePolicy{AutoCleanup: false, MaxCaptureDays: 30, RecordingCapBytes: 15 * gibibyte, IQCapBytes: 10 * gibibyte, CaptureJournalCapBytes: defaultCaptureJournalCapBytes,
		AutoRemoveQuarantine: true, QuarantineRetentionHours: 24}
}

func loadStoragePolicy(dataDirectory string) StoragePolicy {
	policy := defaultStoragePolicy()
	data, err := os.ReadFile(filepath.Join(dataDirectory, "Data", "storage-policy.json"))
	if err == nil {
		_ = json.Unmarshal(data, &policy)
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			if _, exists := raw["autoRemoveQuarantine"]; !exists {
				policy.AutoRemoveQuarantine = true
			}
			if _, exists := raw["quarantineRetentionHours"]; !exists {
				policy.QuarantineRetentionHours = 24
			}
			if _, exists := raw["captureJournalCapBytes"]; !exists {
				policy.CaptureJournalCapBytes = defaultCaptureJournalCapBytes
			}
		}
	}
	validated, err := validateStoragePolicy(policy)
	if err != nil {
		return defaultStoragePolicy()
	}
	return validated
}

func validateStoragePolicy(policy StoragePolicy) (StoragePolicy, error) {
	if policy.MaxCaptureDays < 0 || policy.MaxCaptureDays > 3650 {
		return policy, errors.New("capture retention must be between zero and 3650 days")
	}
	if policy.AutoRemoveQuarantine && policy.QuarantineRetentionHours == 0 {
		policy.QuarantineRetentionHours = 24
	}
	if policy.QuarantineRetentionHours < 0 || policy.QuarantineRetentionHours > 168 || (policy.AutoRemoveQuarantine && policy.QuarantineRetentionHours < 1) {
		return policy, errors.New("rejected IQ retention must be between one hour and seven days")
	}
	for _, capBytes := range []int64{policy.RecordingCapBytes, policy.IQCapBytes} {
		if capBytes < 0 || capBytes > 2*1024*gibibyte {
			return policy, errors.New("storage caps must be between zero and 2048 GB")
		}
	}
	if policy.CaptureJournalCapBytes < 0 || policy.CaptureJournalCapBytes > maxCaptureJournalCapBytes {
		return policy, errors.New("capture journal cap must be between zero and 1024 MB")
	}
	return policy, nil
}

func saveStoragePolicy(dataDirectory string, policy StoragePolicy) error {
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Join(dataDirectory, "Data")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "storage-policy.json"), encoded, 0o600)
}

func enforceStoragePolicy(dataDirectory string, policy StoragePolicy, now time.Time) StorageCleanupResult {
	// Results and event history are intentionally stored under Data and are not
	// cleanup targets. Only derived media under Recordings and IQ, plus the raw
	// capture timing journal, are eligible. Journal compaction retains newest
	// complete records and never changes Mapper findings or event history.
	// Keep TestStorageCleanupNeverDeletesResultsOrEventHistory as a hard guard
	// if storage layout or retention behavior changes in a future release.
	result := StorageCleanupResult{CompletedAt: now}
	before := directoryBytes(filepath.Join(dataDirectory, "Recordings")) + directoryBytes(filepath.Join(dataDirectory, "IQ"))
	if policy.AutoRemoveQuarantine {
		removed, err := pruneFilesOlderThan(filepath.Join(dataDirectory, "IQ", "Quarantine"), now.Add(-time.Duration(policy.QuarantineRetentionHours)*time.Hour), now)
		result.FilesRemoved += removed
		if err != nil {
			result.LastError = err.Error()
		}
	}
	if policy.MaxCaptureDays > 0 {
		removed, err := pruneExpiredRecordings(dataDirectory, policy.MaxCaptureDays, now)
		result.FilesRemoved += removed
		if err != nil {
			result.LastError = err.Error()
		}
	}
	for _, target := range []struct {
		name string
		cap  int64
	}{{"Recordings", policy.RecordingCapBytes}, {"IQ", policy.IQCapBytes}} {
		removed, err := pruneCaptureRootToCap(filepath.Join(dataDirectory, target.name), target.cap, now)
		result.FilesRemoved += removed
		if err != nil {
			if result.LastError != "" {
				result.LastError += " · "
			}
			result.LastError += err.Error()
		}
	}
	journalRemoved, journalBytesFreed, err := compactCaptureIntervalJournal(dataDirectory, policy.CaptureJournalCapBytes)
	result.CaptureIntervalsRemoved = journalRemoved
	result.BytesFreed += journalBytesFreed
	if err != nil {
		if result.LastError != "" {
			result.LastError += " · "
		}
		result.LastError += err.Error()
	}
	after := directoryBytes(filepath.Join(dataDirectory, "Recordings")) + directoryBytes(filepath.Join(dataDirectory, "IQ"))
	if after < before {
		result.BytesFreed += before - after
	}
	return result
}

// compactCaptureIntervalJournal bounds only the append-only diagnostic journal.
// It is serialized with capture appends and writes a replacement atomically, so
// a power interruption leaves either the old journal or a complete new one.
func compactCaptureIntervalJournal(dataDirectory string, capBytes int64) (int, int64, error) {
	if capBytes <= 0 {
		return 0, 0, nil
	}
	path := filepath.Join(dataDirectory, "Data", "capture-intervals.jsonl")
	archiveMu.Lock()
	defer archiveMu.Unlock()
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, 0, errors.New("capture interval journal is not a regular file")
	}
	if info.Size() <= capBytes {
		return 0, 0, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	} else if len(lines) > 0 {
		// A missing newline means the final append may have been interrupted.
		// Never promote a potentially torn JSON record into the retained journal.
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return 0, 0, errors.New("capture journal contains no complete records")
	}
	keptStart, keptBytes := len(lines), 0
	for keptStart > 0 && int64(keptBytes+len(lines[keptStart-1])) <= capBytes {
		keptStart--
		keptBytes += len(lines[keptStart])
	}
	if keptStart == len(lines) {
		return 0, 0, errors.New("capture journal cap is smaller than a single record")
	}
	compacted := bytes.Join(lines[keptStart:], nil)
	if err := writeBytesAtomic(path, compacted); err != nil {
		return 0, 0, err
	}
	return keptStart, info.Size() - int64(len(compacted)), nil
}

func enforceQuarantinePolicy(dataDirectory string, policy StoragePolicy, now time.Time) StorageCleanupResult {
	result := StorageCleanupResult{CompletedAt: now}
	if !policy.AutoRemoveQuarantine {
		return result
	}
	before := directoryBytes(filepath.Join(dataDirectory, "IQ", "Quarantine"))
	removed, err := pruneFilesOlderThan(filepath.Join(dataDirectory, "IQ", "Quarantine"), now.Add(-time.Duration(policy.QuarantineRetentionHours)*time.Hour), now)
	result.FilesRemoved = removed
	if err != nil {
		result.LastError = err.Error()
	}
	after := directoryBytes(filepath.Join(dataDirectory, "IQ", "Quarantine"))
	if after < before {
		result.BytesFreed = before - after
	}
	return result
}

func pruneFilesOlderThan(root string, cutoff, now time.Time) (int, error) {
	removed := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == "Archive" || entry.Name() == "Pending") {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || strings.HasSuffix(entry.Name(), ".part") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) || info.ModTime().After(now.Add(-10*time.Minute)) {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		removed++
		return nil
	})
	return removed, err
}

type captureFile struct {
	path     string
	size     int64
	modified time.Time
	priority int
}

func pruneCaptureRootToCap(root string, capBytes int64, now time.Time) (int, error) {
	if capBytes <= 0 {
		return 0, nil
	}
	files := make([]captureFile, 0, 256)
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || strings.HasSuffix(entry.Name(), ".part") {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			return infoErr
		}
		total += info.Size()
		priority := 1
		normalized := filepath.ToSlash(path)
		if strings.Contains(normalized, "/Quarantine/") {
			priority = 0
		} else if strings.Contains(normalized, "/Retained/") {
			priority = 2
		} else if strings.Contains(normalized, "/Pending/") {
			priority = 3
		}
		files = append(files, captureFile{path: path, size: info.Size(), modified: info.ModTime(), priority: priority})
		return nil
	})
	if err != nil || total <= capBytes {
		return 0, err
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].priority != files[j].priority {
			return files[i].priority < files[j].priority
		}
		if files[i].modified.Equal(files[j].modified) {
			return files[i].path < files[j].path
		}
		return files[i].modified.Before(files[j].modified)
	})
	removed := 0
	protectedAfter := now.Add(-10 * time.Minute)
	for _, file := range files {
		if total <= capBytes {
			break
		}
		if file.modified.After(protectedAfter) || (file.priority == 3 && file.modified.After(now.Add(-24*time.Hour))) {
			continue
		}
		relative, relErr := filepath.Rel(root, file.path)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		if removeErr := os.Remove(file.path); removeErr != nil {
			return removed, removeErr
		}
		total -= file.size
		removed++
	}
	return removed, nil
}
