package app

import (
	"fmt"
	"strconv"
	"strings"
)

type P25RecordingsPage struct {
	Calls      []TransmissionEvent `json:"calls"`
	NextBefore string              `json:"nextBefore,omitempty"`
}

// P25Recordings filters before limiting so survey traffic cannot hide saved
// calls. Cursor paging stays in the in-memory journal; it never opens audio.
func (s *EventStore) P25Recordings(query, before string, limit int) (P25RecordingsPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	limit = max(1, min(100, limit))
	page := P25RecordingsPage{Calls: make([]TransmissionEvent, 0, limit)}
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
		if e.AudioPath == nil || *e.AudioPath == "" || e.Encrypted || e.Simulated || e.TalkgroupID == nil || *e.TalkgroupID == 0 || (protocol != "p25" && protocol != "p25 phase 1" && protocol != "p25 phase 2") {
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
		page.Calls = append(page.Calls, e)
	}
	return page, nil
}
