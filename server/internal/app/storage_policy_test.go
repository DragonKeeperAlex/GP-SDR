package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoragePolicyCleanupIsBoundedAndOldestFirst(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.Local)
	files := []struct {
		path string
		size int
		age  time.Duration
	}{
		{"Recordings/2026-08-20/old-a.wav", 10, 48 * time.Hour},
		{"Recordings/2026-08-21/old-b.wav", 10, 24 * time.Hour},
		{"Recordings/2026-08-22/active.wav", 10, 2 * time.Minute},
		{"Profiles/keep.json", 50, 48 * time.Hour},
	}
	for _, file := range files {
		path := filepath.Join(root, file.path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, file.size), 0o644); err != nil {
			t.Fatal(err)
		}
		modified := now.Add(-file.age)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	result := enforceStoragePolicy(root, StoragePolicy{RecordingCapBytes: 15, IQCapBytes: 1_000, MaxCaptureDays: 0}, now)
	if result.FilesRemoved != 2 || result.BytesFreed != 20 {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
	for _, path := range []string{"Recordings/2026-08-22/active.wav", "Profiles/keep.json"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("cleanup removed protected data %s: %v", path, err)
		}
	}
}

func TestStoragePolicyRoundTripAndValidation(t *testing.T) {
	root := t.TempDir()
	want := StoragePolicy{AutoCleanup: true, MaxCaptureDays: 14, RecordingCapBytes: 8 * gibibyte, IQCapBytes: 4 * gibibyte}
	if err := saveStoragePolicy(root, want); err != nil {
		t.Fatal(err)
	}
	if got := loadStoragePolicy(root); got != want {
		t.Fatalf("storage policy did not round trip: got %+v want %+v", got, want)
	}
	if _, err := validateStoragePolicy(StoragePolicy{MaxCaptureDays: -1}); err == nil {
		t.Fatal("negative retention should be rejected")
	}
}
