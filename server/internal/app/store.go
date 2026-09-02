package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("requested item was not found")
	ErrBuiltIn  = errors.New("built-in profiles cannot be replaced; duplicate the profile first")
)

type ProfileStore struct {
	mu       sync.RWMutex
	dir      string
	profiles map[string]ScanProfile
}

func NewProfileStore(dir string) (*ProfileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := &ProfileStore{dir: dir, profiles: make(map[string]ScanProfile)}
	for _, profile := range builtInProfiles() {
		store.profiles[profile.ID] = profile
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var profile ScanProfile
		if json.Unmarshal(data, &profile) == nil && validateProfile(profile) == nil {
			store.profiles[profile.ID] = normalizeProfile(profile)
		}
	}
	return store, nil
}

func (s *ProfileStore) All() []ScanProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ScanProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		items = append(items, profile)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].BuiltIn != items[j].BuiltIn {
			return items[i].BuiltIn
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items
}

func (s *ProfileStore) Get(id string) (ScanProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.profiles[id]
	return profile, ok
}

func (s *ProfileStore) ReplaceSyncedProfiles(profiles []ScanProfile) error {
	for _, profile := range profiles {
		if !strings.HasPrefix(profile.ID, "sheet-") || !profile.BuiltIn {
			return errors.New("synced range profiles need a sheet ID and built-in flag")
		}
		if err := validateProfile(profile); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.profiles {
		if strings.HasPrefix(id, "sheet-") {
			delete(s.profiles, id)
		}
	}
	for _, profile := range profiles {
		s.profiles[profile.ID] = normalizeProfile(profile)
	}
	return nil
}

func (s *ProfileStore) Save(profile ScanProfile) (ScanProfile, error) {
	if err := validateProfile(profile); err != nil {
		return ScanProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.profiles[profile.ID]; ok && existing.BuiltIn {
		return ScanProfile{}, ErrBuiltIn
	}
	profile = normalizeProfile(profile)
	profile.BuiltIn = false
	s.profiles[profile.ID] = profile
	return profile, s.persist(profile)
}

func (s *ProfileStore) Import(data []byte) (ScanProfile, error) {
	if len(data) > 1_000_000 {
		return ScanProfile{}, errors.New("profile is larger than 1 MB")
	}
	var profile ScanProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return ScanProfile{}, errors.New("not a valid GP-SDR profile")
	}
	if err := validateProfile(profile); err != nil {
		return ScanProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.profiles[profile.ID]; exists || profile.BuiltIn {
		profile.ID = NewID()
	}
	profile = normalizeProfile(profile)
	profile.BuiltIn = false
	s.profiles[profile.ID] = profile
	return profile, s.persist(profile)
}

func (s *ProfileStore) Duplicate(id string) (ScanProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[id]
	if !ok {
		return ScanProfile{}, ErrNotFound
	}
	profile.ID, profile.Name, profile.BuiltIn = NewID(), profile.Name+" Copy", false
	s.profiles[profile.ID] = profile
	return profile, s.persist(profile)
}

func (s *ProfileStore) Export(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.profiles[id]
	if !ok {
		return nil, ErrNotFound
	}
	return json.MarshalIndent(profile, "", "  ")
}

func (s *ProfileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[id]
	if !ok {
		return ErrNotFound
	}
	if profile.BuiltIn {
		return ErrBuiltIn
	}
	delete(s.profiles, id)
	return os.Remove(filepath.Join(s.dir, id+".json"))
}

func (s *ProfileStore) persist(profile ScanProfile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	temporary := filepath.Join(s.dir, profile.ID+".tmp")
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, filepath.Join(s.dir, profile.ID+".json"))
}

func validateProfile(profile ScanProfile) error {
	if profile.SchemaVersion != 1 {
		return fmt.Errorf("unsupported profile version %d", profile.SchemaVersion)
	}
	if strings.TrimSpace(profile.Name) == "" || len(profile.Name) > 80 {
		return errors.New("profile name is required and must be 80 characters or fewer")
	}
	if len(profile.Ranges) > 100 || len(profile.Channels) > 5000 || len(profile.P25Systems) > 20 {
		return errors.New("profile contains too many ranges or channels")
	}
	if rate := profile.Settings.P25SampleRateHz; rate != 0 {
		_, rtlRate := rtlSDRSampleRateName(rate)
		if !rtlRate && !isHackRFSampleRate(rate) {
			return fmt.Errorf("unsupported P25 capture rate %d", rate)
		}
	}
	if area := profile.ReferenceArea; area != nil {
		if !strings.EqualFold(strings.TrimSpace(area.Provider), "RadioReference") || area.Latitude < -90 || area.Latitude > 90 || area.Longitude < -180 || area.Longitude > 180 || area.RadiusMiles < 1 || area.RadiusMiles > 100 {
			return errors.New("profile reference area is invalid")
		}
	}
	for _, item := range profile.Ranges {
		if item.StartHz < 0 || item.EndHz <= item.StartHz || item.StepHz <= 0 || item.DwellMilliseconds < 20 {
			return fmt.Errorf("invalid values in range %q", item.Name)
		}
	}
	for _, item := range profile.Channels {
		if item.FrequencyHz <= 0 || item.BandwidthHz <= 0 {
			return fmt.Errorf("invalid values in channel %q", item.Name)
		}
	}
	for _, system := range profile.P25Systems {
		if strings.TrimSpace(system.Name) == "" || len(system.ControlChannelsHz) == 0 || len(system.ControlChannelsHz) > 32 {
			return fmt.Errorf("P25 system %q needs a name and at least one control channel", system.Name)
		}
		if len(system.Talkgroups) > 20_000 {
			return fmt.Errorf("P25 system %q has too many talkgroups", system.Name)
		}
		seenTalkgroups := make(map[int]bool, len(system.Talkgroups))
		for _, talkgroup := range system.Talkgroups {
			if talkgroup.ID < 1 || talkgroup.ID > 65535 {
				return fmt.Errorf("P25 system %q has an invalid talkgroup ID %d", system.Name, talkgroup.ID)
			}
			if seenTalkgroups[talkgroup.ID] {
				return fmt.Errorf("P25 system %q contains duplicate talkgroup ID %d", system.Name, talkgroup.ID)
			}
			seenTalkgroups[talkgroup.ID] = true
		}
		for _, frequency := range system.ControlChannelsHz {
			if frequency <= 0 {
				return fmt.Errorf("P25 system %q has an invalid control channel", system.Name)
			}
		}
	}
	return nil
}

type EventStore struct {
	mu          sync.RWMutex
	path        string
	events      []TransmissionEvent
	signals     map[string]SignalSummary
	searchIndex map[string]map[string]struct{}
}

func NewEventStore(dir string) (*EventStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := &EventStore{path: filepath.Join(dir, "events.jsonl"), signals: make(map[string]SignalSummary), searchIndex: make(map[string]map[string]struct{})}
	file, err := os.Open(store.path)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.WriteFile(store.path, nil, 0o644); err != nil {
			return nil, err
		}
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	removedLegacyEvents := 0
	for scanner.Scan() {
		var event TransmissionEvent
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			if isLegacyMapperFalsePositive(event) {
				removedLegacyEvents++
				continue
			}
			store.events = append(store.events, event)
			store.aggregate(event)
		}
	}
	scanErr := scanner.Err()
	closeErr := file.Close()
	if scanErr != nil {
		return nil, scanErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(store.events) > 25_000 {
		store.events = store.events[len(store.events)-25_000:]
	}
	store.rebuildSearchIndexLocked()
	if removedLegacyEvents > 0 {
		if err := writeEventFile(store.path, store.events); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func isLegacyMapperFalsePositive(event TransmissionEvent) bool {
	return event.NoiseDBFS == 0 && event.DurationSeconds == .2 && event.Label != nil && *event.Label == "Mapper discovery" && event.Confidence == .72
}

func writeEventFile(path string, events []TransmissionEvent) error {
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			_ = file.Close()
			_ = os.Remove(temporary)
			return err
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return os.Rename(temporary, path)
}

func (s *EventStore) Append(event TransmissionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	s.events = append(s.events, event)
	if len(s.events) > 25_000 {
		s.events = s.events[len(s.events)-25_000:]
		s.rebuildSearchIndexLocked()
	} else {
		s.indexEventLocked(event)
	}
	s.aggregate(event)
	return nil
}

func (s *EventStore) Recent(limit int) []TransmissionEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 {
		limit = 1
	}
	if limit > 2000 {
		limit = 2000
	}
	if limit > len(s.events) {
		limit = len(s.events)
	}
	items := make([]TransmissionEvent, limit)
	for i := 0; i < limit; i++ {
		items[i] = s.events[len(s.events)-1-i]
	}
	return items
}

// Search uses an in-memory token index so transcript, callsign, label, system,
// protocol and frequency searches remain responsive as the journal grows.
func (s *EventStore) Search(query string, limit int) []TransmissionEvent {
	terms := searchTokens(query)
	if len(terms) == 0 {
		return s.Recent(limit)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 {
		limit = 1
	}
	if limit > 2000 {
		limit = 2000
	}
	var matches map[string]struct{}
	for _, term := range terms {
		termMatches := make(map[string]struct{})
		for token, ids := range s.searchIndex {
			if token != term && !strings.HasPrefix(token, term) {
				continue
			}
			for id := range ids {
				termMatches[id] = struct{}{}
			}
		}
		if matches == nil {
			matches = termMatches
			continue
		}
		for id := range matches {
			if _, ok := termMatches[id]; !ok {
				delete(matches, id)
			}
		}
	}
	result := make([]TransmissionEvent, 0, minInt(limit, len(matches)))
	for index := len(s.events) - 1; index >= 0 && len(result) < limit; index-- {
		if _, ok := matches[s.events[index].ID]; ok {
			result = append(result, s.events[index])
		}
	}
	return result
}

func (s *EventStore) Get(id string) (TransmissionEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for index := len(s.events) - 1; index >= 0; index-- {
		if s.events[index].ID == id {
			return s.events[index], true
		}
	}
	return TransmissionEvent{}, false
}

func (s *EventStore) UpdateTranscript(id, transcript string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for index := range s.events {
		if s.events[index].ID == id {
			s.events[index].Transcript = ptr(transcript)
			callsigns := ExtractCallsigns(transcript)
			s.events[index].Callsigns = mergeUniqueStrings(s.events[index].Callsigns, callsigns)
			if s.events[index].Analysis != nil {
				s.events[index].Analysis.Callsigns = mergeUniqueStrings(s.events[index].Analysis.Callsigns, callsigns)
			}
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	s.rebuildSearchIndexLocked()
	return writeEventFile(s.path, s.events)
}

func (s *EventStore) UpdateAnalysis(id string, analysis SignalIntelligence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.events {
		if s.events[index].ID != id {
			continue
		}
		// A language model may summarize evidence but cannot overrule a real
		// decoder. Keep decoder-proven protocol and confidence authoritative.
		if len(s.events[index].DecoderMessages) > 0 {
			analysis.Modulation = s.events[index].Modulation
			analysis.SignalFamily = stringValue(s.events[index].ProtocolName)
			analysis.Confidence = maxFloat(analysis.Confidence, s.events[index].Confidence)
		}
		s.events[index].Analysis = &analysis
		s.events[index].Callsigns = mergeUniqueStrings(s.events[index].Callsigns, analysis.Callsigns)
		s.rebuildSearchIndexLocked()
		return writeEventFile(s.path, s.events)
	}
	return ErrNotFound
}

func (s *EventStore) UpdateDecoderMessages(id string, messages []DecoderMessage) error {
	if len(messages) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.events {
		if s.events[index].ID != id {
			continue
		}
		s.events[index].DecoderMessages = append([]DecoderMessage(nil), messages...)
		s.events[index].ProtocolName = ptr(messages[0].Protocol)
		s.events[index].Confidence = maxFloat(s.events[index].Confidence, messages[0].Confidence)
		for _, message := range messages {
			s.events[index].Callsigns = mergeUniqueStrings(s.events[index].Callsigns, message.Callsigns)
		}
		s.rebuildSearchIndexLocked()
		return writeEventFile(s.path, s.events)
	}
	return ErrNotFound
}

func (s *EventStore) UpdateAudioPath(id, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.events {
		if s.events[index].ID == id {
			s.events[index].AudioPath = ptr(path)
			return writeEventFile(s.path, s.events)
		}
	}
	return ErrNotFound
}

func (s *EventStore) UpdateIQPath(id, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.events {
		if s.events[index].ID == id {
			if strings.TrimSpace(path) == "" {
				s.events[index].IQPath = nil
			} else {
				s.events[index].IQPath = ptr(path)
			}
			return writeEventFile(s.path, s.events)
		}
	}
	return ErrNotFound
}

func (s *EventStore) UpdateAnalysisStatus(id, status, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.events {
		if s.events[index].ID != id {
			continue
		}
		s.events[index].AnalysisStatus = status
		s.events[index].AnalysisError = strings.TrimSpace(message)
		if status == "complete" || status == "error" {
			now := time.Now().UTC()
			s.events[index].AnalysisCompletedAt = &now
		} else {
			s.events[index].AnalysisCompletedAt = nil
		}
		return writeEventFile(s.path, s.events)
	}
	return ErrNotFound
}

func (s *EventStore) PendingAnalysis(limit int, jobID string) []TransmissionEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 || limit > 25_000 {
		limit = 25_000
	}
	items := make([]TransmissionEvent, 0, minInt(limit, len(s.events)))
	for index := len(s.events) - 1; index >= 0 && len(items) < limit; index-- {
		event := s.events[index]
		if event.AnalysisStatus != "pending" || (jobID != "" && event.MapperJobID != jobID) {
			continue
		}
		items = append(items, event)
	}
	return items
}

func (s *EventStore) Signals(limit int) []SignalSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]SignalSummary, 0, len(s.signals))
	for _, item := range s.signals {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].LastSeen.After(items[j].LastSeen) })
	if limit < 1 {
		limit = 1
	}
	if limit < len(items) {
		items = items[:limit]
	}
	return items
}

