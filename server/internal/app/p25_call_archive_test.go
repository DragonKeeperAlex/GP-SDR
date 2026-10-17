package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestP25CallArchiveSavesOnlyItsStream(t *testing.T) {
	a := newP25CallArchive(1)
	a.Append(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{1, 2}})
	a.Append(AudioFrame{ChannelID: "p25-stream-1", SampleRate: 8000, Samples: []int16{3, 4, 5}})
	path, err := a.Save(t.TempDir(), 1201, time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	samples, rate, err := readPCM16WAV(path)
	if err != nil || rate != 8000 || len(samples) != 3 {
		t.Fatalf("archive=%q rate=%d samples=%v err=%v", path, rate, samples, err)
	}
	if filepath.Ext(path) != ".wav" {
		t.Fatalf("unexpected path %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestP25ArchiveConcurrentNamesDurationAndStorageFailure(t *testing.T) {
	root := t.TempDir()
	started := time.Now()
	a, b := newP25CallArchive(0), newP25CallArchive(1)
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: make([]int16, 8000)})
	b.Append(AudioFrame{ChannelID: b.streamID, SampleRate: 8000, Samples: make([]int16, 16000)})
	left, err := a.Save(root, 1201, started)
	if err != nil {
		t.Fatal(err)
	}
	right, err := b.Save(root, 1201, started)
	if err != nil {
		t.Fatal(err)
	}
	if left == right || a.DurationSeconds() != 1 || b.DurationSeconds() != 2 {
		t.Fatal("collision or incorrect PCM duration")
	}
	blocked := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Save(blocked, 1201, started); err == nil {
		t.Fatal("storage failure ignored")
	}
	if data, _ := os.ReadFile(blocked); string(data) != "preserve" {
		t.Fatal("unrelated file damaged")
	}
}
