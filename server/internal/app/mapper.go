package app

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MapperConfig struct {
	WebhookURL            string   `json:"webhookURL"`
	SheetURL              string   `json:"sheetURL,omitempty"`
	Contributor           string   `json:"contributor,omitempty"`
	Secret                string   `json:"secret,omitempty"`
	AutoUpload            bool     `json:"autoUpload"`
	Mode                  string   `json:"mode,omitempty"`
	DeviceID              string   `json:"deviceID,omitempty"`
	StartHz               float64  `json:"startHz,omitempty"`
	EndHz                 float64  `json:"endHz,omitempty"`
	StepHz                float64  `json:"stepHz,omitempty"`
	DwellMilliseconds     int      `json:"dwellMilliseconds,omitempty"`
	SampleRateHz          int      `json:"sampleRateHz,omitempty"`
	DecipherListenSeconds int64    `json:"decipherListenSeconds,omitempty"`
	Transcribe            bool     `json:"transcribe"`
	IncludeLocation       bool     `json:"includeLocation"`
	LocationPrecision     string   `json:"locationPrecision,omitempty"`
	Latitude              *float64 `json:"latitude,omitempty"`
	Longitude             *float64 `json:"longitude,omitempty"`
	LocationLabel         string   `json:"locationLabel,omitempty"`
}

type MapperFrequencyRecord struct {
	FrequencyHz          float64              `json:"frequencyHz"`
	FirstSeen            time.Time            `json:"firstSeen"`
	LastSeen             time.Time            `json:"lastSeen"`
	Checks               int                  `json:"checks"`
	Hits                 int                  `json:"hits"`
	Occupancy            float64              `json:"occupancy"`
	StrongestDBFS        float64              `json:"strongestDBFS"`
	NoiseDBFS            float64              `json:"noiseDBFS"`
	Modulation           string               `json:"modulation"`
	ProtocolName         string               `json:"protocolName,omitempty"`
	Name                 string               `json:"name,omitempty"`
	Callsigns            []string             `json:"callsigns,omitempty"`
	Confidence           float64              `json:"confidence"`
	IdentificationSource string               `json:"identificationSource,omitempty"`
	CandidateDecoder     string               `json:"candidateDecoder,omitempty"`
	DetectionStatus      string               `json:"detectionStatus,omitempty"`
	DetectionEvidence    string               `json:"detectionEvidence,omitempty"`
	DecoderReady         bool                 `json:"decoderReady"`
	AnalysisEngine       string               `json:"analysisEngine,omitempty"`
	AnalysisSummary      string               `json:"analysisSummary,omitempty"`
	AnalysisEvidence     []string             `json:"analysisEvidence,omitempty"`
	HourlyHits           [24]int              `json:"hourlyHits"`
	ActivityTimeZone     string               `json:"activityTimeZone,omitempty"`
	Location             *ObservationLocation `json:"location,omitempty"`
	LastTranscript       string               `json:"lastTranscript,omitempty"`
}

type MapperStatus struct {
	Config       MapperConfig            `json:"config"`
	LastUpload   *time.Time              `json:"lastUpload,omitempty"`
	LastError    string                  `json:"lastError,omitempty"`
	UploadedRows int                     `json:"uploadedRows"`
	Records      []MapperFrequencyRecord `json:"records"`
}

