package app

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type MediaRecoveryReport struct {
	CheckedAt time.Time `json:"checkedAt"`
	Recovered int       `json:"recovered"`
	Relinked  int       `json:"relinked"`
	Missing   int       `json:"missing"`
	Invalid   int       `json:"invalid"`
	Requeued  int       `json:"requeued"`
}

// ReconcileMedia runs before receiver/analysis workers start. It never deletes
// payloads or invents decoder evidence, identities, locations, or signal levels.
func (s *EventStore) ReconcileMedia(root string) (MediaRecoveryReport, error) {
	report := MediaRecoveryReport{CheckedAt: time.Now().UTC()}
	files := map[string]string{}
	ambiguous := map[string]bool{}
	var iqFiles, wavFiles []string
	for _, folder := range []string{"IQ", "Recordings"} {
		err := filepath.WalkDir(filepath.Join(root, folder), func(path string, entry fs.DirEntry, err error) error {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() {
				if entry.Name() == "Archive" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".cs8" && ext != ".cu8" && ext != ".wav" {
				return nil
			}
			if _, exists := files[entry.Name()]; exists {
				ambiguous[entry.Name()] = true
			}
			files[entry.Name()] = path
			if ext == ".wav" {
				wavFiles = append(wavFiles, path)
			} else {
				iqFiles = append(iqFiles, path)
			}
			return nil
		})
		if err != nil {
			return report, err
		}
	}
	newEvents := []TransmissionEvent{}
	recoverEvent := func(event TransmissionEvent) {
		if _, exists := s.positions[event.ID]; exists {
			return
		}
		s.positions[event.ID] = len(s.events)
		s.events = append(s.events, event)
		newEvents = append(newEvents, event)
		report.Recovered++
	}
	referenced := map[string]bool{}
	for index := range s.events {
		before := s.events[index]
		event := before
		event.MediaIssues = nil
		filteredMessages := validDecoderMessages(event.DecoderMessages)
		if len(filteredMessages) != len(event.DecoderMessages) {
			event.DecoderMessages = filteredMessages
			event.Analysis = nil
			if len(filteredMessages) == 0 {
				event.ProtocolName = nil
				event.Confidence = minFloat(event.Confidence, .72)
			}
		}
		if event.Transcript != nil {
			cleaned := cleanRadioTranscript(*event.Transcript)
			if cleaned != *event.Transcript {
				if cleaned == "" {
					event.Transcript = nil
				} else {
					event.Transcript = ptr(cleaned)
				}
				event.Analysis = nil
			}
		}
		for _, field := range []**string{&event.AudioPath, &event.IQPath} {
			if *field == nil || **field == "" {
				continue
			}
			path := **field
			if _, err := os.Stat(path); err != nil {
				name := filepath.Base(path)
				if replacement, ok := files[name]; ok && !ambiguous[name] {
					*field = ptr(replacement)
					path = replacement
					report.Relinked++
				} else {
					event.MediaIssues = append(event.MediaIssues, "Missing recording: "+path)
					report.Missing++
				}
			}
			referenced[path] = true
		}
		if event.AnalysisStatus == "running" || (event.AnalysisStatus == "error" && (event.AnalysisError == "context canceled" || event.AnalysisError == "context deadline exceeded")) {
			event.AnalysisStatus, event.AnalysisError, event.AnalysisCompletedAt = "pending", "", nil
			report.Requeued++
		}
		if !reflect.DeepEqual(before.Analysis, event.Analysis) && (event.AudioPath != nil || event.IQPath != nil) {
			event.AnalysisStatus, event.AnalysisError, event.AnalysisCompletedAt = "pending", "", nil
			report.Requeued++
		}
		if !reflect.DeepEqual(before, event) {
			if err := appendDurableJSON(s.updatesPath(), event); err != nil {
				return report, err
			}
			s.events[index] = event
		}
	}
	for _, path := range iqFiles {
		if referenced[path] {
			continue
		}
		encoded, err := os.ReadFile(iqMetadataPath(path))
		if err != nil {
			report.Invalid++
			continue
		}
		var meta IQCaptureMetadata
		info, statErr := os.Stat(path)
		if json.Unmarshal(encoded, &meta) != nil || statErr != nil || meta.SampleRateHz <= 0 || meta.SampleBytes <= 0 || int64(meta.SampleBytes) != info.Size() {
			report.Invalid++
			continue
		}
		if index, ok := s.positions[meta.EventID]; ok {
			event := s.events[index]
			if event.IQPath == nil || !fileExists(*event.IQPath) {
				event.IQPath = ptr(path)
				if err := appendDurableJSON(s.updatesPath(), event); err != nil {
					return report, err
				}
				s.events[index] = event
				report.Relinked++
			}
			continue
		}
		event := recoveredEvent(root, path)
		if meta.EventID != "" {
			event.ID = meta.EventID
		}
		event.IQPath, event.StartedAt, event.FrequencyHz = ptr(path), meta.CapturedAt, meta.FrequencyHz
		event.DurationSeconds = float64(meta.SampleBytes) / float64(2*meta.SampleRateHz)
		recoverEvent(event)
	}
	for _, path := range wavFiles {
		if referenced[path] {
			continue
		}
		event := recoveredEvent(root, path)
		if _, exists := s.positions[event.ID]; exists {
			continue
		}
		// GP-SDR's canonical PCM WAV header; leave unfamiliar formats untouched.
		file, err := os.Open(path)
		if err != nil {
			return report, err
		}
		header := make([]byte, 44)
		_, err = io.ReadFull(file, header)
		info, statErr := file.Stat()
		file.Close()
		parts := strings.Split(filepath.Base(path), "-")
		if err != nil || statErr != nil || string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" || string(header[36:40]) != "data" || binary.LittleEndian.Uint16(header[20:22]) != 1 || len(parts) < 3 {
			report.Invalid++
			continue
		}
		byteRate := binary.LittleEndian.Uint32(header[28:32])
		size := binary.LittleEndian.Uint32(header[40:44])
		at, timeErr := time.Parse("20060102T150405.000Z", parts[0])
		freq, freqErr := strconv.ParseFloat(parts[1], 64)
		if byteRate == 0 || int64(size)+44 > info.Size() || timeErr != nil || freqErr != nil {
			report.Invalid++
			continue
		}
		event.AudioPath, event.FrequencyHz = ptr(path), freq
		event.DurationSeconds = float64(size) / float64(byteRate)
		event.StartedAt = at.Add(-time.Duration(event.DurationSeconds * float64(time.Second)))
		recoverEvent(event)
	}
	if len(newEvents) > 0 {
		file, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return report, err
		}
		writer := bufio.NewWriterSize(file, 256*1024)
		encoder := json.NewEncoder(writer)
		for _, event := range newEvents {
			if err = encoder.Encode(event); err != nil {
				break
			}
		}
		if err == nil {
			err = writer.Flush()
		}
		if err == nil {
			err = file.Sync()
		}
		err = errors.Join(err, file.Close())
		if err != nil {
			return report, err
		}
		for _, event := range newEvents {
			s.aggregate(event)
		}
	}
	s.rebuildSearchIndexLocked()
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(root, "Data", "media-recovery.json"), encoded, 0600)
	}
	return report, err
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func recoveredEvent(root, path string) TransmissionEvent {
	relative, _ := filepath.Rel(root, path)
	hash := sha256.Sum256([]byte(relative))
	return TransmissionEvent{ID: "recovered-" + hex.EncodeToString(hash[:16]), Recovered: true, Modulation: "UNKNOWN",
		Label: ptr("Recovered recording · original event metadata unavailable"), AnalysisPolicy: "manual", AnalysisStatus: "pending", IQRetentionPolicy: "quarantine"}
}