func (s *EventStore) Count() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.events) }

func (s *EventStore) rebuildSearchIndexLocked() {
	s.searchIndex = make(map[string]map[string]struct{})
	for _, event := range s.events {
		s.indexEventLocked(event)
	}
}

func (s *EventStore) indexEventLocked(event TransmissionEvent) {
	parts := []string{event.ID, fmt.Sprintf("%.0f %.6f", event.FrequencyHz, event.FrequencyHz/1e6), event.Modulation,
		stringValue(event.ProtocolName), stringValue(event.Label), event.DeviceID, stringValue(event.SystemName), stringValue(event.Transcript), strings.Join(event.Callsigns, " ")}
	for _, message := range event.DecoderMessages {
		parts = append(parts, message.DecoderID, message.Protocol, message.Summary, message.RawText, strings.Join(message.Callsigns, " "))
	}
	for _, token := range searchTokens(strings.Join(parts, " ")) {
		ids := s.searchIndex[token]
		if ids == nil {
			ids = make(map[string]struct{})
			s.searchIndex[token] = ids
		}
		ids[event.ID] = struct{}{}
	}
}

func searchTokens(value string) []string {
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	seen := make(map[string]bool, len(fields))
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" || seen[field] {
			continue
		}
		seen[field] = true
		result = append(result, field)
	}
	return result
}