type MapperProgress struct {
	Running            bool       `json:"running"`
	Mode               string     `json:"mode,omitempty"`
	CurrentFrequencyHz float64    `json:"currentFrequencyHz,omitempty"`
	CurrentLabel       string     `json:"currentLabel,omitempty"`
	CurrentIndex       int        `json:"currentIndex"`
	TotalTargets       int        `json:"totalTargets"`
	ChecksCompleted    int64      `json:"checksCompleted"`
	PassesCompleted    int        `json:"passesCompleted"`
	PassStartedAt      *time.Time `json:"passStartedAt,omitempty"`
	EstimatedPassEndAt *time.Time `json:"estimatedPassEndAt,omitempty"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	StoppedAt          *time.Time `json:"stoppedAt,omitempty"`
	TargetStartedAt    *time.Time `json:"targetStartedAt,omitempty"`
	TargetEndsAt       *time.Time `json:"targetEndsAt,omitempty"`
	LastCheckAt        *time.Time `json:"lastCheckAt,omitempty"`
	LastActivityAt     *time.Time `json:"lastActivityAt,omitempty"`
}

type MapperExportResult struct {
	Path string `json:"path"`
	Rows int    `json:"rows"`
}

type MapperManager struct {
	mu           sync.RWMutex
	path         string
	config       MapperConfig
	lastUpload   *time.Time
	lastError    string
	uploadedRows int
	lastSeen     map[string]time.Time
	records      map[string]MapperFrequencyRecord
	recordsPath  string
	events       *EventStore
	client       *http.Client
	progress     MapperProgress
	sessionID    uint64
	lastArchived map[string]time.Time
}

func NewMapperManager(dataDirectory string, events *EventStore) *MapperManager {
	m := &MapperManager{path: filepath.Join(dataDirectory, "Data", "mapper.json"), recordsPath: filepath.Join(dataDirectory, "Data", "mapper-records.json"), events: events, client: &http.Client{Timeout: 12 * time.Second}, lastSeen: make(map[string]time.Time), records: make(map[string]MapperFrequencyRecord), lastArchived: make(map[string]time.Time)}
	if data, err := os.ReadFile(m.path); err == nil {
		_ = json.Unmarshal(data, &m.config)
	}
	if data, err := os.ReadFile(m.recordsPath); err == nil {
		_ = json.Unmarshal(data, &m.records)
	}
	if pruneLegacyMapperFalsePositives(m.records) > 0 {
		m.persistRecords()
	}
	if m.config.Mode == "" {
		m.config.Mode = "discovery"
	}
	if m.config.DecipherListenSeconds == 0 {
		m.config.DecipherListenSeconds = 60
	}
	go m.loop()
	return m
}

// Older builds could create a hit before a per-frequency noise baseline was
// established. Those rows are uniquely identifiable by an impossible exact
// 0 dBFS noise value and a single 1/1 observation. Keep every normally
// measured record while removing only that known-bad legacy signature.
func pruneLegacyMapperFalsePositives(records map[string]MapperFrequencyRecord) int {
	removed := 0
	for key, record := range records {
		if record.Checks == 1 && record.Hits == 1 && record.NoiseDBFS == 0 {
			delete(records, key)
			removed++
		}
	}
	return removed
}

func (m *MapperManager) Status() MapperStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	records := make([]MapperFrequencyRecord, 0, len(m.records))
	for _, record := range m.records {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].FrequencyHz < records[j].FrequencyHz })
	if len(records) > 5000 {
		records = records[len(records)-5000:]
	}
	return MapperStatus{Config: m.config, LastUpload: m.lastUpload, LastError: m.lastError, UploadedRows: m.uploadedRows, Records: records}
}

func (m *MapperManager) Progress() MapperProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.progress
}

func (m *MapperManager) BeginSession(mode string, totalTargets int) uint64 {
	now := time.Now()
	m.mu.Lock()
	m.sessionID++
	sessionID := m.sessionID
	m.progress = MapperProgress{Running: true, Mode: mode, CurrentIndex: -1, TotalTargets: totalTargets, StartedAt: &now, PassStartedAt: &now}
	m.mu.Unlock()
	return sessionID
}

func (m *MapperManager) BeginTarget(sessionID uint64, index, totalTargets int, frequencyHz float64, label string, listenFor time.Duration) {
	now := time.Now()
	m.mu.Lock()
	if sessionID != m.sessionID {
		m.mu.Unlock()
		return
	}
	progress := &m.progress
	progress.Running = true
	progress.CurrentIndex = index
	progress.TotalTargets = totalTargets
	progress.CurrentFrequencyHz = frequencyHz
	progress.CurrentLabel = label
	progress.TargetStartedAt = &now
	if index == 0 || progress.PassStartedAt == nil {
		progress.PassStartedAt = &now
		progress.EstimatedPassEndAt = nil
	}
	if listenFor > 0 {
		endsAt := now.Add(listenFor)
		progress.TargetEndsAt = &endsAt
		remaining := time.Duration(totalTargets-index) * listenFor
		estimated := now.Add(remaining)
		progress.EstimatedPassEndAt = &estimated
	} else {
		progress.TargetEndsAt = nil
		if index > 0 && progress.PassStartedAt != nil {
			average := now.Sub(*progress.PassStartedAt) / time.Duration(index)
			estimated := now.Add(average * time.Duration(totalTargets-index))
			progress.EstimatedPassEndAt = &estimated
		}
	}
	m.mu.Unlock()
}

func (m *MapperManager) CompletePass(sessionID uint64) {
	m.mu.Lock()
	if sessionID == m.sessionID {
		m.progress.PassesCompleted++
	}
	m.mu.Unlock()
}

func (m *MapperManager) EndSession(sessionID uint64) {
	now := time.Now()
	m.mu.Lock()
	if sessionID != m.sessionID {
		m.mu.Unlock()
		return
	}
	m.progress.Running = false
	m.progress.StoppedAt = &now
	m.progress.TargetEndsAt = nil
	m.mu.Unlock()
}

func (m *MapperManager) Update(config MapperConfig) (MapperStatus, error) {
	config.WebhookURL = strings.TrimSpace(config.WebhookURL)
	config.SheetURL = strings.TrimSpace(config.SheetURL)
	config.Contributor = strings.TrimSpace(config.Contributor)
	if config.Contributor == "" {
		config.Contributor = "GP-SDR"
	}
	config.Mode = strings.ToLower(strings.TrimSpace(config.Mode))
	if config.Mode == "" {
		config.Mode = "discovery"
	}
	if config.Mode != "discovery" && config.Mode != "decipher" {
		return MapperStatus{}, errors.New("mapper mode must be discovery or decipher")
	}
	if config.DecipherListenSeconds == 0 {
		config.DecipherListenSeconds = 60
	}
	if config.DecipherListenSeconds < 5 || config.DecipherListenSeconds > 7*24*60*60 {
		return MapperStatus{}, errors.New("decipher listen time must be between 5 seconds and 7 days")
	}
	if !supportedUserSampleRate(config.SampleRateHz) {
		return MapperStatus{}, errors.New("choose Auto or a supported Mapper sample rate")
	}
	if config.IncludeLocation {
		if config.Latitude == nil || config.Longitude == nil || *config.Latitude < -90 || *config.Latitude > 90 || *config.Longitude < -180 || *config.Longitude > 180 {
			return MapperStatus{}, errors.New("add a valid latitude and longitude or turn location tagging off")
		}
		if config.LocationPrecision == "" {
			config.LocationPrecision = "approximate"
		}
	}
	if config.WebhookURL != "" {
		parsed, err := url.Parse(config.WebhookURL)
		if err != nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost"))) {
			return MapperStatus{}, errors.New("enter a valid HTTPS Google Apps Script webhook URL")
		}
	}
	if config.SheetURL != "" {
		parsed, err := url.Parse(config.SheetURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "docs.google.com" || !strings.Contains(parsed.Path, "/spreadsheets/") {
			return MapperStatus{}, errors.New("enter a valid Google Sheets URL")
		}
	}
	if config.AutoUpload && config.WebhookURL == "" {
		return MapperStatus{}, errors.New("add a webhook URL before enabling automatic upload")
	}
	m.mu.Lock()
	m.config = config
	m.lastError = ""
	data, _ := json.MarshalIndent(config, "", "  ")
	_ = os.MkdirAll(filepath.Dir(m.path), 0o700)
	err := os.WriteFile(m.path, data, 0o600)
	m.mu.Unlock()
	return m.Status(), err
}

func (m *MapperManager) Config() MapperConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *MapperManager) Observe(frequencyHz float64, active bool, signalDBFS, noiseDBFS float64, modulation, protocol, name, transcript string) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	now := time.Now()
	m.mu.Lock()
	m.progress.ChecksCompleted++
	m.progress.LastCheckAt = &now
	if active {
		m.progress.LastActivityAt = &now
	}
	record, exists := m.records[key]
	if !exists && !active {
		m.mu.Unlock()
		return
	}
	if !exists {
		record = MapperFrequencyRecord{FrequencyHz: frequencyHz, FirstSeen: now, StrongestDBFS: -200}
	}
	record.Checks++
	if active {
		record.Hits++
		record.LastSeen = now
		record.HourlyHits[now.Hour()]++
		record.ActivityTimeZone = now.Location().String()
		if signalDBFS > record.StrongestDBFS {
			record.StrongestDBFS = signalDBFS
		}
		record.NoiseDBFS = noiseDBFS
		if modulation != "" {
			record.Modulation = modulation
		}
		if protocol != "" {
			record.ProtocolName = protocol
		}
		if name != "" {
			record.Name = name
		}
		if transcript != "" {
			record.LastTranscript = transcript
		}
		record.Confidence = math.Max(record.Confidence, .72)
		if location := observationLocation(m.config); location != nil {
			record.Location = location
		}
	}
	if record.Checks > 0 {
		record.Occupancy = float64(record.Hits) / float64(record.Checks)
	}
	m.records[key] = record
	m.mu.Unlock()
}

func (m *MapperManager) SetIdentification(frequencyHz float64, source string, confidence float64) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists {
		record.IdentificationSource = strings.TrimSpace(source)
		record.Confidence = math.Max(record.Confidence, confidence)
		m.records[key] = record
	}
	m.mu.Unlock()
}

func (m *MapperManager) SetDecoderEvidence(frequencyHz float64, decoderID, status, evidence string, ready bool) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists {
		record.CandidateDecoder = canonicalDecoderID(decoderID)
		record.DetectionStatus = strings.TrimSpace(status)
		record.DetectionEvidence = strings.TrimSpace(evidence)
		record.DecoderReady = ready
		m.records[key] = record
	}
	m.mu.Unlock()
}

func (m *MapperManager) SetSignalIntelligence(frequencyHz float64, analysis SignalIntelligence) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists {
		record.AnalysisEngine = strings.TrimSpace(analysis.Engine)
		record.AnalysisSummary = strings.TrimSpace(analysis.Summary)
		record.AnalysisEvidence = append([]string(nil), analysis.Evidence...)
		// Local waveform classification is useful evidence, but it must not
		// replace an explicit mode supplied by an imported or saved channel.
		knownMode := strings.ToUpper(strings.TrimSpace(record.Modulation))
		if analysis.Modulation != "" && analysis.Modulation != "UNKNOWN" &&
			(knownMode == "" || knownMode == "AUTO" || knownMode == "UNKNOWN") {
			record.Modulation = analysis.Modulation
		}
		record.Confidence = math.Max(record.Confidence, analysis.Confidence)
		record.Callsigns = mergeUniqueStrings(record.Callsigns, analysis.Callsigns)
		m.records[key] = record
	}
	m.mu.Unlock()
}

func (m *MapperManager) SetDecodedMessages(frequencyHz float64, decoderID string, messages []DecoderMessage) {
	if len(messages) == 0 {
		return
	}
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists {
		record.CandidateDecoder = canonicalDecoderID(decoderID)
		record.DetectionStatus = "confirmed"
		record.DecoderReady = true
		record.ProtocolName = messages[0].Protocol
		record.Confidence = math.Max(record.Confidence, messages[0].Confidence)
		evidence := make([]string, 0, len(messages))
		for _, message := range messages {
			evidence = append(evidence, message.Summary)
			record.Callsigns = mergeUniqueStrings(record.Callsigns, message.Callsigns)
		}
		record.DetectionEvidence = strings.Join(evidence, " · ")
		m.records[key] = record
	}
	m.mu.Unlock()
}

func (m *MapperManager) SetTranscript(frequencyHz float64, transcript string) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists && strings.TrimSpace(transcript) != "" {
		record.LastTranscript = strings.TrimSpace(transcript)
		record.Callsigns = mergeUniqueStrings(record.Callsigns, ExtractCallsigns(transcript))
		m.records[key] = record
	}
	m.mu.Unlock()
}

func mergeUniqueStrings(existing, additions []string) []string {
	seen := make(map[string]bool, len(existing)+len(additions))
	for _, value := range append(append([]string(nil), existing...), additions...) {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (m *MapperManager) ShouldArchive(frequencyHz float64, interval time.Duration) bool {
	key := fmt.Sprintf("%.0f", frequencyHz)
	now := time.Now()
	m.mu.Lock()
	if m.lastArchived == nil {
		m.lastArchived = make(map[string]time.Time)
	}
	last := m.lastArchived[key]
	if !last.IsZero() && now.Sub(last) < interval {
		m.mu.Unlock()
		return false
	}
	m.lastArchived[key] = now
	m.mu.Unlock()
	return true
}

func observationLocation(config MapperConfig) *ObservationLocation {
	if !config.IncludeLocation || config.Latitude == nil || config.Longitude == nil {
		return nil
	}
	latitude, longitude := *config.Latitude, *config.Longitude
	switch config.LocationPrecision {
	case "city":
		latitude, longitude = math.Round(latitude*10)/10, math.Round(longitude*10)/10
	case "approximate":
		latitude, longitude = math.Round(latitude*100)/100, math.Round(longitude*100)/100
	}
	return &ObservationLocation{Latitude: latitude, Longitude: longitude, Label: strings.TrimSpace(config.LocationLabel), Precision: config.LocationPrecision}
}

func (m *MapperManager) UploadNow() MapperStatus {
	m.mu.Lock()
	if m.config.WebhookURL == "" {
		m.lastError = "add the Apps Script webhook URL before sending to the sheet"
		m.mu.Unlock()
		return m.Status()
	}
	m.mu.Unlock()
	m.upload()
	return m.Status()
}

func (m *MapperManager) ClearRecords() MapperStatus {
	m.mu.Lock()
	m.records = make(map[string]MapperFrequencyRecord)
	m.lastSeen = make(map[string]time.Time)
	m.uploadedRows = 0
	m.lastUpload = nil
	m.lastError = ""
	m.mu.Unlock()
	m.persistRecords()
	return m.Status()
}

func (m *MapperManager) CSV() ([]byte, int, error) {
	m.mu.RLock()
	records := make([]MapperFrequencyRecord, 0, len(m.records))
	for _, record := range m.records {
		records = append(records, record)
	}
	m.mu.RUnlock()
	sort.Slice(records, func(i, j int) bool { return records[i].FrequencyHz < records[j].FrequencyHz })

	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := []string{"frequency_hz", "frequency_mhz", "name", "modulation", "protocol", "candidate_decoder", "detection_status", "detection_evidence", "decoder_ready", "analysis_engine", "analysis_summary", "analysis_evidence", "callsigns", "first_seen", "last_seen", "checks", "hits", "occupancy", "strongest_dbfs", "noise_dbfs", "confidence", "transcript", "latitude", "longitude", "location_name", "location_precision", "identification_source", "peak_activity_hours", "activity_time_zone"}
	if err := writer.Write(header); err != nil {
		return nil, 0, err
	}
	for _, record := range records {
		latitude, longitude, locationName, locationPrecision := "", "", "", ""
		if record.Location != nil {
			latitude = strconv.FormatFloat(record.Location.Latitude, 'f', 6, 64)
			longitude = strconv.FormatFloat(record.Location.Longitude, 'f', 6, 64)
			locationName = safeSpreadsheetText(record.Location.Label)
			locationPrecision = safeSpreadsheetText(record.Location.Precision)
		}
		row := []string{
			strconv.FormatFloat(record.FrequencyHz, 'f', 0, 64), strconv.FormatFloat(record.FrequencyHz/1e6, 'f', 6, 64),
			safeSpreadsheetText(record.Name), safeSpreadsheetText(record.Modulation), safeSpreadsheetText(record.ProtocolName), safeSpreadsheetText(record.CandidateDecoder), safeSpreadsheetText(record.DetectionStatus), safeSpreadsheetText(record.DetectionEvidence), strconv.FormatBool(record.DecoderReady), safeSpreadsheetText(record.AnalysisEngine), safeSpreadsheetText(record.AnalysisSummary), safeSpreadsheetText(strings.Join(record.AnalysisEvidence, " | ")), safeSpreadsheetText(strings.Join(record.Callsigns, " | ")),
			record.FirstSeen.Format(time.RFC3339Nano), record.LastSeen.Format(time.RFC3339Nano), strconv.Itoa(record.Checks), strconv.Itoa(record.Hits),
			strconv.FormatFloat(record.Occupancy, 'f', 6, 64), strconv.FormatFloat(record.StrongestDBFS, 'f', 2, 64), strconv.FormatFloat(record.NoiseDBFS, 'f', 2, 64),
			strconv.FormatFloat(record.Confidence, 'f', 3, 64), safeSpreadsheetText(record.LastTranscript), latitude, longitude, locationName, locationPrecision,
			safeSpreadsheetText(record.IdentificationSource), safeSpreadsheetText(mapperPeakHours(record.HourlyHits)), safeSpreadsheetText(record.ActivityTimeZone),
		}
		if err := writer.Write(row); err != nil {
			return nil, 0, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, 0, err
	}
	return output.Bytes(), len(records), nil
}

func mapperPeakHours(hourly [24]int) string {
	type hourCount struct{ hour, count int }
	items := make([]hourCount, 0, 24)
	for hour, count := range hourly {
		if count > 0 {
			items = append(items, hourCount{hour: hour, count: count})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].hour < items[j].hour
	})
	if len(items) > 3 {
		items = items[:3]
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%02d:00-%02d:59 (%d)", item.hour, item.hour, item.count))
	}
	return strings.Join(parts, " | ")
}

func safeSpreadsheetText(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}

func (m *MapperManager) SaveCSV() (MapperExportResult, error) {
	data, rows, err := m.CSV()
	if err != nil {
		return MapperExportResult{}, err
	}
	directory := filepath.Join(filepath.Dir(filepath.Dir(m.recordsPath)), "Exports", "Mapper")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return MapperExportResult{}, err
	}
	path := filepath.Join(directory, "GP-SDR-Mapper-"+time.Now().Format("20060102-150405.000")+".csv")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return MapperExportResult{}, err
	}
	return MapperExportResult{Path: path, Rows: rows}, nil
}

func (m *MapperManager) loop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.RLock()
		enabled := m.config.AutoUpload
		m.mu.RUnlock()
		if enabled {
			m.upload()
		}
		m.persistRecords()
	}
}

func (m *MapperManager) persistRecords() {
	m.mu.RLock()
	data, err := json.MarshalIndent(m.records, "", "  ")
	m.mu.RUnlock()
	if err == nil {
		_ = os.MkdirAll(filepath.Dir(m.recordsPath), 0o700)
		_ = os.WriteFile(m.recordsPath, data, 0o600)
	}
}

func (m *MapperManager) upload() {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()
	if config.WebhookURL == "" {
		return
	}
	rows := make([]MapperFrequencyRecord, 0)
	m.mu.RLock()
	for key, signal := range m.records {
		if !signal.LastSeen.After(m.lastSeen[key]) {
			continue
		}
		rows = append(rows, signal)
	}
	m.mu.RUnlock()
	if len(rows) == 0 {
		return
	}
	err := m.postRows(config, rows)
	m.finishUpload(rows, err)
}

func (m *MapperManager) UploadFrequency(frequencyHz float64) MapperStatus {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.RLock()
	record, exists := m.records[key]
	config := m.config
	m.mu.RUnlock()
	if !exists {
		m.mu.Lock()
		m.lastError = "the selected Mapper record no longer exists"
		m.mu.Unlock()
		return m.Status()
	}
	if config.WebhookURL == "" {
		m.mu.Lock()
		m.lastError = "add the Apps Script webhook URL before sending to the sheet"
		m.mu.Unlock()
		return m.Status()
	}
	rows := []MapperFrequencyRecord{record}
	err := m.postRows(config, rows)
	m.finishUpload(rows, err)
	return m.Status()
}

func (m *MapperManager) postRows(config MapperConfig, rows []MapperFrequencyRecord) error {
	additions := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		additions = append(additions, mapperAdditionRow(row, config))
	}
	payload, _ := json.Marshal(map[string]any{
		"source": "GP-SDR Identify", "secret": config.Secret, "spreadsheetURL": config.SheetURL,
		"sheetName": "Additions Queue", "rows": additions, "signals": rows,
	})
	request, _ := http.NewRequest(http.MethodPost, config.WebhookURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err == nil && response != nil {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
		response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			err = errors.New("sheet webhook rejected the upload")
		} else if readErr != nil {
			err = readErr
		} else {
			var result struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &result) == nil && !result.OK {
				if result.Error == "" {
					result.Error = "sheet webhook did not confirm the write"
				}
				err = errors.New(result.Error)
			}
		}
	}
	return err
}

func (m *MapperManager) finishUpload(rows []MapperFrequencyRecord, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastError = err.Error()
		return
	}
	now := time.Now()
	m.lastUpload = &now
	m.lastError = ""
	m.uploadedRows += len(rows)
	for _, row := range rows {
		m.lastSeen[fmt.Sprintf("%.0f", row.FrequencyHz)] = row.LastSeen
	}
}

func mapperAdditionRow(record MapperFrequencyRecord, config MapperConfig) map[string]any {
	entryType := "Conventional"
	protocol := strings.ToUpper(record.ProtocolName + " " + record.Modulation)
	if strings.Contains(protocol, "P25") {
		entryType = "P25 System"
	}
	location := ""
	if record.Location != nil {
		location = strings.TrimSpace(record.Location.Label)
		if location == "" {
			location = fmt.Sprintf("%.6f, %.6f (%s)", record.Location.Latitude, record.Location.Longitude, record.Location.Precision)
		}
	}
	whatWasHeard := strings.TrimSpace(record.LastTranscript)
	if whatWasHeard == "" {
		whatWasHeard = fmt.Sprintf("RF activity detected on %d of %d checks (%.1f%% occupancy)", record.Hits, record.Checks, record.Occupancy*100)
	}
	confidence := "Heard once"
	if record.Hits >= 3 {
		confidence = "Repeated observation"
	}
	notes := fmt.Sprintf("GP-SDR Identify: signal %.1f dBFS; local noise %.1f dBFS; confidence %.0f%%", record.StrongestDBFS, record.NoiseDBFS, record.Confidence*100)
	if record.AnalysisSummary != "" {
		notes += "; " + record.AnalysisSummary
	}
	if len(record.Callsigns) > 0 {
		notes += "; decoded callsigns: " + strings.Join(record.Callsigns, ", ")
	}
	return map[string]any{
		"dateAdded": time.Now().Format("2006-01-02"), "contributor": config.Contributor, "type": entryType,
		"nameLabel": record.Name, "rxMHz": record.FrequencyHz / 1e6, "txMHz": "", "mode": firstNonEmpty(record.ProtocolName, record.Modulation),
		"toneCode": "", "locationSystem": location, "whatWasHeard": whatWasHeard, "dateTimeHeard": record.LastSeen.Format(time.RFC3339),
		"sourceURLFile": "GP-SDR Mapper observation", "confidence": confidence, "reviewStatus": "New", "reviewerNotes": notes,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "Unknown"
}
