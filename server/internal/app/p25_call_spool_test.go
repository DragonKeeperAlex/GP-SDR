package app

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func spoolFixture(t *testing.T, root string) (*p25CallArchive, TransmissionEvent) {
	t.Helper()
	call := P25ActiveCall{StreamIndex: 1, DeviceSerial: "fixture-radio", TDMASlot: ptr(1), StartedAt: time.Now().UTC(), Grant: p25GrantState{System: "Synthetic recovery site", GroupID: 1201, FrequencyHz: 772881250}}
	event := (&Runtime{}).p25CallEvent(call)
	a := newP25CallArchive(1)
	if err := a.beginSpool(root, event, true); err != nil {
		t.Fatal(err)
	}
	return a, event
}

// Run in a separate process so the parent proves recovery without deferred
// finalizers, Stop, or any of the old in-memory archive surviving.
func TestP25SpoolCrashHelper(t *testing.T) {
	root := os.Getenv("GPSDR_P25_SPOOL_CRASH_FIXTURE")
	if root == "" {
		return
	}
	a, _ := spoolFixture(t, root)
	pcm := make([]int16, 8000)
	for i := range pcm {
		pcm[i] = int16(i % 200)
	}
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: pcm})
	if err := a.checkpoint(); err != nil {
		t.Fatal(err)
	}
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: make([]int16, 4000)})
	os.Stdout.WriteString("checkpoint ready\n")
	time.Sleep(time.Minute)
}

func TestP25SpoolRecoversAfterForcedProcessKillWithoutDuplicates(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestP25SpoolCrashHelper$")
	cmd.Env = append(os.Environ(), "GPSDR_P25_SPOOL_CRASH_FIXTURE="+root)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ready, err := bufio.NewReader(out).ReadString('\n')
	if err != nil || ready != "checkpoint ready\n" {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("fixture not checkpointed: %q %v", ready, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	events, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{dataDirectory: root, Events: events, p25Transcription: newP25TranscriptionQueue(root, events, nil)}
	dir := filepath.Join(root, "Data", "p25-call-spool")
	entries, _ := os.ReadDir(dir)
	backup := map[string][]byte{}
	for _, e := range entries {
		backup[e.Name()], _ = os.ReadFile(filepath.Join(dir, e.Name()))
	}
	if err := r.recoverP25CallSpools(); err != nil {
		t.Fatal(err)
	}
	all := events.Recent(10)
	if len(all) != 1 || !all[0].Recovered || all[0].AudioPath == nil || all[0].DurationSeconds != 1 || all[0].DeviceID != "fixture-radio" || *all[0].P25TDMASlot != 1 || *all[0].TalkgroupID != 1201 {
		t.Fatalf("lost checkpoint identity/audio: %+v", all)
	}
	samples, rate, err := readPCM16WAV(*all[0].AudioPath)
	if err != nil || rate != 8000 || len(samples) != 8000 || samples[199] != 199 {
		t.Fatalf("checkpoint PCM changed: samples=%d rate=%d err=%v", len(samples), rate, err)
	}
	jobs := r.p25Transcription.snapshot()
	if len(jobs) != 1 || jobs[0].EventID != all[0].ID || jobs[0].State != "pending" {
		t.Fatal("recovered transcription not queued")
	}
	// Simulate a crash after journal + queue persistence but before checkpoint
	// unlink. Recovery must retain the same event and audio, not duplicate it.
	for name, data := range backup {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	reopened, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	r.Events = reopened
	r.p25Transcription = newP25TranscriptionQueue(root, reopened, nil)
	if err := r.recoverP25CallSpools(); err != nil || len(reopened.Recent(10)) != 1 {
		t.Fatalf("non-idempotent recovery: %v", err)
	}
	remaining, _ := os.ReadDir(dir)
	if len(remaining) != 0 {
		t.Fatal("durable checkpoint not cleaned")
	}
}

func TestP25SpoolStorageRetryCorruptionAndEncryptedExclusion(t *testing.T) {
	root := t.TempDir()
	a, event := spoolFixture(t, root)
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: []int16{11, 22, 33}})
	if err := os.Mkdir(a.spool.pcm, 0700); err != nil {
		t.Fatal(err)
	}
	if err := a.checkpoint(); err == nil || a.spool.written != 0 {
		t.Fatal("failed storage acknowledged")
	}
	if err := os.Remove(a.spool.pcm); err != nil {
		t.Fatal(err)
	}
	if err := a.checkpoint(); err != nil {
		t.Fatal(err)
	}
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: []int16{44}})
	if err := a.checkpoint(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(a.spool.pcm)
	if len(data) != 8 {
		t.Fatal("retry duplicated samples")
	}
	event.Encrypted = true
	if err := newP25CallArchive(0).beginSpool(root, event, true); err == nil {
		t.Fatal("encrypted call spooled")
	}
	if err := os.WriteFile(a.spool.manifest, []byte("broken metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	events, _ := NewEventStore(filepath.Join(root, "Data"))
	r := &Runtime{Events: events, dataDirectory: root}
	if err := r.recoverP25CallSpools(); err == nil || !strings.Contains(err.Error(), "preserved") {
		t.Fatal("corruption silently ignored")
	}
	if data, _ := os.ReadFile(a.spool.manifest); string(data) != "broken metadata" {
		t.Fatal("corrupt metadata overwritten")
	}
	if len(events.Recent(10)) != 0 {
		t.Fatal("corrupt checkpoint invented an event")
	}
}

func TestP25SpoolJournalFailureRetainsRecoverableAudio(t *testing.T) {
	root := t.TempDir()
	a, event := spoolFixture(t, root)
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: make([]int16, 8000)})
	events, _ := NewEventStore(filepath.Join(root, "Data"))
	journal := events.path
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	events.path = filepath.Join(blocker, "events.jsonl")
	r := &Runtime{Events: events, dataDirectory: root}
	call := P25ActiveCall{StreamIndex: 1, DeviceSerial: event.DeviceID, StartedAt: event.StartedAt, Grant: p25GrantState{System: *event.SystemName, GroupID: *event.TalkgroupID}}
	r.finishP25Call(make(chan struct{}), ScanProfile{}, call, a)
	if _, err := os.Stat(a.spool.pcm); err != nil || len(events.Recent(10)) != 0 {
		t.Fatal("journal failure lost checkpoint or published failed event")
	}
	events.path = journal
	r.p25Transcription = newP25TranscriptionQueue(root, events, nil)
	if err := r.recoverP25CallSpools(); err != nil {
		t.Fatal(err)
	}
	if all := events.Recent(10); len(all) != 1 || all[0].ID != event.ID || all[0].DurationSeconds != 1 {
		t.Fatalf("journal recovery identity/duration: %+v", all)
	}
}

