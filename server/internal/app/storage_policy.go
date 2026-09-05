package app

import (
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

type StoragePolicy struct {
	AutoCleanup              bool  `json:"autoCleanup"`
	MaxCaptureDays           int   `json:"maxCaptureDays"`
	RecordingCapBytes        int64 `json:"recordingCapBytes"`
	IQCapBytes               int64 `json:"iqCapBytes"`
	AutoRemoveQuarantine     bool  `json:"autoRemoveQuarantine"`
	QuarantineRetentionHours int   `json:"quarantineRetentionHours"`
}

type StorageCleanupResult struct {
	FilesRemoved int       `json:"filesRemoved"`
	BytesFreed   int64     `json:"bytesFreed"`
	CompletedAt  time.Time `json:"completedAt"`
	LastError    string    `json:"lastError,omitempty"`
}

func defaultStoragePolicy() StoragePolicy {
	return StoragePolicy{AutoCleanup: false, MaxCaptureDays: 30, RecordingCapBytes: 15 * gibibyte, IQCapBytes: 10 * gibibyte,
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
		_, err := pruneExpiredRecordings(dataDirectory, policy.MaxCaptureDays, now)
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
	after := directoryBytes(filepath.Join(dataDirectory, "Recordings")) + directoryBytes(filepath.Join(dataDirectory, "IQ"))
	if after < before {
		result.BytesFreed = before - after
	}
	return result
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
