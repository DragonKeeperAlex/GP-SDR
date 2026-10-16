package app

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const p25SpoolLimit = 128
const p25MaxSamples = 8000 * 180

// Only newly received samples are written, once per monitor tick. Keep the
// existing bounded RAM buffer so slow/removable storage cannot grow memory.
type p25CallSpool struct {
	manifest string
	pcm      string
	written  int
}

type p25SpoolManifest struct {
	Version    int               `json:"version"`
	Event      TransmissionEvent `json:"event"`
	Transcribe bool              `json:"transcribe"`
}

func (a *p25CallArchive) beginSpool(root string, event TransmissionEvent, transcribe bool) error {
	if event.Encrypted || event.Simulated || event.TalkgroupID == nil || *event.TalkgroupID == 0 {
		return fmt.Errorf("only attributed unencrypted calls can be checkpointed")
	}
	dir := filepath.Join(root, "Data", "p25-call-spool")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	count := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}
	if count >= p25SpoolLimit {
		return fmt.Errorf("%d unfinished call checkpoints; restart after restoring storage to recover them", p25SpoolLimit)
	}
	a.eventID = event.ID
	s := &p25CallSpool{manifest: filepath.Join(dir, event.ID+".json"), pcm: filepath.Join(dir, event.ID+".pcm")}
	encoded, err := json.Marshal(p25SpoolManifest{Version: 1, Event: event, Transcribe: transcribe})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(s.manifest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(encoded)
	if err == nil {
		err = file.Sync()
	}
	if err = errors.Join(err, file.Close()); err != nil {
		// Preserve any partial metadata for diagnosis. Never claim it is durable.
		return err
	}
	a.spool = s
	return nil
}

func (a *p25CallArchive) checkpoint() error {
	if a.spool == nil || len(a.samples) == a.spool.written {
		return nil
	}
	s := a.spool
	file, err := os.OpenFile(s.pcm, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	data := make([]byte, 2*(len(a.samples)-s.written))
	for i, sample := range a.samples[s.written:] {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(sample))
	}
	// Retry at the last acknowledged offset after a partial/full-disk write.
	// O_APPEND would duplicate a partially written batch on the next tick.
	_, err = file.WriteAt(data, int64(2*s.written))
	if err == nil {
		err = file.Sync()
	}
	if err = errors.Join(err, file.Close()); err == nil {
		s.written = len(a.samples)
	}
	return err
}

func (s *p25CallSpool) remove() error {
	// Drop PCM first. A leftover manifest is safe to retry if unlink fails.
	if err := os.Remove(s.pcm); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Remove(s.manifest)
}

var p25SpoolID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// Recover before workers start, one bounded call at a time. Failed/corrupt
// checkpoints stay in place and are reported; no other media is deleted.
func (r *Runtime) recoverP25CallSpools() error {
	dir := filepath.Join(r.dataDirectory, "Data", "p25-call-spool")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var failures []error
	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		count++
		if count > p25SpoolLimit {
			failures = append(failures, fmt.Errorf("recovery limited to %d calls per startup; remaining checkpoints preserved", p25SpoolLimit))
			break
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !entry.Type().IsRegular() || !p25SpoolID.MatchString(id) {
			failures = append(failures, fmt.Errorf("invalid checkpoint name (preserved)"))
			continue
		}
		if err := r.recoverP25CallSpool(dir, id); err != nil {
			failures = append(failures, fmt.Errorf("call %s: %w", id, err))
		}
	}
	return errors.Join(failures...)
}

func (r *Runtime) recoverP25CallSpool(dir, id string) error {
	s := &p25CallSpool{manifest: filepath.Join(dir, id+".json"), pcm: filepath.Join(dir, id+".pcm")}
	file, err := os.Open(s.manifest)
	if err != nil {
		return err
	}
	encoded, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	file.Close()
	var meta p25SpoolManifest
	if err != nil || len(encoded) > 64*1024 || json.Unmarshal(encoded, &meta) != nil || meta.Version != 1 {
		return fmt.Errorf("invalid metadata (preserved)")
	}
	event := meta.Event
	if event.ID != id || event.Encrypted || event.Simulated || event.TalkgroupID == nil || *event.TalkgroupID == 0 || event.P25StreamIndex == nil || *event.P25StreamIndex < 0 || event.StartedAt.IsZero() || !strings.HasPrefix(stringValue(event.ProtocolName), "P25") {
		return fmt.Errorf("ineligible metadata (preserved)")
	}
	existing, exists := r.Events.Get(id)
	if !exists || existing.AudioPath == nil || !fileExists(*existing.AudioPath) {
		info, err := os.Lstat(s.pcm)
		if errors.Is(err, os.ErrNotExist) {
			// A call may stop before its first PCM packet. Preserve its identity
			// with an explicit unavailable-audio reason instead of inventing audio.
			event.MediaIssues = []string{"Interrupted recording before the first audio checkpoint"}
			event.Recovered = true
			if !exists {
				if err := r.Events.Append(event); err != nil {
					return err
				}
			}
			return s.remove()
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 2*p25MaxSamples {
			return fmt.Errorf("invalid PCM checkpoint (preserved)")
		}
		data, err := os.ReadFile(s.pcm)
		if err != nil {
			return err
		}
		a := newP25CallArchive(*event.P25StreamIndex)
		a.eventID = id
		a.samples = make([]int16, len(data)/2)
		for i := range a.samples {
			a.samples[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
		}
		if len(a.samples) == 0 {
			return fmt.Errorf("empty PCM checkpoint (preserved)")
		}
		path, err := a.Save(r.dataDirectory, *event.TalkgroupID, event.StartedAt)
		if err != nil {
			return err
		}
		event.AudioPath, event.DurationSeconds, event.Recovered = &path, a.DurationSeconds(), true
		event.MediaIssues = []string{"Recovered interrupted recording; audio after the last checkpoint may be missing"}
		if exists {
			if err := r.Events.restoreP25CallAudio(id, event); err != nil {
				return err
			}
		} else if err := r.Events.Append(event); err != nil {
			return err
		}
	}
	if meta.Transcribe {
		if r.p25Transcription == nil {
			return fmt.Errorf("transcription queue not ready (checkpoint preserved)")
		}
		if err := r.p25Transcription.enqueue(id, false); err != nil {
			return err
		}
	}
	return s.remove()
}

// Do not append a second event or replace a transcript/analysis when repairing
// a prior finalization that saved the journal but could not save the audio.
func (s *EventStore) restoreP25CallAudio(id string, recovered TransmissionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.positions[id]
	if !ok {
		return ErrNotFound
	}
	event := s.events[index]
	event.AudioPath, event.DurationSeconds, event.Recovered = recovered.AudioPath, recovered.DurationSeconds, true
	event.MediaIssues = recovered.MediaIssues
	if err := appendDurableJSON(s.updatesPath(), event); err != nil {
		return err
	}
	s.events[index] = event
	s.indexEventLocked(event)
	return nil
}