func TestP25SpoolNormalFinishCleansEmptyAndRecordedCalls(t *testing.T) {
	for _, count := range []int{0, 8000} {
		root := t.TempDir()
		a, event := spoolFixture(t, root)
		a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: make([]int16, count)})
		events, err := NewEventStore(filepath.Join(root, "Data"))
		if err != nil {
			t.Fatal(err)
		}
		r := &Runtime{Events: events, dataDirectory: root, p25Transcription: newP25TranscriptionQueue(root, events, nil)}
		profile := ScanProfile{}
		profile.Settings.TranscribeVoice = true
		call := P25ActiveCall{StreamIndex: 1, DeviceSerial: event.DeviceID, StartedAt: event.StartedAt, Grant: p25GrantState{System: *event.SystemName, GroupID: *event.TalkgroupID}}
		r.finishP25Call(make(chan struct{}), profile, call, a)
		files, err := os.ReadDir(filepath.Dir(a.spool.manifest))
		if err != nil || len(files) != 0 {
			t.Fatalf("normal finish leaked checkpoint for %d samples: %v files=%d", count, err, len(files))
		}
		if all := events.Recent(10); len(all) != 1 || (all[0].AudioPath != nil) != (count > 0) {
			t.Fatalf("empty/recorded call metadata lost: %+v", all)
		}
	}
}

// Opt-in on an expendable, separately mounted <=2 MiB filesystem. Never fill
// the real recording volume to exercise full-disk behavior.
func TestP25SpoolRealENOSPCRetry(t *testing.T) {
	fixture := os.Getenv("GPSDR_P25_ENOSPC_FIXTURE_ROOT")
	if fixture == "" {
		t.Skip("requires expendable small filesystem")
	}
	root, err := os.MkdirTemp(fixture, "p25-full-disk-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	a, _ := spoolFixture(t, root)
	a.Append(AudioFrame{ChannelID: a.streamID, SampleRate: 8000, Samples: make([]int16, p25MaxSamples)})
	if err := a.checkpoint(); !errors.Is(err, syscall.ENOSPC) || a.spool.written != 0 {
		t.Fatalf("expected unacknowledged ENOSPC: %v written=%d", err, a.spool.written)
	}
	if err := os.Remove(a.spool.pcm); err != nil {
		t.Fatal(err)
	}
	a.samples = a.samples[:8000]
	if err := a.checkpoint(); err != nil {
		t.Fatal(err)
	}
	events, err := NewEventStore(filepath.Join(root, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{Events: events, dataDirectory: root, p25Transcription: newP25TranscriptionQueue(root, events, nil)}
	if err := r.recoverP25CallSpools(); err != nil {
		t.Fatal(err)
	}
	if all := events.Recent(10); len(all) != 1 || all[0].DurationSeconds != 1 {
		t.Fatalf("ENOSPC recovery failed: %+v", all)
	}
}
