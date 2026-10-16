package app

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type P25Recording struct {
	RecordedDurationSeconds *float64 `json:"recordedDurationSeconds,omitempty"`
	TransmissionEvent
	TranscriptionState string `json:"transcriptionState,omitempty"`
	TranscriptionError string `json:"transcriptionError,omitempty"`
	AudioAvailable     bool   `json:"audioAvailable"`
}

type P25RecordingsPage struct {
	QueueError string         `json:"queueError,omitempty"`
	Calls      []P25Recording `json:"calls"`
	NextBefore string         `json:"nextBefore,omitempty"`
}

// P25Recordings filters before limiting so survey traffic cannot hide saved
// calls. Cursor paging stays in the in-memory journal; it never opens audio.
func (s *EventStore) P25Recordings(query, before string, limit int) (P25RecordingsPage, error) {
	page, err := s.p25RecordingsSnapshot(query, before, limit)
	if err != nil {
		return page, err
	}
	// At most 100 metadata checks, after releasing the event journal lock.
	for index := range page.Calls {
		call := &page.Calls[index]
		if call.AudioPath != nil {
			info, err := os.Stat(*call.AudioPath)
			call.AudioAvailable = err == nil && info.Mode().IsRegular() && info.Size() > 44
			if call.AudioAvailable {
				call.RecordedDurationSeconds = p25RecordedWAVDuration(*call.AudioPath, info.Size())
			}
		}
	}
	return page, nil
}

func (s *EventStore) p25RecordingsSnapshot(query, before string, limit int) (P25RecordingsPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	limit = max(1, min(100, limit))
	page := P25RecordingsPage{Calls: make([]P25Recording, 0, limit)}
	end := len(s.events) - 1
	if before != "" {
		index, ok := s.positions[before]
		if !ok {
			return page, fmt.Errorf("recording page is no longer available; refresh recent calls")
		}
		end = index - 1
	}
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	for i := end; i >= 0; i-- {
		e := s.events[i]
		if latest, ok := s.positions[e.ID]; ok && latest != i {
			continue
		}
		protocol := strings.ToLower(stringValue(e.ProtocolName))
		if ((e.AudioPath == nil || *e.AudioPath == "") && len(e.MediaIssues) == 0) || e.Encrypted || e.Simulated || e.TalkgroupID == nil || *e.TalkgroupID == 0 || (protocol != "p25" && protocol != "p25 phase 1" && protocol != "p25 phase 2") {
			continue
		}
		if len(terms) > 0 {
			text := strings.ToLower(strings.Join([]string{stringValue(e.SystemName), stringValue(e.Label), stringValue(e.Transcript), strconv.FormatUint(uint64(*e.TalkgroupID), 10), e.StartedAt.Local().Format("2006-01-02 01/02/2006")}, " "))
			matches := true
			for _, term := range terms {
				if !strings.Contains(text, term) {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}
		if len(page.Calls) == limit {
			page.NextBefore = page.Calls[len(page.Calls)-1].ID
			break
		}
		page.Calls = append(page.Calls, P25Recording{TransmissionEvent: e})
	}
	return page, nil
}

// Read only the canonical PCM header produced by this application, not the
// waveform. Unsupported layouts keep their historical duration untouched.
func p25RecordedWAVDuration(path string, size int64) *float64 {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var header [44]byte
	if _, err := file.ReadAt(header[:], 0); err != nil {
		return nil
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" || string(header[12:16]) != "fmt " || binary.LittleEndian.Uint32(header[16:20]) != 16 || binary.LittleEndian.Uint16(header[20:22]) != 1 || string(header[36:40]) != "data" {
		return nil
	}
	rate := binary.LittleEndian.Uint32(header[24:28])
	channels := binary.LittleEndian.Uint16(header[22:24])
	bits := binary.LittleEndian.Uint16(header[34:36])
	if rate == 0 || channels == 0 || bits != 16 {
		return nil
	}
	bytes := min(int64(binary.LittleEndian.Uint32(header[40:44])), size-44)
	duration := float64(bytes) / float64(uint64(rate)*uint64(channels)*2)
	return &duration
}
