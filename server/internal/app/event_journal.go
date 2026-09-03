package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Base events and updates are append-only. UI pagination never deletes history.
func (s *EventStore) updatesPath() string {
	return filepath.Join(filepath.Dir(s.path), "event-updates.jsonl")
}

func appendDurableJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}

func (s *EventStore) persistUpdateLocked(index int) error {
	return appendDurableJSON(s.updatesPath(), s.events[index])
}

func (s *EventStore) loadUpdates() error {
	if err := repairTornEventTail(s.updatesPath()); err != nil {
		return err
	}
	file, err := os.Open(s.updatesPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	positions := make(map[string]int, len(s.events))
	for i, e := range s.events {
		positions[e.ID] = i
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 65536), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var event TransmissionEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("event update journal line %d: %w (file preserved)", line, err)
		}
		if i, ok := positions[event.ID]; ok {
			s.events[i] = event
		}
	}
	return scanner.Err()
}

func (s *EventStore) eventIndicesLocked(id string) []int {
	if s.positions == nil {
		s.positions = make(map[string]int)
		for i, e := range s.events {
			s.positions[e.ID] = i
		}
	}
	if i, ok := s.positions[id]; ok {
		return []int{i}
	}
	return nil
}

// A power interruption can leave only the final append incomplete. Preserve
// the original file before removing that uncommitted tail; never skip bad rows
// in the middle of a journal.
func repairTornEventTail(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	if info.Size() == 0 {
		f.Close()
		return nil
	}
	last := make([]byte, 1)
	_, err = f.ReadAt(last, info.Size()-1)
	f.Close()
	if err != nil {
		return err
	}
	if last[0] == '\n' {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	boundary := bytes.LastIndexByte(data, '\n') + 1
	var event TransmissionEvent
	if json.Unmarshal(data[boundary:], &event) == nil {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte{'\n'})
		if err == nil {
			err = f.Sync()
		}
		return errors.Join(err, f.Close())
	}
	if err := writeDurableFile(path+".interrupted-"+NewID(), data); err != nil {
		return err
	}
	return os.Truncate(path, int64(boundary))
}
