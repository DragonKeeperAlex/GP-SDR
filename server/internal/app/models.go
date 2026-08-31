package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

var Version = "1.4.1"

type SDRDevice struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Kind               string             `json:"kind"`
	Serial             *string            `json:"serial"`
	Driver             string             `json:"driver"`
	Connected          bool               `json:"connected"`
	Available          bool               `json:"available"`
	HealthWarning      string             `json:"healthWarning,omitempty"`
	TunerID            string             `json:"tunerID,omitempty"`
	SampleRateLimit    *float64           `json:"sampleRateLimit"`
	FrequencyMinimumHz float64            `json:"frequencyMinimumHz,omitempty"`
	FrequencyMaximumHz float64            `json:"frequencyMaximumHz,omitempty"`
	FrequencyRangeNote string             `json:"frequencyRangeNote,omitempty"`
	HelperArchitecture *string            `json:"helperArchitecture"`
	Note               *string            `json:"note"`
	Calibration        *DeviceCalibration `json:"calibration,omitempty"`
	Host               string             `json:"host,omitempty"`
	Port               int                `json:"port,omitempty"`
}

type RemoteReceiver struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

type DeviceCalibration struct {
	DeviceID        string    `json:"deviceID"`
	DeviceKind      string    `json:"deviceKind"`
	Serial          string    `json:"serial,omitempty"`
	ReferenceHz     float64   `json:"referenceHz"`
	PPMCorrection   int       `json:"ppmCorrection"`
	IQGain          float64   `json:"iqGain"`
	IQPhase         float64   `json:"iqPhase"`
	IQSwap          bool      `json:"iqSwap"`
	DCRemoval       bool      `json:"dcRemoval"`
	LNAGainDB       int       `json:"lnaGainDB"`
	VGAGainDB       int       `json:"vgaGainDB"`
	AmpEnabled      bool      `json:"ampEnabled"`
	Confidence      float64   `json:"confidence"`
	SignalToNoiseDB float64   `json:"signalToNoiseDB"`
	MeasuredAt      time.Time `json:"measuredAt"`
	Source          string    `json:"source"`
}

type CalibrationRequest struct {
	DeviceID     string  `json:"deviceID"`
	ReferenceHz  float64 `json:"referenceHz"`
	SampleRateHz int     `json:"sampleRateHz"`
	LNAGainDB    int     `json:"lnaGainDB"`
	VGAGainDB    int     `json:"vgaGainDB"`
}

type ScanRange struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	StartHz           float64 `json:"startHz"`
	EndHz             float64 `json:"endHz"`
	StepHz            float64 `json:"stepHz"`
	DwellMilliseconds int     `json:"dwellMilliseconds"`
	PreferredMode     string  `json:"preferredMode"`
	Decoder           *string `json:"decoder,omitempty"`
	Enabled           bool    `json:"enabled"`
}

type ChannelDefinition struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	FrequencyHz float64 `json:"frequencyHz"`
	BandwidthHz float64 `json:"bandwidthHz"`
	Mode        string  `json:"mode"`
	Decoder     *string `json:"decoder"`
	Enabled     bool    `json:"enabled"`
	Priority    int     `json:"priority"`
}

type DeviceAssignment struct {
	ID       string  `json:"id"`
	DeviceID *string `json:"deviceID"`
	Role     string  `json:"role"`
	Target   *string `json:"target"`
}

type TalkgroupDefinition struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Mode      string `json:"mode"`
	Encrypted bool   `json:"encrypted"`
	Enabled   bool   `json:"enabled"`
}

type P25SystemConfig struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	ControlChannelsHz []float64             `json:"controlChannelsHz"`
	NAC               string                `json:"nac"`
	WACN              string                `json:"wacn"`
	SystemID          string                `json:"systemID"`
	TDMAControl       bool                  `json:"tdmaControl"`
	Talkgroups        []TalkgroupDefinition `json:"talkgroups"`
	Enabled           bool                  `json:"enabled"`
}

