package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func pcmFixtureTime() time.Time { return time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC) }

func pcmRecorderFixture(t *testing.T) (*p25PCMRecorder, []P25ActiveCall) {
	t.Helper()
	root := t.TempDir()
	events, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{dataDirectory: root, Events: events}
	profile := ScanProfile{}
	profile.Settings.P25ArchiveCalls = true
	p := newP25PCMRecorder(r, profile, make(chan struct{}))
	var calls []P25ActiveCall
	for i := 0; i < 3; i++ {
		calls = append(calls, P25ActiveCall{StreamIndex: i, DeviceSerial: fmt.Sprintf("fixture-radio-%d", i), TDMASlot: ptr(i % 2), Following: true, StartedAt: pcmFixtureTime().Add(-time.Second), LastHeardAt: pcmFixtureTime(), Grant: p25GrantState{System: "Synthetic system", GroupID: uint32(100 + i), SourceID: uint32(200 + i), FrequencyHz: uint32(770000000 + i*12500)}})
	}
	p.setCalls(calls)
	return p, calls
}

func pcmPacket(index int, sequence uint64, value int16) AudioFrame {
	samples := make([]int16, 800)
	for i := range samples {
		samples[i] = value
	}
	return AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", index), SampleRate: 8000, Sequence: sequence, ReceivedAt: pcmFixtureTime().Add(20 * time.Millisecond), Samples: samples}
}

func pcmMarker(index int, sequence uint64, control string) AudioFrame {
	return AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", index), SampleRate: 8000, Sequence: sequence, ReceivedAt: pcmFixtureTime().Add(20 * time.Millisecond), Control: control}
}

func TestP25PCMRecorderThreeInterleavedStreamsAndBackToBackSegments(t *testing.T) {
	p, _ := pcmRecorderFixture(t)
	for i := 0; i < 3; i++ {
		p.accept(pcmPacket(i, 1, int16(i+1)))
	}
	for i := 2; i >= 0; i-- {
		p.accept(pcmPacket(i, 2, int16(i+1)))
	}
	p.accept(pcmMarker(0, 3, "drain"))
	// Same poll metadata, separate decoder termination: never merge these.
	p.accept(pcmPacket(0, 4, 99))
	p.accept(pcmMarker(1, 3, "drain"))
	p.accept(pcmMarker(2, 3, "drain"))
	p.accept(pcmMarker(0, 5, "drain"))
	p.finishAll()
	all := p.runtime.Events.Recent(10)
	if len(all) != 4 {
		t.Fatalf("recordings=%d want4", len(all))
	}
	seen := map[string]bool{}
	for _, event := range all {
		if seen[event.ID] || event.AudioPath == nil {
			t.Fatal("duplicate or missing recording")
		}
		seen[event.ID] = true
		samples, rate, err := readPCM16WAV(*event.AudioPath)
		if err != nil || rate != 8000 {
			t.Fatal("invalid WAV", err)
		}
		value := samples[0]
		for _, sample := range samples {
			if sample != value {
				t.Fatal("interleaved audio mixed")
			}
		}
		wantIndex := int(value) - 1
		wantSamples := 1600
		if value == 99 {
			wantIndex = 0
			wantSamples = 800
		}
		if len(samples) != wantSamples || event.DurationSeconds != float64(wantSamples)/8000 || event.DeviceID != fmt.Sprintf("fixture-radio-%d", wantIndex) || *event.TalkgroupID != uint32(100+wantIndex) || *event.P25TDMASlot != wantIndex%2 {
			t.Fatalf("identity/duration wrong: %+v", event)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(p.runtime.dataDirectory, "Data", "p25-call-spool"))
	if len(entries) != 0 {
		t.Fatal("normal segments left checkpoints")
	}
}

func TestP25PCMRecorderDropDiscardsCheckpointEvenAfterSequenceGap(t *testing.T) {
	p, _ := pcmRecorderFixture(t)
	p.accept(pcmPacket(0, 1, 77))
	p.checkpoint()
	recording := p.recordings[0]
	manifest, _ := os.ReadFile(recording.archive.spool.manifest)
	pcm, _ := os.ReadFile(recording.archive.spool.pcm)
	p.accept(pcmMarker(0, 5, "drop"))
	p.finishAll()
	if len(p.runtime.Events.Recent(10)) != 0 {
		t.Fatal("DROP archived audio")
	}
	// A crash after durable revocation but before unlink must not resurrect it.
	var meta p25SpoolManifest
	if err := json.Unmarshal(manifest, &meta); err != nil {
		t.Fatal(err)
	}
	meta.Discarded = true
	manifest, _ = json.Marshal(meta)
	if err := os.WriteFile(recording.archive.spool.manifest, manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recording.archive.spool.pcm, pcm, 0600); err != nil {
		t.Fatal(err)
	}
	if err := p.runtime.recoverP25CallSpools(); err != nil {
		t.Fatal(err)
	}
	if len(p.runtime.Events.Recent(10)) != 0 {
		t.Fatal("discarded checkpoint revived")
	}
}

func TestP25PCMRecorderQueueGapSplitsPartialAudioAndPreservesFlag(t *testing.T) {
	p, _ := pcmRecorderFixture(t)
	p.accept(pcmPacket(0, 1, 11))
	p.accept(pcmPacket(0, 3, 22))
	p.accept(pcmMarker(0, 4, "drain"))
	all := p.runtime.Events.Recent(10)
	if len(all) != 2 {
		t.Fatal("gap joined separate audio")
	}
	for _, event := range all {
		if !event.P25AudioIncomplete || len(event.MediaIssues) == 0 {
			t.Fatal("gap not disclosed")
		}
	}
	events, err := NewEventStore(filepath.Join(p.runtime.dataDirectory, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = events.ReconcileMedia(p.runtime.dataDirectory); err != nil {
		t.Fatal(err)
	}
	for _, event := range events.Recent(10) {
		if !event.P25AudioIncomplete {
			t.Fatal("restart cleared partial flag")
		}
	}
}

func TestP25PCMRecorderRejectsUnassignedStaleEncryptedAndEmptyAssignments(t *testing.T) {
	p, calls := pcmRecorderFixture(t)
	p.setCalls(nil)
	p.accept(pcmPacket(0, 1, 11))
	calls[0].LastHeardAt = pcmFixtureTime().Add(-3 * time.Second)
	p.setCalls(calls)
	p.accept(pcmPacket(0, 2, 11))
	calls[0].LastHeardAt = pcmFixtureTime()
	calls[0].Grant.Encrypted = true
	p.setCalls(calls)
	p.accept(pcmPacket(0, 3, 11))
	p.setCalls(calls[1:])
	p.finishAll()
	if len(p.runtime.Events.Recent(10)) != 0 || len(p.recordings) != 0 {
		t.Fatal("ineligible assignment became recording")
	}
	if _, err := os.Stat(filepath.Join(p.runtime.dataDirectory, "Data", "p25-call-spool")); !os.IsNotExist(err) {
		t.Fatal("empty assignments wrote checkpoints")
	}
}
