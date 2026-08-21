package app

import (
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
	r.mu.Unlock()
	go func() {
		status := calculateStorageStatus(directory)
		r.mu.Lock()
		r.storage = status
		r.storageRefreshing = false
		r.mu.Unlock()
	}()
}

func calculateStorageStatus(dataDirectory string) StorageStatus {
	status := StorageStatus{CheckedAt: time.Now()}
	status.JournalBytes = directoryBytes(filepath.Join(dataDirectory, "Data"))
	status.RecordingBytes = directoryBytes(filepath.Join(dataDirectory, "Recordings"))
	status.IQBytes = directoryBytes(filepath.Join(dataDirectory, "IQ"))
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