type SurveySettings struct {
	NoiseMarginDB      float64 `json:"noiseMarginDB"`
	RevisitSeconds     int     `json:"revisitSeconds"`
	RecordAudio        bool    `json:"recordAudio"`
	RecordIQForUnknown bool    `json:"recordIQForUnknown"`
	TranscribeVoice    bool    `json:"transcribeVoice"`
	MaxRecordingDays   int     `json:"maxRecordingDays"`
	P25SampleRateHz    int     `json:"p25SampleRateHz,omitempty"`
	P25AmpMode         string  `json:"p25AmpMode,omitempty"`
	P25LNAGainDB       *int    `json:"p25LNAGainDB,omitempty"`
	P25VGAGainDB       *int    `json:"p25VGAGainDB,omitempty"`
}

type ScanProfile struct {
	SchemaVersion     int                   `json:"schemaVersion"`
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Summary           string                `json:"summary"`
	ReferenceArea     *ProfileReferenceArea `json:"referenceArea,omitempty"`
	Ranges            []ScanRange           `json:"ranges"`
	Channels          []ChannelDefinition   `json:"channels"`
	DeviceAssignments []DeviceAssignment    `json:"deviceAssignments"`
	P25Systems        []P25SystemConfig     `json:"p25Systems,omitempty"`
	Settings          SurveySettings        `json:"settings"`
	BuiltIn           bool                  `json:"builtIn"`
}

