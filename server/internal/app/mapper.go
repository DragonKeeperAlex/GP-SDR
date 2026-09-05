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
	WebhookURL         string `json:"webhookURL"`
	SheetURL           string `json:"sheetURL,omitempty"`
	Contributor        string `json:"contributor,omitempty"`
	Secret             string `json:"secret,omitempty"`
	AutoUpload         bool   `json:"autoUpload"`
	UploadVerifiedOnly bool   `json:"uploadVerifiedOnly"`
	Mode               string `json:"mode,omitempty"`
	PreferredMode      string `json:"preferredMode,omitempty"`
	PreferredDecoder   string `json:"preferredDecoder,omitempty"`
	DeviceID           string `json:"deviceID,omitempty"`
	// UseAllReceivers fans one Mapper template out to every currently connected
	// receiver when it is started from the Mapper page.
	UseAllReceivers          bool     `json:"useAllReceivers,omitempty"`
	StartHz                  float64  `json:"startHz,omitempty"`
	EndHz                    float64  `json:"endHz,omitempty"`
	StepHz                   float64  `json:"stepHz,omitempty"`
	DwellMilliseconds        int      `json:"dwellMilliseconds,omitempty"`
	SampleRateHz             int      `json:"sampleRateHz,omitempty"`
	ConcurrentChannels       int      `json:"concurrentChannels,omitempty"`
	GainMode                 string   `json:"gainMode,omitempty"`
	GainDB                   float64  `json:"gainDB,omitempty"`
	LNAGainDB                int      `json:"lnaGainDB,omitempty"`
	VGAGainDB                int      `json:"vgaGainDB,omitempty"`
	AmpMode                  string   `json:"ampMode,omitempty"`
	Sensitivity              string   `json:"sensitivity,omitempty"`
	NoiseMarginDB            float64  `json:"noiseMarginDB,omitempty"`
	DecipherListenSeconds    int64    `json:"decipherListenSeconds,omitempty"`
	IdentifyMinimumHits      int      `json:"identifyMinimumHits,omitempty"`
	IdentifyHitSource        string   `json:"identifyHitSource,omitempty"`
	IdentifyMinimumOccupancy float64  `json:"identifyMinimumOccupancy,omitempty"`
	IdentifySeenWithinHours  int      `json:"identifySeenWithinHours,omitempty"`
	IdentifyMaximumChannels  int      `json:"identifyMaximumChannels,omitempty"`
	IdentifyOrder            string   `json:"identifyOrder,omitempty"`
	Transcribe               bool     `json:"transcribe"`
	IncludeLocation          bool     `json:"includeLocation"`
	LocationPrecision        string   `json:"locationPrecision,omitempty"`
	Latitude                 *float64 `json:"latitude,omitempty"`
	Longitude                *float64 `json:"longitude,omitempty"`
	LocationLabel            string   `json:"locationLabel,omitempty"`
	// AnalysisPolicy controls when expensive decoder, transcription, and local
	// model work runs: live, after-job, or manual. RF detection and bounded IQ
	// evidence capture always remain real-time.
	CapturePolicy            string `json:"capturePolicy,omitempty"`
	AnalysisPolicy           string `json:"analysisPolicy,omitempty"`
	RejectedIQPolicy         string `json:"rejectedIQPolicy,omitempty"`
	ScheduleEnabled          bool   `json:"scheduleEnabled,omitempty"`
	DiscoveryDurationSeconds int64  `json:"discoveryDurationSeconds,omitempty"`
	IdentifyDurationSeconds  int64  `json:"identifyDurationSeconds,omitempty"`
	ScheduleRepeat           bool   `json:"scheduleRepeat,omitempty"`
}

type MapperFrequencyRecord struct {
	FirstCheckedAt         time.Time            `json:"firstCheckedAt,omitempty"`
	LastCheckedAt          time.Time            `json:"lastCheckedAt,omitempty"`
	FrequencyHz            float64              `json:"frequencyHz"`
	FirstSeen              time.Time            `json:"firstSeen"`
	LastSeen               time.Time            `json:"lastSeen"`
	Checks                 int                  `json:"checks"`
	Hits                   int                  `json:"hits"`
	DiscoveryChecks        int                  `json:"discoveryChecks,omitempty"`
	DiscoveryHits          int                  `json:"discoveryHits,omitempty"`
	IdentifyChecks         int                  `json:"identifyChecks,omitempty"`
	IdentifyHits           int                  `json:"identifyHits,omitempty"`
	Occupancy              float64              `json:"occupancy"`
	StrongestDBFS          float64              `json:"strongestDBFS"`
	NoiseDBFS              float64              `json:"noiseDBFS"`
	Modulation             string               `json:"modulation"`
	ProtocolName           string               `json:"protocolName,omitempty"`
	Name                   string               `json:"name,omitempty"`
	Callsigns              []string             `json:"callsigns,omitempty"`
	Confidence             float64              `json:"confidence"`
	IdentificationSource   string               `json:"identificationSource,omitempty"`
	IdentificationVerified bool                 `json:"identificationVerified"`
	VerificationReason     string               `json:"verificationReason,omitempty"`
	ReferenceDistanceMiles *float64             `json:"referenceDistanceMiles,omitempty"`
	CandidateDecoder       string               `json:"candidateDecoder,omitempty"`
	DetectionStatus        string               `json:"detectionStatus,omitempty"`
	DetectionEvidence      string               `json:"detectionEvidence,omitempty"`
	DecoderReady           bool                 `json:"decoderReady"`
	AnalysisEngine         string               `json:"analysisEngine,omitempty"`
	AnalysisSummary        string               `json:"analysisSummary,omitempty"`
	AnalysisEvidence       []string             `json:"analysisEvidence,omitempty"`
	HourlyHits             [24]int              `json:"hourlyHits"`
	ActivityTimeZone       string               `json:"activityTimeZone,omitempty"`
	Location               *ObservationLocation `json:"location,omitempty"`
	LastTranscript         string               `json:"lastTranscript,omitempty"`
	JobIDs                 []string             `json:"jobIDs,omitempty"`
	DeviceIDs              []string             `json:"deviceIDs,omitempty"`
}

