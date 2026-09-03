package app

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"
)

func (r *Runtime) refreshStorageStatus() {
	r.mu.Lock()
	if r.storageRefreshing || (!r.storage.CheckedAt.IsZero() && time.Since(r.storage.CheckedAt) < 30*time.Second) {
		r.mu.Unlock()
		return
	}
	r.storageRefreshing = true
	directory := r.dataDirectory
	policy := r.storagePolicy
	lastCleanup := r.storageCleanup
	pruning := r.storagePruning
	r.mu.Unlock()
	go func() {
		if (policy.AutoCleanup || policy.AutoRemoveQuarantine) && !pruning && (lastCleanup.CompletedAt.IsZero() || time.Since(lastCleanup.CompletedAt) >= 15*time.Minute) {
			r.mu.Lock()
			r.storagePruning = true
			r.mu.Unlock()
			if policy.AutoCleanup {
				lastCleanup = enforceStoragePolicy(directory, policy, time.Now())
			} else {
				lastCleanup = enforceQuarantinePolicy(directory, policy, time.Now())
			}
			r.mu.Lock()
			r.storageCleanup = lastCleanup
			r.storagePruning = false
			r.mu.Unlock()
		}
		status := calculateStorageStatus(directory)
		r.mu.Lock()
		status.Policy = r.storagePolicy
		status.LastCleanup = r.storageCleanup
		status.CleanupRunning = r.storagePruning
		r.storage = status
		r.storageRefreshing = false
		r.mu.Unlock()
	}()
}

func (r *Runtime) UpdateStoragePolicy(policy StoragePolicy) (StorageStatus, error) {
	validated, err := validateStoragePolicy(policy)
	if err != nil {
		return r.StorageStatus(), err
	}
	if err := saveStoragePolicy(r.dataDirectory, validated); err != nil {
		return r.StorageStatus(), err
	}
	r.mu.Lock()
	r.storagePolicy = validated
	r.storage.Policy = validated
	r.mu.Unlock()
	return r.StorageStatus(), nil
}

func (r *Runtime) StorageStatus() StorageStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := r.storage
	status.Policy = r.storagePolicy
	status.LastCleanup = r.storageCleanup
	status.CleanupRunning = r.storagePruning
	return status
}

func (r *Runtime) CleanStorageNow() (StorageStatus, error) {
	r.mu.Lock()
	if r.storagePruning {
		r.mu.Unlock()
		return r.StorageStatus(), errors.New("storage cleanup is already running")
	}
	r.storagePruning = true
	policy, directory := r.storagePolicy, r.dataDirectory
	r.mu.Unlock()
	result := enforceStoragePolicy(directory, policy, time.Now())
	status := calculateStorageStatus(directory)
	r.mu.Lock()
	r.storageCleanup = result
	r.storagePruning = false
	status.Policy = policy
	status.LastCleanup = result
	r.storage = status
	r.mu.Unlock()
	if result.LastError != "" {
		return status, errors.New(result.LastError)
	}
	return status, nil
}

func calculateStorageStatus(dataDirectory string) StorageStatus {
	status := StorageStatus{CheckedAt: time.Now()}
	status.JournalBytes = directoryBytes(filepath.Join(dataDirectory, "Data"))
	status.RecordingBytes = directoryBytes(filepath.Join(dataDirectory, "Recordings"))
	status.IQBytes = directoryBytes(filepath.Join(dataDirectory, "IQ"))
	status.IQArchiveBytes = directoryBytes(filepath.Join(dataDirectory, "IQ", "Archive"))
	status.IQPendingBytes = directoryBytes(filepath.Join(dataDirectory, "IQ", "Pending"))
	status.IQRetainedBytes = directoryBytes(filepath.Join(dataDirectory, "IQ", "Retained"))
	status.IQQuarantineBytes = directoryBytes(filepath.Join(dataDirectory, "IQ", "Quarantine"))
	status.ProfileBytes = directoryBytes(filepath.Join(dataDirectory, "Profiles"))
	status.TotalBytes = status.JournalBytes + status.RecordingBytes + status.IQBytes + status.ProfileBytes
	return status
}

func directoryBytes(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