func (s *EventStore) aggregate(event TransmissionEvent) {
	key := fmt.Sprintf("%.0f", event.FrequencyHz)
	summary, ok := s.signals[key]
	if !ok {
		s.signals[key] = SignalSummary{ID: key, FrequencyHz: event.FrequencyHz, FirstSeen: event.StartedAt, LastSeen: event.StartedAt, EventCount: 1, StrongestDBFS: event.SignalDBFS, Modulation: event.Modulation, ProtocolName: event.ProtocolName, Label: event.Label, Confidence: event.Confidence, Location: event.Location, CTCSSHz: event.CTCSSHz}
		return
	}
	if event.StartedAt.Before(summary.FirstSeen) {
		summary.FirstSeen = event.StartedAt
	}
	if event.StartedAt.After(summary.LastSeen) {
		summary.LastSeen = event.StartedAt
	}
	summary.EventCount++
	if event.SignalDBFS > summary.StrongestDBFS {
		summary.StrongestDBFS = event.SignalDBFS
	}
	if event.Confidence >= summary.Confidence {
		summary.Modulation = event.Modulation
		summary.ProtocolName = event.ProtocolName
		if event.Label != nil {
			summary.Label = event.Label
		}
		summary.Confidence = event.Confidence
	}
	if event.Location != nil {
		summary.Location = event.Location
	}
	if event.CTCSSHz != nil {
		summary.CTCSSHz = event.CTCSSHz
	}
	s.signals[key] = summary
}