type MapperJob struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Config    MapperConfig   `json:"config"`
	State     string         `json:"state"`
	LastError string         `json:"lastError,omitempty"`
	Progress  MapperProgress `json:"progress"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type MapperStatus struct {
	Config           MapperConfig            `json:"config"`
	LastUpload       *time.Time              `json:"lastUpload,omitempty"`
	LastError        string                  `json:"lastError,omitempty"`
	UploadedRows     int                     `json:"uploadedRows"`
	VerifiedRecords  int                     `json:"verifiedRecords"`
	AnalyzedRecords  int                     `json:"analyzedRecords"`
	CandidateRecords int                     `json:"candidateRecords"`
	Records          []MapperFrequencyRecord `json:"records"`
	Jobs             []MapperJob             `json:"jobs"`
}

type MapperProgress struct {
	Running            bool               `json:"running"`
	Mode               string             `json:"mode,omitempty"`
	CurrentFrequencyHz float64            `json:"currentFrequencyHz,omitempty"`
	CurrentFrequencies []float64          `json:"currentFrequenciesHz,omitempty"`
	CurrentLabel       string             `json:"currentLabel,omitempty"`
	CurrentIndex       int                `json:"currentIndex"`
	TotalTargets       int                `json:"totalTargets"`
	CurrentBatch       int                `json:"currentBatch"`
	TotalBatches       int                `json:"totalBatches"`
	MonitoredChannels  int                `json:"monitoredChannels"`
	ChecksCompleted    int64              `json:"checksCompleted"`
	PassesCompleted    int                `json:"passesCompleted"`
	PassStartedAt      *time.Time         `json:"passStartedAt,omitempty"`
	EstimatedPassEndAt *time.Time         `json:"estimatedPassEndAt,omitempty"`
	StartedAt          *time.Time         `json:"startedAt,omitempty"`
	StoppedAt          *time.Time         `json:"stoppedAt,omitempty"`
	TargetStartedAt    *time.Time         `json:"targetStartedAt,omitempty"`
	TargetEndsAt       *time.Time         `json:"targetEndsAt,omitempty"`
	PhaseEndsAt        *time.Time         `json:"phaseEndsAt,omitempty"`
	LastCheckAt        *time.Time         `json:"lastCheckAt,omitempty"`
	LastActivityAt     *time.Time         `json:"lastActivityAt,omitempty"`
	Tuning             MapperTuningStatus `json:"tuning"`
}

// MapperTuningStatus exposes the settings that Auto actually selected. This
// keeps "Auto" truthful and lets an operator reproduce a useful combination.
type MapperTuningStatus struct {
	Mode           string  `json:"mode,omitempty"`
	GainDB         float64 `json:"gainDB,omitempty"`
	LNAGainDB      int     `json:"lnaGainDB,omitempty"`
	VGAGainDB      int     `json:"vgaGainDB,omitempty"`
	AmpEnabled     bool    `json:"ampEnabled"`
	NoiseMarginDB  float64 `json:"noiseMarginDB,omitempty"`
	RMSDBFS        float64 `json:"rmsDBFS,omitempty"`
	PeakDBFS       float64 `json:"peakDBFS,omitempty"`
	ClippedPercent float64 `json:"clippedPercent,omitempty"`
	Decision       string  `json:"decision,omitempty"`
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
	jobsPath     string
	jobs         map[string]MapperJob
	jobSessions  map[string]uint64
	lastArchived map[string]time.Time
}

func NewMapperManager(dataDirectory string, events *EventStore) *MapperManager {
	m := &MapperManager{path: filepath.Join(dataDirectory, "Data", "mapper.json"), recordsPath: filepath.Join(dataDirectory, "Data", "mapper-records.json"), jobsPath: filepath.Join(dataDirectory, "Data", "mapper-jobs.json"), events: events, client: &http.Client{Timeout: 12 * time.Second}, lastSeen: make(map[string]time.Time), records: make(map[string]MapperFrequencyRecord), jobs: make(map[string]MapperJob), jobSessions: make(map[string]uint64), lastArchived: make(map[string]time.Time)}
	configLoaded := false
	if data, err := os.ReadFile(m.path); err == nil {
		configLoaded = json.Unmarshal(data, &m.config) == nil
		var rawConfig map[string]json.RawMessage
		if json.Unmarshal(data, &rawConfig) == nil {
			if _, exists := rawConfig["uploadVerifiedOnly"]; !exists {
				m.config.UploadVerifiedOnly = true
			}
		}
	}
	if !configLoaded {
		m.config.UploadVerifiedOnly = true
	}
	if data, err := os.ReadFile(m.recordsPath); err == nil {
		_ = json.Unmarshal(data, &m.records)
	}
	if data, err := os.ReadFile(m.jobsPath); err == nil {
		_ = json.Unmarshal(data, &m.jobs)
	}
	for id, job := range m.jobs {
		job.State = "idle"
		job.Progress.Running = false
		job.Config.Secret = ""
		m.jobs[id] = job
	}
	if pruneLegacyMapperFalsePositives(m.records) > 0 {
		m.persistRecords()
	}
	if pruneInvalidMapperDecoderBanners(m.records) > 0 {
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

func pruneInvalidMapperDecoderBanners(records map[string]MapperFrequencyRecord) int {
	updated := 0
	for key, record := range records {
		if canonicalDecoderID(record.CandidateDecoder) != "dsd-fme" || decoderLineIsEvidence("dsd-fme", record.DetectionEvidence) {
			continue
		}
		if record.DetectionStatus != "confirmed" && !record.IdentificationVerified {
			continue
		}
		record.DetectionStatus, record.DetectionEvidence = "candidate", ""
		if strings.HasPrefix(record.VerificationReason, "Valid ") {
			record.IdentificationVerified, record.VerificationReason, record.ReferenceDistanceMiles = false, "", nil
		}
		records[key], updated = record, updated+1
	}
	return updated
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
	verifiedRecords, analyzedRecords, candidateRecords := 0, 0, 0
	for _, record := range records {
		if mapperRecordFullyIdentified(record) {
			verifiedRecords++
		}
		if strings.TrimSpace(record.AnalysisSummary) != "" {
			analyzedRecords++
		}
		if strings.TrimSpace(record.Name) != "" || strings.TrimSpace(record.ProtocolName) != "" || strings.TrimSpace(record.CandidateDecoder) != "" {
			candidateRecords++
		}
	}
	// Keep the bounded UI response useful on very large sweeps. Previously the
	// highest 5,000 frequencies won, silently hiding every verified lower-band
	// result and making the counter read zero.
	sort.Slice(records, func(i, j int) bool {
		iv, jv := mapperRecordFullyIdentified(records[i]), mapperRecordFullyIdentified(records[j])
		if iv != jv {
			return iv
		}
		ih, jh := records[i].Hits > 0, records[j].Hits > 0
		if ih != jh {
			return ih
		}
		ia, ja := strings.TrimSpace(records[i].AnalysisSummary) != "", strings.TrimSpace(records[j].AnalysisSummary) != ""
		if ia != ja {
			return ia
		}
		return records[i].LastSeen.After(records[j].LastSeen)
	})
	if len(records) > 5000 {
		records = records[:5000]
	}
	sort.Slice(records, func(i, j int) bool { return records[i].FrequencyHz < records[j].FrequencyHz })
	jobs := make([]MapperJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].State != jobs[j].State {
			return jobs[i].State == "running"
		}
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	return MapperStatus{Config: m.config, LastUpload: m.lastUpload, LastError: m.lastError, UploadedRows: m.uploadedRows, VerifiedRecords: verifiedRecords, AnalyzedRecords: analyzedRecords, CandidateRecords: candidateRecords, Records: records, Jobs: jobs}
}

func (m *MapperManager) Progress() MapperProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, job := range m.jobs {
		if job.Progress.Running {
			return job.Progress
		}
	}
	return m.progress
}

func (m *MapperManager) Job(id string) (MapperJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	return job, ok
}

func (m *MapperManager) Jobs() []MapperJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jobs := make([]MapperJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.Before(jobs[j].CreatedAt) })
	return jobs
}

func validateMapperScanConfig(config MapperConfig) (MapperConfig, error) {
	config.Mode = strings.ToLower(strings.TrimSpace(config.Mode))
	if config.Mode == "" {
		config.Mode = "discovery"
	}
	if config.Mode != "adaptive" && config.Mode != "discovery" && config.Mode != "decipher" {
		return config, errors.New("Mapper workflow must be Map, Discovery, or Identify")
	}
	config.AnalysisPolicy = strings.ToLower(strings.TrimSpace(config.AnalysisPolicy))
	if config.AnalysisPolicy == "" {
		if config.Mode == "discovery" {
			config.AnalysisPolicy = "manual"
		} else {
			config.AnalysisPolicy = "live"
		}
	}
	if config.AnalysisPolicy != "live" && config.AnalysisPolicy != "after-job" && config.AnalysisPolicy != "manual" {
		return config, errors.New("analysis timing must be Live, after job, or manual")
	}
	if config.CapturePolicy == "" {
		config.CapturePolicy = "channel"
	}
	if config.CapturePolicy != "channel" && config.CapturePolicy != "archive" {
		return config, errors.New("capture policy must be channel or archive")
	}
	config.RejectedIQPolicy = strings.ToLower(strings.TrimSpace(config.RejectedIQPolicy))
	if config.RejectedIQPolicy == "" {
		config.RejectedIQPolicy = "delete"
	}
	if config.RejectedIQPolicy != "delete" && config.RejectedIQPolicy != "quarantine" {
		return config, errors.New("rejected IQ policy must be delete or quarantine")
	}
	if config.ScheduleEnabled {
		if config.DiscoveryDurationSeconds < 5 || config.DiscoveryDurationSeconds > 7*24*60*60 {
			return config, errors.New("scheduled Discovery must run between 5 seconds and 7 days")
		}
		if config.IdentifyDurationSeconds < 5 || config.IdentifyDurationSeconds > 7*24*60*60 {
			return config, errors.New("scheduled Identify must run between 5 seconds and 7 days")
		}
	}
	config.PreferredMode = strings.ToLower(strings.TrimSpace(config.PreferredMode))
	if config.PreferredMode == "" {
		config.PreferredMode = "auto"
	}
	if config.PreferredMode != "auto" && config.PreferredMode != "am" && config.PreferredMode != "nfm" && config.PreferredMode != "wfm" {
		if digitalVoiceProtocol(config.PreferredMode) == "" {
			return config, errors.New("Mapper modulation is not supported")
		}
	}
	config.PreferredDecoder = strings.ToLower(strings.TrimSpace(config.PreferredDecoder))
	if config.PreferredDecoder == "auto" {
		config.PreferredDecoder = ""
	}
	if config.PreferredDecoder != "" {
		known := map[string]bool{"dsd-fme": true, "dmr": true, "nxdn": true, "d-star": true, "ysf": true, "m17": true,
			"rtl-433": true, "dump1090": true, "multimon-ng": true, "acarsdec": true, "ais": true}
		if !known[config.PreferredDecoder] {
			return config, errors.New("Mapper decoder is not supported")
		}
	}
	if strings.TrimSpace(config.DeviceID) == "" {
		return config, errors.New("choose a receiver for this Mapper job")
	}
	if config.Mode != "decipher" || config.ScheduleEnabled {
		if !isFinitePositive(config.StartHz) || !isFinitePositive(config.EndHz) || config.EndHz < config.StartHz {
			return config, errors.New("enter a valid discovery frequency range")
		}
		if config.StepHz <= 0 {
			return config, errors.New("discovery step must be greater than zero")
		}
		if config.DwellMilliseconds < 100 || int64(config.DwellMilliseconds) > 7*24*60*60*1000 {
			return config, errors.New("Mapper dwell must be between 0.1 seconds and 7 days")
		}
	}
	if config.DecipherListenSeconds == 0 {
		config.DecipherListenSeconds = 60
	}
	if config.DecipherListenSeconds < 5 || config.DecipherListenSeconds > 7*24*60*60 {
		return config, errors.New("Identify listen time must be between 5 seconds and 7 days")
	}
	if config.Mode == "decipher" && config.IdentifyMinimumHits == 0 {
		config.IdentifyMinimumHits = 2
	}
	if config.IdentifyMinimumHits < 0 || config.IdentifyMinimumHits > 10_000 {
		return config, errors.New("Identify minimum hits must be between 1 and 10,000, or zero for the default")
	}
	config.IdentifyHitSource = strings.ToLower(strings.TrimSpace(config.IdentifyHitSource))
	if config.IdentifyHitSource == "" {
		config.IdentifyHitSource = "discovery"
	}
	if config.IdentifyHitSource != "discovery" && config.IdentifyHitSource != "combined" {
		return config, errors.New("Identify hit source must be Discovery or combined history")
	}
	if config.IdentifyMinimumOccupancy < 0 || config.IdentifyMinimumOccupancy > 1 {
		return config, errors.New("Identify minimum occupancy must be between 0 and 100 percent")
	}
	if config.IdentifySeenWithinHours < 0 || config.IdentifySeenWithinHours > 24*365 {
		return config, errors.New("Identify recent activity window must be between zero and one year")
	}
	if config.IdentifyMaximumChannels < 0 || config.IdentifyMaximumChannels > 10_000 {
		return config, errors.New("Identify channel limit must be between zero and 10,000")
	}
	config.IdentifyOrder = strings.ToLower(strings.TrimSpace(config.IdentifyOrder))
	if config.IdentifyOrder == "" {
		config.IdentifyOrder = "hits"
	}
	if config.IdentifyOrder != "hits" && config.IdentifyOrder != "recent" && config.IdentifyOrder != "occupancy" && config.IdentifyOrder != "frequency" && config.IdentifyOrder != "oldest" {
		return config, errors.New("choose a supported Identify channel order")
	}
	if !supportedUserSampleRate(config.SampleRateHz) {
		return config, errors.New("choose Auto or a supported Mapper sample rate")
	}
	if config.ConcurrentChannels == 0 {
		config.ConcurrentChannels = defaultMapperConcurrentChannels(config.Mode)
	}
	if config.ConcurrentChannels < 1 || config.ConcurrentChannels > 1024 {
		return config, errors.New("simultaneous Mapper channels must be between 1 and 1,024")
	}
	config.GainMode = strings.ToLower(strings.TrimSpace(config.GainMode))
	if config.GainMode == "" {
		config.GainMode = "auto"
	}
	if config.GainMode != "auto" && config.GainMode != "saved" && config.GainMode != "manual" {
		return config, errors.New("receiver tuning must be Auto, saved calibration, or manual")
	}
	if config.GainDB < 0 || config.GainDB > 62 {
		return config, errors.New("receiver gain must be between 0 and 62 dB")
	}
	if config.LNAGainDB < 0 || config.LNAGainDB > 40 || config.LNAGainDB%8 != 0 {
		return config, errors.New("HackRF LNA gain must be 0 to 40 dB in 8 dB steps")
	}
	if config.VGAGainDB < 0 || config.VGAGainDB > 62 || config.VGAGainDB%2 != 0 {
		return config, errors.New("HackRF VGA gain must be 0 to 62 dB in 2 dB steps")
	}
	config.AmpMode = strings.ToLower(strings.TrimSpace(config.AmpMode))
	if config.AmpMode == "" {
		config.AmpMode = "auto"
	}
	if config.AmpMode != "auto" && config.AmpMode != "off" && config.AmpMode != "on" {
		return config, errors.New("HackRF RF amplifier must be Auto, off, or on")
	}
	config.Sensitivity = strings.ToLower(strings.TrimSpace(config.Sensitivity))
	if config.Sensitivity == "" {
		config.Sensitivity = "auto"
	}
	if config.Sensitivity != "auto" && config.Sensitivity != "weak" && config.Sensitivity != "balanced" && config.Sensitivity != "conservative" && config.Sensitivity != "manual" {
		return config, errors.New("Mapper sensitivity must be Auto, weak, balanced, conservative, or manual")
	}
	if config.NoiseMarginDB == 0 {
		config.NoiseMarginDB = 6
	}
	if config.NoiseMarginDB < 3 || config.NoiseMarginDB > 30 {
		return config, errors.New("Mapper signal margin must be between 3 and 30 dB")
	}
	if config.IncludeLocation {
		if config.Latitude == nil || config.Longitude == nil || *config.Latitude < -90 || *config.Latitude > 90 || *config.Longitude < -180 || *config.Longitude > 180 {
			return config, errors.New("add a valid latitude and longitude or turn location tagging off")
		}
		if config.LocationPrecision == "" {
			config.LocationPrecision = "approximate"
		}
	}
	return config, nil
}

func (m *MapperManager) UpdateJobTuning(id string, sessionID uint64, status MapperTuningStatus) {
	m.mu.Lock()
	if m.jobSessions[id] == sessionID {
		job := m.jobs[id]
		job.Progress.Tuning = status
		m.jobs[id] = job
	}
	m.mu.Unlock()
}

func defaultMapperConcurrentChannels(mode string) int {
	if strings.EqualFold(mode, "decipher") {
		return 1
	}
	if strings.EqualFold(mode, "discovery") {
		return 512
	}
	return 64
}

func (m *MapperManager) SaveJob(job MapperJob) (MapperJob, error) {
	config, err := validateMapperScanConfig(job.Config)
	if err != nil {
		return MapperJob{}, err
	}
	now := time.Now()
	job.ID = strings.TrimSpace(job.ID)
	job.Name = strings.TrimSpace(job.Name)
	if job.ID == "" {
		job.ID = NewID()
		job.CreatedAt = now
	}
	if job.Name == "" {
		if config.Mode == "decipher" {
			job.Name = "Identify · found frequencies"
		} else if config.Mode == "adaptive" {
			job.Name = "Map · " + fmt.Sprintf("%.3f–%.3f MHz", config.StartHz/1e6, config.EndHz/1e6)
		} else {
			job.Name = "Discovery · " + fmt.Sprintf("%.3f–%.3f MHz", config.StartHz/1e6, config.EndHz/1e6)
		}
	}
	job.Config = config
	job.UpdatedAt = now
	m.mu.Lock()
	if existing, exists := m.jobs[job.ID]; exists {
		if existing.State == "running" {
			m.mu.Unlock()
			return MapperJob{}, errors.New("stop this Mapper job before editing it")
		}
		job.CreatedAt = existing.CreatedAt
	}
	job.State = "idle"
	job.LastError = ""
	job.Progress = MapperProgress{Mode: config.Mode, CurrentIndex: -1}
	m.jobs[job.ID] = job
	m.mu.Unlock()
	m.persistJobs()
	return job, nil
}

func (m *MapperManager) DeleteJob(id string) error {
	m.mu.Lock()
	job, exists := m.jobs[id]
	if !exists {
		m.mu.Unlock()
		return ErrNotFound
	}
	if job.State == "running" || job.State == "stopping" {
		m.mu.Unlock()
		return errors.New("stop this Mapper job before deleting it")
	}
	delete(m.jobs, id)
	delete(m.jobSessions, id)
	m.mu.Unlock()
	m.persistJobs()
	return nil
}

func (m *MapperManager) SetJobError(id, message string) {
	m.mu.Lock()
	if job, ok := m.jobs[id]; ok {
		job.State = "error"
		job.LastError = strings.TrimSpace(message)
		job.Progress.Running = false
		job.UpdatedAt = time.Now()
		m.jobs[id] = job
	}
	m.mu.Unlock()
	m.persistJobs()
}

func (m *MapperManager) MarkJobStopping(id string) {
	m.mu.Lock()
	if job, ok := m.jobs[id]; ok && job.State == "running" {
		job.State = "stopping"
		m.jobs[id] = job
	}
	m.mu.Unlock()
}

func (m *MapperManager) BeginJobSession(id string, totalTargets, totalBatches int) uint64 {
	now := time.Now()
	m.mu.Lock()
	m.sessionID++
	sessionID := m.sessionID
	m.jobSessions[id] = sessionID
	if job, ok := m.jobs[id]; ok {
		job.State = "running"
		job.LastError = ""
		job.Progress = MapperProgress{Running: true, Mode: job.Config.Mode, CurrentIndex: -1, CurrentBatch: -1, TotalTargets: totalTargets, TotalBatches: totalBatches, StartedAt: &now, PassStartedAt: &now}
		job.UpdatedAt = now
		m.jobs[id] = job
	}
	m.mu.Unlock()
	return sessionID
}

func (m *MapperManager) SetJobPhase(id string, sessionID uint64, mode string, endsAt *time.Time) {
	m.mu.Lock()
	if m.jobSessions[id] == sessionID {
		job := m.jobs[id]
		job.Progress.Mode = mode
		job.Progress.PhaseEndsAt = endsAt
		m.jobs[id] = job
	}
	m.mu.Unlock()
}

func (m *MapperManager) BeginJobBatch(id string, sessionID uint64, batchIndex, totalBatches, targetIndex, totalTargets int, frequencies []float64, label string, listenFor time.Duration) {
	now := time.Now()
	m.mu.Lock()
	if m.jobSessions[id] != sessionID {
		m.mu.Unlock()
		return
	}
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	progress := &job.Progress
	progress.Running = true
	progress.CurrentIndex = targetIndex
	progress.TotalTargets = totalTargets
	progress.CurrentBatch = batchIndex
	progress.TotalBatches = totalBatches
	progress.MonitoredChannels = len(frequencies)
	progress.CurrentFrequencies = append([]float64(nil), frequencies...)
	if len(frequencies) > 0 {
		progress.CurrentFrequencyHz = frequencies[0]
	}
	progress.CurrentLabel = label
	progress.TargetStartedAt = &now
	if batchIndex == 0 || progress.PassStartedAt == nil {
		progress.PassStartedAt = &now
		progress.EstimatedPassEndAt = nil
	}
	if listenFor > 0 {
		endsAt := now.Add(listenFor)
		progress.TargetEndsAt = &endsAt
		estimated := now.Add(time.Duration(totalBatches-batchIndex) * listenFor)
		progress.EstimatedPassEndAt = &estimated
	} else {
		progress.TargetEndsAt = nil
		if batchIndex > 0 && progress.PassStartedAt != nil {
			average := now.Sub(*progress.PassStartedAt) / time.Duration(batchIndex)
			estimated := now.Add(average * time.Duration(totalBatches-batchIndex))
			progress.EstimatedPassEndAt = &estimated
		}
	}
	job.Progress = *progress
	m.jobs[id] = job
	m.mu.Unlock()
}

func (m *MapperManager) BeginJobTarget(id string, sessionID uint64, index, totalTargets int, frequencyHz float64, label string, listenFor time.Duration) {
	m.BeginJobBatch(id, sessionID, index, totalTargets, index, totalTargets, []float64{frequencyHz}, label, listenFor)
}

func (m *MapperManager) CompleteJobPass(id string, sessionID uint64) {
	m.mu.Lock()
	if m.jobSessions[id] == sessionID {
		job := m.jobs[id]
		job.Progress.PassesCompleted++
		m.jobs[id] = job
	}
	m.mu.Unlock()
}

func (m *MapperManager) EndJobSession(id string, sessionID uint64) {
	now := time.Now()
	m.mu.Lock()
	if m.jobSessions[id] == sessionID {
		job := m.jobs[id]
		job.State = "idle"
		job.Progress.Running = false
		job.Progress.StoppedAt = &now
		job.Progress.TargetEndsAt = nil
		job.UpdatedAt = now
		m.jobs[id] = job
		delete(m.jobSessions, id)
	}
	m.mu.Unlock()
	m.persistJobs()
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
	if config.Mode != "adaptive" && config.Mode != "discovery" && config.Mode != "decipher" {
		return MapperStatus{}, errors.New("Mapper mode must be Map, Discovery, or Identify")
	}
	config.AnalysisPolicy = strings.ToLower(strings.TrimSpace(config.AnalysisPolicy))
	if config.AnalysisPolicy == "" {
		if config.Mode == "discovery" {
			config.AnalysisPolicy = "manual"
		} else {
			config.AnalysisPolicy = "live"
		}
	}
	if config.AnalysisPolicy != "live" && config.AnalysisPolicy != "after-job" && config.AnalysisPolicy != "manual" {
		return MapperStatus{}, errors.New("analysis timing must be Live, after job, or manual")
	}
	if config.CapturePolicy == "" {
		config.CapturePolicy = "channel"
	}
	if config.CapturePolicy != "channel" && config.CapturePolicy != "archive" {
		return MapperStatus{}, errors.New("capture policy must be channel or archive")
	}
	config.RejectedIQPolicy = strings.ToLower(strings.TrimSpace(config.RejectedIQPolicy))
	if config.RejectedIQPolicy == "" {
		config.RejectedIQPolicy = "delete"
	}
	if config.RejectedIQPolicy != "delete" && config.RejectedIQPolicy != "quarantine" {
		return MapperStatus{}, errors.New("rejected IQ policy must be delete or quarantine")
	}
	if config.DecipherListenSeconds == 0 {
		config.DecipherListenSeconds = 60
	}
	if config.DecipherListenSeconds < 5 || config.DecipherListenSeconds > 7*24*60*60 {
		return MapperStatus{}, errors.New("Identify listen time must be between 5 seconds and 7 days")
	}
	if !supportedUserSampleRate(config.SampleRateHz) {
		return MapperStatus{}, errors.New("choose Auto or a supported Mapper sample rate")
	}
	if config.ConcurrentChannels == 0 {
		config.ConcurrentChannels = defaultMapperConcurrentChannels(config.Mode)
	}
	if config.ConcurrentChannels < 1 || config.ConcurrentChannels > 1024 {
		return MapperStatus{}, errors.New("simultaneous Mapper channels must be between 1 and 1,024")
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
	m.ObserveJob("", "", m.Config(), frequencyHz, active, signalDBFS, noiseDBFS, modulation, protocol, name, transcript)
}

func (m *MapperManager) ObserveJob(jobID, deviceID string, config MapperConfig, frequencyHz float64, active bool, signalDBFS, noiseDBFS float64, modulation, protocol, name, transcript string) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	now := time.Now()
	m.mu.Lock()
	if jobID == "" {
		m.progress.ChecksCompleted++
		m.progress.LastCheckAt = &now
		if active {
			m.progress.LastActivityAt = &now
		}
	} else if job, ok := m.jobs[jobID]; ok {
		job.Progress.ChecksCompleted++
		job.Progress.LastCheckAt = &now
		if active {
			job.Progress.LastActivityAt = &now
		}
		m.jobs[jobID] = job
	}
	record, exists := m.records[key]
	if !exists {
		record = MapperFrequencyRecord{FrequencyHz: frequencyHz, FirstCheckedAt: now, StrongestDBFS: -200}
	}
	if record.FirstCheckedAt.IsZero() {
		record.FirstCheckedAt = now
	}
	record.LastCheckedAt = now
	record.Checks++
	identifyObservation := strings.EqualFold(config.Mode, "decipher")
	if identifyObservation {
		record.IdentifyChecks++
	} else {
		record.DiscoveryChecks++
	}
	if active {
		if record.FirstSeen.IsZero() {
			record.FirstSeen = now
		}
		record.Hits++
		if identifyObservation {
			record.IdentifyHits++
		} else {
			record.DiscoveryHits++
		}
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
		record.JobIDs = mergeUniqueStrings(record.JobIDs, []string{jobID})
		record.DeviceIDs = mergeUniqueStrings(record.DeviceIDs, []string{deviceID})
		record.Confidence = math.Max(record.Confidence, .72)
		if location := observationLocation(config); location != nil {
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
	m.SetIdentificationEvidence(frequencyHz, source, confidence, false, "", nil)
}

func (m *MapperManager) SetIdentificationEvidence(frequencyHz float64, source string, confidence float64, verified bool, reason string, distanceMiles *float64) {
	key := fmt.Sprintf("%.0f", frequencyHz)
	m.mu.Lock()
	record, exists := m.records[key]
	if exists {
		record.IdentificationSource = strings.TrimSpace(source)
		record.Confidence = math.Max(record.Confidence, confidence)
		if verified {
			record.IdentificationVerified = true
			record.VerificationReason = strings.TrimSpace(reason)
			if distanceMiles != nil {
				distance := *distanceMiles
				record.ReferenceDistanceMiles = &distance
			}
		}
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
	messages = validDecoderMessages(messages)
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
		if strings.TrimSpace(record.Name) == "" {
			record.Name = firstNonEmpty(messages[0].Protocol, canonicalDecoderID(decoderID))
		}
		record.Confidence = math.Max(record.Confidence, messages[0].Confidence)
		evidence := make([]string, 0, len(messages))
		for _, message := range messages {
			evidence = append(evidence, message.Summary)
			record.Callsigns = mergeUniqueStrings(record.Callsigns, message.Callsigns)
		}
		record.DetectionEvidence = strings.Join(evidence, " · ")
		record.IdentificationVerified = true
		record.VerificationReason = "Valid " + firstNonEmpty(messages[0].Protocol, canonicalDecoderID(decoderID)) + " decoder output"
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
	m.upload(true)
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
	header := []string{"frequency_hz", "frequency_mhz", "name", "modulation", "protocol", "candidate_decoder", "detection_status", "detection_evidence", "decoder_ready", "analysis_engine", "analysis_summary", "analysis_evidence", "callsigns", "first_seen", "last_seen", "checks", "hits", "occupancy", "strongest_dbfs", "noise_dbfs", "confidence", "transcript", "latitude", "longitude", "location_name", "location_precision", "identification_source", "peak_activity_hours", "activity_time_zone", "mapper_job_ids", "receiver_ids", "discovery_checks", "discovery_hits", "identify_checks", "identify_hits", "fully_identified", "verification_reason", "reference_distance_miles"}
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
			safeSpreadsheetText(strings.Join(record.JobIDs, " | ")), safeSpreadsheetText(strings.Join(record.DeviceIDs, " | ")),
			strconv.Itoa(record.DiscoveryChecks), strconv.Itoa(record.DiscoveryHits), strconv.Itoa(record.IdentifyChecks), strconv.Itoa(record.IdentifyHits),
			strconv.FormatBool(mapperRecordFullyIdentified(record)), safeSpreadsheetText(record.VerificationReason), optionalFloat(record.ReferenceDistanceMiles, 2),
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

func optionalFloat(value *float64, precision int) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', precision, 64)
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
			m.upload(false)
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

func (m *MapperManager) persistJobs() {
	m.mu.RLock()
	jobs := make(map[string]MapperJob, len(m.jobs))
	for id, job := range m.jobs {
		copy := job
		copy.Config.Secret = ""
		jobs[id] = copy
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	path := m.jobsPath
	m.mu.RUnlock()
	if err == nil && path != "" {
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		_ = os.WriteFile(path, data, 0o600)
	}
}

func (m *MapperManager) upload(manual bool) {
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
		if config.UploadVerifiedOnly && !mapperRecordFullyIdentified(signal) {
			continue
		}
		rows = append(rows, signal)
	}
	m.mu.RUnlock()
	if len(rows) == 0 {
		if manual && config.UploadVerifiedOnly {
			m.mu.Lock()
			m.lastError = "no new fully identified frequencies are ready to send"
			m.mu.Unlock()
		}
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
	if config.UploadVerifiedOnly && !mapperRecordFullyIdentified(record) {
		m.mu.Lock()
		m.lastError = "this frequency is not fully identified; turn off Identified only or collect authoritative evidence"
		m.mu.Unlock()
		return m.Status()
	}
	rows := []MapperFrequencyRecord{record}
	err := m.postRows(config, rows)
	m.finishUpload(rows, err)
	return m.Status()
}

func mapperRecordFullyIdentified(record MapperFrequencyRecord) bool {
	if !record.IdentificationVerified || strings.TrimSpace(record.Name) == "" {
		return false
	}
	classification := strings.ToLower(strings.TrimSpace(firstNonEmpty(record.ProtocolName, record.Modulation)))
	return classification != "" && classification != "unknown" && classification != "auto" && !strings.Contains(classification, "candidate") && !strings.Contains(classification, "likely")
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
	if mapperRecordFullyIdentified(record) {
		confidence = "Fully identified"
	}
	notes := fmt.Sprintf("GP-SDR Identify: signal %.1f dBFS; local noise %.1f dBFS; confidence %.0f%%", record.StrongestDBFS, record.NoiseDBFS, record.Confidence*100)
	if record.VerificationReason != "" {
		notes += "; verified: " + record.VerificationReason
	}
	if record.ReferenceDistanceMiles != nil {
		notes += fmt.Sprintf("; reference distance %.1f mi", *record.ReferenceDistanceMiles)
	}
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
