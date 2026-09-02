package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestCompactIQEvidenceReducesWidebandCapture(t *testing.T) {
	data := make([]byte, 8_000_000*2/10) // 100 ms at 8 MS/s
	for index := range data {
		data[index] = byte(index)
	}
	spec := CaptureSpec{CenterFrequencyHz: 450_000_000, SampleRateHz: 8_000_000}
	compacted, resultSpec, format := compactIQEvidence(data, ComplexSigned8, spec, 452_000_000, 12_500)
	if format != ComplexUnsigned8 || resultSpec.CenterFrequencyHz != 452_000_000 || resultSpec.SampleRateHz != 250_000 {
		t.Fatalf("unexpected compacted IQ description: format=%s spec=%+v", format, resultSpec)
	}
	if len(compacted) >= len(data)/10 {
		t.Fatalf("expected major evidence reduction, got %d bytes from %d", len(compacted), len(data))
	}
}

func TestFinalizeIQEvidenceRetainsUsefulAndQuarantinesJunk(t *testing.T) {
	root := t.TempDir()
	usefulPath, err := writeIQEvidence(root, 155_25e4, CaptureSpec{CenterFrequencyHz: 155_25e4, SampleRateHz: 250_000}, ComplexUnsigned8, []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	protocol := "P25"
	retained, metadata, err := finalizeIQEvidence(usefulPath, TransmissionEvent{ID: "useful", ProtocolName: &protocol})
	if err != nil || !metadata.Valuable || !strings.Contains(filepath.ToSlash(retained), "/IQ/Retained/") {
		t.Fatalf("useful evidence was not retained: path=%s metadata=%+v err=%v", retained, metadata, err)
	}
	junkPath, err := writeIQEvidence(root, 456e6, CaptureSpec{CenterFrequencyHz: 456e6, SampleRateHz: 250_000}, ComplexUnsigned8, []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	quarantined, metadata, err := finalizeIQEvidence(junkPath, TransmissionEvent{ID: "junk", SignalDBFS: -70, NoiseDBFS: -72})
	if err != nil || metadata.Valuable || !strings.Contains(filepath.ToSlash(quarantined), "/IQ/Quarantine/") {
		t.Fatalf("junk evidence was not quarantined: path=%s metadata=%+v err=%v", quarantined, metadata, err)
	}
}

func TestFinalizeIQEvidenceDeletesRejectedWhenRequested(t *testing.T) {
	root := t.TempDir()
	path, err := writeIQEvidence(root, 456e6, CaptureSpec{CenterFrequencyHz: 456e6, SampleRateHz: 250_000}, ComplexUnsigned8, []byte{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	deleted, metadata, err := finalizeIQEvidence(path, TransmissionEvent{ID: "delete", SignalDBFS: -70, NoiseDBFS: -72, IQRetentionPolicy: "delete"})
	if err != nil || deleted != "" || metadata.LifecycleStatus != "deleted-low-value" {
		t.Fatalf("expected rejected IQ deletion, got path=%q metadata=%+v error=%v", deleted, metadata, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("IQ file still exists: %v", err)
	}
	if _, err := os.Stat(iqMetadataPath(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("metadata file still exists: %v", err)
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
