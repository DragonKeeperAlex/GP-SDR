package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteIQEvidenceIncludesSidecarMetadata(t *testing.T) {
	directory := t.TempDir()
	path, err := writeIQEvidence(directory, 155_250_000, CaptureSpec{CenterFrequencyHz: 155_500_000, SampleRateHz: 2_000_000}, ComplexSigned8, []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(path) != ".cs8" {
		t.Fatalf("unexpected IQ path: %s", path)
	}
	if _, err := os.Stat(path[:len(path)-4] + ".json"); err != nil {
		t.Fatal("missing IQ metadata sidecar:", err)
	}
}

func TestWriteIQEvidenceUsesUnsignedExtensionForRTLSDR(t *testing.T) {
	path, err := writeIQEvidence(t.TempDir(), 433_920_000, CaptureSpec{CenterFrequencyHz: 433_920_000, SampleRateHz: 250_000}, ComplexUnsigned8, []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(path) != ".cu8" {
		t.Fatalf("unexpected unsigned IQ path: %s", path)
	}
}

func TestPruneExpiredRecordingsIsBoundedToDatedCaptureFolders(t *testing.T) {
	directory := t.TempDir()
	for _, relative := range []string{"Recordings/2026-01-01/old.wav", "IQ/2026-08-19/new.cs8", "Recordings/keep/readme.txt"} {
		path := filepath.Join(directory, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := pruneExpiredRecordings(directory, 30, time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local))
	if err != nil || removed != 1 {
		t.Fatalf("unexpected prune result: removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "Recordings", "keep", "readme.txt")); err != nil {
		t.Fatal("non-date folder must be preserved")
	}
	if _, err := os.Stat(filepath.Join(directory, "IQ", "2026-08-19", "new.cs8")); err != nil {
		t.Fatal("recent IQ evidence must be preserved")
	}
}
