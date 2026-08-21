package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageStatusOnlyCountsGPSDROwnedRoots(t *testing.T) {
	directory := t.TempDir()
	files := map[string]int{
		"Data/events.jsonl": 11, "Recordings/2026-08-21/call.wav": 12,
		"IQ/2026-08-21/frame.cs8": 13, "Profiles/local.json": 14, "unrelated.bin": 100,
	}
	for relative, size := range files {
		path := filepath.Join(directory, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	status := calculateStorageStatus(directory)
	if status.JournalBytes != 11 || status.RecordingBytes != 12 || status.IQBytes != 13 || status.ProfileBytes != 14 || status.TotalBytes != 50 {
		t.Fatalf("unexpected storage status: %+v", status)
	}
}