// ProfileReferenceArea preserves the geographic scope used when reference
// data was imported. Mapper requires this evidence before treating a
// RadioReference frequency match as verified, preventing an identical
// frequency in another state from being accepted merely by number.
type ProfileReferenceArea struct {
	Provider    string    `json:"provider"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusMiles float64   `json:"radiusMiles"`
	Label       string    `json:"label,omitempty"`
	ImportedAt  time.Time `json:"importedAt,omitempty"`
}

type TransmissionEvent struct {
	ID              string               `json:"id"`
	StartedAt       time.Time            `json:"startedAt"`
	DurationSeconds float64              `json:"durationSeconds"`
	FrequencyHz     float64              `json:"frequencyHz"`
	BandwidthHz     float64              `json:"bandwidthHz"`
	SignalDBFS      float64              `json:"signalDBFS"`
	NoiseDBFS       float64              `json:"noiseDBFS"`
	Modulation      string               `json:"modulation"`
	ProtocolName    *string              `json:"protocolName"`
	Label           *string              `json:"label"`
	DeviceID        string               `json:"deviceID"`
	SystemName      *string              `json:"systemName,omitempty"`
	TalkgroupID     *uint32              `json:"talkgroupID,omitempty"`
	SourceRadioID   *uint32              `json:"sourceRadioID,omitempty"`
	Encrypted       bool                 `json:"encrypted,omitempty"`
	Transcript      *string              `json:"transcript"`
	Callsigns       []string             `json:"callsigns,omitempty"`
	Analysis        *SignalIntelligence  `json:"analysis,omitempty"`
	DecoderMessages []DecoderMessage     `json:"decoderMessages,omitempty"`
	CTCSSHz         *float64             `json:"ctcssHz,omitempty"`
	Confidence      float64              `json:"confidence"`
	AudioPath       *string              `json:"audioPath"`
	IQPath          *string              `json:"iqPath"`
	Simulated       bool                 `json:"simulated"`
	Location        *ObservationLocation `json:"location,omitempty"`
}

// SignalIntelligence records locally-derived evidence. It deliberately keeps
// waveform classification separate from authoritative protocol decoding: a
// likely digital waveform is useful, but it is not proof of P25, DMR, or any
// other specific protocol until a decoder produces frames.
type SignalIntelligence struct {
	Engine               string   `json:"engine"`
	Modulation           string   `json:"modulation"`
	SignalFamily         string   `json:"signalFamily"`
	Confidence           float64  `json:"confidence"`
	EstimatedDeviationHz float64  `json:"estimatedDeviationHz,omitempty"`
	AmplitudeVariation   float64  `json:"amplitudeVariation,omitempty"`
	PhaseClusterScore    float64  `json:"phaseClusterScore,omitempty"`
	Callsigns            []string `json:"callsigns,omitempty"`
	Evidence             []string `json:"evidence,omitempty"`
	Summary              string   `json:"summary"`
}

type ObservationLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Label     string  `json:"label,omitempty"`
	Precision string  `json:"precision,omitempty"`
}

type SignalSummary struct {
	ID            string               `json:"id"`
	FrequencyHz   float64              `json:"frequencyHz"`
	FirstSeen     time.Time            `json:"firstSeen"`
	LastSeen      time.Time            `json:"lastSeen"`
	EventCount    int                  `json:"eventCount"`
	StrongestDBFS float64              `json:"strongestDBFS"`
	Modulation    string               `json:"modulation"`
	ProtocolName  *string              `json:"protocolName"`
	Label         *string              `json:"label"`
	Confidence    float64              `json:"confidence"`
	Location      *ObservationLocation `json:"location,omitempty"`
	CTCSSHz       *float64             `json:"ctcssHz,omitempty"`
}

type DecoderDescriptor struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Standards  []string `json:"standards"`
	State      string   `json:"state"`
	Executable *string  `json:"executable"`
	Note       string   `json:"note"`
}

type MixerChannel struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Channel     ChannelDefinition `json:"channel"`
	SystemName  *string           `json:"systemName,omitempty"`
	TalkgroupID *uint32           `json:"talkgroupID,omitempty"`
	Encrypted   bool              `json:"encrypted,omitempty"`
	Discovered  bool              `json:"discovered,omitempty"`
	EventCount  int               `json:"eventCount,omitempty"`
	LastHeardAt *time.Time        `json:"lastHeardAt,omitempty"`
	Muted       bool              `json:"muted"`
	Solo        bool              `json:"solo"`
	Volume      float64           `json:"volume"`
	Pan         float64           `json:"pan"`
	Active      bool              `json:"active"`
	Level       float64           `json:"level"`
}

type ReceiverPlanItem struct {
	AssignmentID string  `json:"assignmentID"`
	Role         string  `json:"role"`
	Target       *string `json:"target"`
	DeviceID     *string `json:"deviceID"`
	DeviceName   *string `json:"deviceName"`
	State        string  `json:"state"`
	Note         string  `json:"note"`
}

type RuntimeStatus struct {
	Running              bool                `json:"running"`
	Mode                 string              `json:"mode"`
	StartedAt            *time.Time          `json:"startedAt"`
	ActiveProfileID      *string             `json:"activeProfileID"`
	ActiveProfileName    *string             `json:"activeProfileName"`
	ConnectedDeviceCount int                 `json:"connectedDeviceCount"`
	EventCount           int                 `json:"eventCount"`
	WebAddress           string              `json:"webAddress"`
	SimulatorEnabled     bool                `json:"simulatorEnabled"`
	Version              string              `json:"version"`
	LastError            *string             `json:"lastError"`
	DroppedSamples       uint64              `json:"droppedSamples"`
	SignalAnalysis       *SignalIntelligence `json:"signalAnalysis,omitempty"`
	ReceiverTelemetry    *ReceiverTelemetry  `json:"receiverTelemetry,omitempty"`
	Storage              StorageStatus       `json:"storage"`
	HealthNotices        []HealthNotice      `json:"healthNotices"`
}

type StorageStatus struct {
	JournalBytes      int64                `json:"journalBytes"`
	RecordingBytes    int64                `json:"recordingBytes"`
	IQBytes           int64                `json:"iqBytes"`
	IQPendingBytes    int64                `json:"iqPendingBytes"`
	IQRetainedBytes   int64                `json:"iqRetainedBytes"`
	IQQuarantineBytes int64                `json:"iqQuarantineBytes"`
	ProfileBytes      int64                `json:"profileBytes"`
	TotalBytes        int64                `json:"totalBytes"`
	CheckedAt         time.Time            `json:"checkedAt"`
	Policy            StoragePolicy        `json:"policy"`
	LastCleanup       StorageCleanupResult `json:"lastCleanup"`
	CleanupRunning    bool                 `json:"cleanupRunning"`
}

type HealthNotice struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type ReceiverTelemetry struct {
	DeviceID          string  `json:"deviceID"`
	HardwareCenterHz  float64 `json:"hardwareCenterHz"`
	ListenFrequencyHz float64 `json:"listenFrequencyHz"`
	SampleRateHz      int     `json:"sampleRateHz"`
	SignalDBFS        float64 `json:"signalDBFS"`
	NoiseDBFS         float64 `json:"noiseDBFS"`
	PeakDBFS          float64 `json:"peakDBFS"`
	ClippedPercent    float64 `json:"clippedPercent"`
	Overloaded        bool    `json:"overloaded"`
	SignalDetected    bool    `json:"signalDetected"`
	SquelchOpen       bool    `json:"squelchOpen"`
	LNAGainDB         int     `json:"lnaGainDB"`
	VGAGainDB         int     `json:"vgaGainDB"`
	AmpEnabled        bool    `json:"ampEnabled"`
}

type TunerRequest struct {
	DeviceID         string  `json:"deviceID"`
	FrequencyHz      float64 `json:"frequencyHz"`
	Mode             string  `json:"mode"`
	Decoder          string  `json:"decoder,omitempty"`
	BandwidthHz      float64 `json:"bandwidthHz"`
	SampleRateHz     int     `json:"sampleRateHz"`
	GainDB           float64 `json:"gainDB"`
	LNAGainDB        int     `json:"lnaGainDB"`
	VGAGainDB        int     `json:"vgaGainDB"`
	PPMCorrection    int     `json:"ppmCorrection"`
	AmpEnabled       bool    `json:"ampEnabled"`
	AntennaPower     bool    `json:"antennaPower"`
	IQDCRemoval      bool    `json:"iqDCRemoval"`
	IQGain           float64 `json:"iqGain"`
	IQPhase          float64 `json:"iqPhase"`
	IQSwap           bool    `json:"iqSwap"`
	AutoGain         bool    `json:"autoGain"`
	SquelchDB        float64 `json:"squelchDB"`
	MonitorOpen      bool    `json:"monitorOpen"`
	NoiseReduction   string  `json:"noiseReduction"`
	UseCalibration   bool    `json:"useCalibration"`
	LockCenter       bool    `json:"lockCenter"`
	HardwareCenterHz float64 `json:"hardwareCenterHz,omitempty"`
}

type SpectrumSnapshot struct {
	CenterFrequencyHz float64   `json:"centerFrequencyHz"`
	StartFrequencyHz  float64   `json:"startFrequencyHz"`
	EndFrequencyHz    float64   `json:"endFrequencyHz"`
	SampleRateHz      int       `json:"sampleRateHz"`
	BinsDBFS          []float64 `json:"binsDBFS"`
	CapturedAt        time.Time `json:"capturedAt"`
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	text := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", text[:8], text[8:12], text[12:16], text[16:20], text[20:])
}

func ptr[T any](value T) *T { return &value }

func defaultSettings() SurveySettings {
	return SurveySettings{NoiseMarginDB: 8, RevisitSeconds: 20, RecordAudio: true,
		RecordIQForUnknown: true, TranscribeVoice: false, MaxRecordingDays: 30}
}

func builtInProfiles() []ScanProfile {
	discovery := ScanProfile{
		SchemaVersion: 1, ID: "2cc07614-d89a-4a3c-be11-e5f344e478a3", Name: "Local Discovery",
		Summary: "Common nearby VHF and UHF activity", BuiltIn: true, Settings: defaultSettings(),
		Ranges: []ScanRange{
			{ID: NewID(), Name: "VHF Low", StartHz: 136e6, EndHz: 155e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "VHF High", StartHz: 155e6, EndHz: 174e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF 400–420", StartHz: 400e6, EndHz: 420e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF 420–440", StartHz: 420e6, EndHz: 440e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF 440–460", StartHz: 440e6, EndHz: 460e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF 460–480", StartHz: 460e6, EndHz: 480e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF 480–500", StartHz: 480e6, EndHz: 500e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
		},
		Channels:          []ChannelDefinition{},
		DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "discovery"}},
	}
	gmrs := ScanProfile{
		SchemaVersion: 1, ID: "be8e8ba2-ef4d-47f4-875f-f489bc8d894b", Name: "GMRS · Whole Band",
		Summary: "All 462 and 467 MHz GMRS/FRS channels", BuiltIn: true, Settings: defaultSettings(),
		Ranges:            []ScanRange{},
		Channels:          []ChannelDefinition{},
		DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "channelBank", Target: ptr("GMRS")}},
	}
	for i, mhz := range []float64{462.5625, 462.5875, 462.6125, 462.6375, 462.6625, 462.6875, 462.7125} {
		gmrs.Channels = append(gmrs.Channels, channel(fmt.Sprintf("FRS/GMRS %d", i+1), mhz, 12500, "nfm"))
	}
	for i := 0; i < 8; i++ {
		gmrs.Channels = append(gmrs.Channels, channel(fmt.Sprintf("GMRS %d", i+15), 462.55+float64(i)*.025, 20000, "nfm"))
	}
	for i, mhz := range []float64{467.5625, 467.5875, 467.6125, 467.6375, 467.6625, 467.6875, 467.7125} {
		gmrs.Channels = append(gmrs.Channels, channel(fmt.Sprintf("FRS %d", i+8), mhz, 12500, "nfm"))
	}
	for i := 0; i < 8; i++ {
		gmrs.Channels = append(gmrs.Channels, channel(fmt.Sprintf("GMRS RPT IN %d", i+15), 467.55+float64(i)*.025, 20000, "nfm"))
	}
	weather := fixedChannelProfile("61a7f9a2-1173-4c23-b62c-696964f16348", "NOAA Weather", "All seven US weather radio channels", "NOAA")
	for index, mhz := range []float64{162.400, 162.425, 162.450, 162.475, 162.500, 162.525, 162.550} {
		weather.Channels = append(weather.Channels, channel(fmt.Sprintf("WX %d", index+1), mhz, 16_000, "nfm"))
	}
	murs := fixedChannelProfile("87537387-ad2d-4cf2-8763-83590c47955b", "MURS", "All five US Multi-Use Radio Service channels", "MURS")
	for index, mhz := range []float64{151.820, 151.880, 151.940, 154.570, 154.600} {
		murs.Channels = append(murs.Channels, channel(fmt.Sprintf("MURS %d", index+1), mhz, 12_500, "nfm"))
	}
	cb := rangeProfile("b4a86f73-bce9-4605-94a8-a44d7f19d379", "CB · 40 Channels", "US citizens-band voice channels", "CB",
		ScanRange{ID: NewID(), Name: "CB", StartHz: 26.965e6, EndHz: 27.405e6, StepHz: 10_000, DwellMilliseconds: 180, PreferredMode: "am", Enabled: true})
	fm := fixedChannelProfile("0836ac3e-d346-4d63-8fd2-17dddf3b5b68", "US FM Broadcast · 100 Channels", "Every standard US FM channel from 88.1 through 107.9 MHz", "Broadcast FM")
	for index := 0; index < 100; index++ {
		mhz := 88.1 + float64(index)*0.2
		fm.Channels = append(fm.Channels, channel(fmt.Sprintf("FM %.1f", mhz), mhz, 180_000, "wfm"))
	}
	am := rangeProfile("72d352ba-ed66-420a-9843-0130afca6469", "Broadcast AM", "530–1710 kHz medium-wave AM", "Broadcast AM",
		ScanRange{ID: NewID(), Name: "AM broadcast", StartHz: 530e3, EndHz: 1710e3, StepHz: 10_000, DwellMilliseconds: 220, PreferredMode: "am", Enabled: true})
	airband := rangeProfile("766bb943-a674-43b3-a446-1bc32706b672", "Civil Airband", "118–136.975 MHz AM voice", "Airband",
		ScanRange{ID: NewID(), Name: "Civil airband", StartHz: 118e6, EndHz: 136.975e6, StepHz: 25_000, DwellMilliseconds: 160, PreferredMode: "am", Enabled: true})
	marine := rangeProfile("c97e9493-c155-45c3-8cdb-f1950d8cfcc9", "Marine VHF", "US marine voice band", "Marine VHF",
		ScanRange{ID: NewID(), Name: "Marine VHF", StartHz: 156.025e6, EndHz: 162.025e6, StepHz: 25_000, DwellMilliseconds: 160, PreferredMode: "nfm", Enabled: true})
	ham := rangeProfile("c0fe39a5-3c5c-4a34-8488-d20e85c6c9ec", "Amateur VHF/UHF", "2 m and 70 cm amateur bands", "Amateur",
		ScanRange{ID: NewID(), Name: "2 m", StartHz: 144e6, EndHz: 148e6, StepHz: 12_500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
		ScanRange{ID: NewID(), Name: "70 cm Low", StartHz: 420e6, EndHz: 435e6, StepHz: 12_500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
		ScanRange{ID: NewID(), Name: "70 cm High", StartHz: 435e6, EndHz: 450e6, StepHz: 12_500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true})
	publicSafety := rangeProfile("5fdcfc10-2b76-43aa-b8e6-4b7c7831f3a4", "Public Safety Discovery", "Common 700 and 800 MHz receive segments", "Public safety",
		ScanRange{ID: NewID(), Name: "700 MHz", StartHz: 769e6, EndHz: 775e6, StepHz: 12_500, DwellMilliseconds: 180, PreferredMode: "digital", Enabled: true},
		ScanRange{ID: NewID(), Name: "800 MHz", StartHz: 851e6, EndHz: 869e6, StepHz: 12_500, DwellMilliseconds: 180, PreferredMode: "digital", Enabled: true})
	profiles := []ScanProfile{discovery, gmrs, weather, murs, cb, fm, am, airband, marine, ham, publicSafety}
	profiles = append(profiles, decoderScanProfiles()...)
	profiles = append(profiles, handheldProfiles()...)
	profiles = append(profiles, regionalConventionalProfiles()...)
	profiles = append(profiles, regionalP25Profiles()...)
	return profiles
}

func fixedChannelProfile(id, name, summary, target string) ScanProfile {
	return ScanProfile{SchemaVersion: 1, ID: id, Name: name, Summary: summary, BuiltIn: true, Settings: defaultSettings(),
		Ranges: []ScanRange{}, Channels: []ChannelDefinition{}, DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "channelBank", Target: &target}}}
}

func rangeProfile(id, name, summary, target string, ranges ...ScanRange) ScanProfile {
	return ScanProfile{SchemaVersion: 1, ID: id, Name: name, Summary: summary, BuiltIn: true, Settings: defaultSettings(), Ranges: ranges,
		Channels: []ChannelDefinition{}, DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "discovery", Target: &target}}}
}

func normalizeProfile(profile ScanProfile) ScanProfile {
	if profile.Ranges == nil {
		profile.Ranges = []ScanRange{}
	}
	if profile.Channels == nil {
		profile.Channels = []ChannelDefinition{}
	}
	if profile.DeviceAssignments == nil {
		profile.DeviceAssignments = []DeviceAssignment{}
	}
	if profile.P25Systems == nil {
		profile.P25Systems = []P25SystemConfig{}
	}
	return profile
}

func channel(name string, mhz, bandwidth float64, mode string) ChannelDefinition {
	return ChannelDefinition{ID: NewID(), Name: name, FrequencyHz: mhz * 1e6, BandwidthHz: bandwidth,
		Mode: mode, Enabled: true, Priority: 5}
}
