package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

var Version = "1.0.1-dev"

type SDRDevice struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Serial             *string  `json:"serial"`
	Driver             string   `json:"driver"`
	Connected          bool     `json:"connected"`
	Available          bool     `json:"available"`
	SampleRateLimit    *float64 `json:"sampleRateLimit"`
	HelperArchitecture *string  `json:"helperArchitecture"`
	Note               *string  `json:"note"`
}

type ScanRange struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	StartHz           float64 `json:"startHz"`
	EndHz             float64 `json:"endHz"`
	StepHz            float64 `json:"stepHz"`
	DwellMilliseconds int     `json:"dwellMilliseconds"`
	PreferredMode     string  `json:"preferredMode"`
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
}

type ScanProfile struct {
	SchemaVersion     int                 `json:"schemaVersion"`
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Summary           string              `json:"summary"`
	Ranges            []ScanRange         `json:"ranges"`
	Channels          []ChannelDefinition `json:"channels"`
	DeviceAssignments []DeviceAssignment  `json:"deviceAssignments"`
	P25Systems        []P25SystemConfig   `json:"p25Systems,omitempty"`
	Settings          SurveySettings      `json:"settings"`
	BuiltIn           bool                `json:"builtIn"`
}

type TransmissionEvent struct {
	ID              string    `json:"id"`
	StartedAt       time.Time `json:"startedAt"`
	DurationSeconds float64   `json:"durationSeconds"`
	FrequencyHz     float64   `json:"frequencyHz"`
	BandwidthHz     float64   `json:"bandwidthHz"`
	SignalDBFS      float64   `json:"signalDBFS"`
	NoiseDBFS       float64   `json:"noiseDBFS"`
	Modulation      string    `json:"modulation"`
	ProtocolName    *string   `json:"protocolName"`
	Label           *string   `json:"label"`
	DeviceID        string    `json:"deviceID"`
	Transcript      *string   `json:"transcript"`
	Confidence      float64   `json:"confidence"`
	AudioPath       *string   `json:"audioPath"`
	IQPath          *string   `json:"iqPath"`
	Simulated       bool      `json:"simulated"`
}

type SignalSummary struct {
	ID            string    `json:"id"`
	FrequencyHz   float64   `json:"frequencyHz"`
	FirstSeen     time.Time `json:"firstSeen"`
	LastSeen      time.Time `json:"lastSeen"`
	EventCount    int       `json:"eventCount"`
	StrongestDBFS float64   `json:"strongestDBFS"`
	Modulation    string    `json:"modulation"`
	ProtocolName  *string   `json:"protocolName"`
	Label         *string   `json:"label"`
	Confidence    float64   `json:"confidence"`
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
	Running              bool       `json:"running"`
	Mode                 string     `json:"mode"`
	StartedAt            *time.Time `json:"startedAt"`
	ActiveProfileID      *string    `json:"activeProfileID"`
	ActiveProfileName    *string    `json:"activeProfileName"`
	ConnectedDeviceCount int        `json:"connectedDeviceCount"`
	EventCount           int        `json:"eventCount"`
	WebAddress           string     `json:"webAddress"`
	SimulatorEnabled     bool       `json:"simulatorEnabled"`
	Version              string     `json:"version"`
	LastError            *string    `json:"lastError"`
	DroppedSamples       uint64     `json:"droppedSamples"`
}

type TunerRequest struct {
	DeviceID     string  `json:"deviceID"`
	FrequencyHz  float64 `json:"frequencyHz"`
	Mode         string  `json:"mode"`
	BandwidthHz  float64 `json:"bandwidthHz"`
	SampleRateHz int     `json:"sampleRateHz"`
	GainDB       float64 `json:"gainDB"`
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
			{ID: NewID(), Name: "VHF", StartHz: 136e6, EndHz: 174e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
			{ID: NewID(), Name: "UHF", StartHz: 400e6, EndHz: 500e6, StepHz: 12500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
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
	fm := rangeProfile("0836ac3e-d346-4d63-8fd2-17dddf3b5b68", "Broadcast FM", "88–108 MHz wideband FM", "Broadcast FM",
		ScanRange{ID: NewID(), Name: "FM broadcast", StartHz: 88e6, EndHz: 108e6, StepHz: 200_000, DwellMilliseconds: 220, PreferredMode: "wfm", Enabled: true})
	am := rangeProfile("72d352ba-ed66-420a-9843-0130afca6469", "Broadcast AM", "530–1710 kHz medium-wave AM", "Broadcast AM",
		ScanRange{ID: NewID(), Name: "AM broadcast", StartHz: 530e3, EndHz: 1710e3, StepHz: 10_000, DwellMilliseconds: 220, PreferredMode: "am", Enabled: true})
	airband := rangeProfile("766bb943-a674-43b3-a446-1bc32706b672", "Civil Airband", "118–136.975 MHz AM voice", "Airband",
		ScanRange{ID: NewID(), Name: "Civil airband", StartHz: 118e6, EndHz: 136.975e6, StepHz: 25_000, DwellMilliseconds: 160, PreferredMode: "am", Enabled: true})
	marine := rangeProfile("c97e9493-c155-45c3-8cdb-f1950d8cfcc9", "Marine VHF", "US marine voice band", "Marine VHF",
		ScanRange{ID: NewID(), Name: "Marine VHF", StartHz: 156.025e6, EndHz: 162.025e6, StepHz: 25_000, DwellMilliseconds: 160, PreferredMode: "nfm", Enabled: true})
	ham := rangeProfile("c0fe39a5-3c5c-4a34-8488-d20e85c6c9ec", "Amateur VHF/UHF", "2 m and 70 cm amateur bands", "Amateur",
		ScanRange{ID: NewID(), Name: "2 m", StartHz: 144e6, EndHz: 148e6, StepHz: 12_500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true},
		ScanRange{ID: NewID(), Name: "70 cm", StartHz: 420e6, EndHz: 450e6, StepHz: 12_500, DwellMilliseconds: 160, PreferredMode: "auto", Enabled: true})
	publicSafety := rangeProfile("5fdcfc10-2b76-43aa-b8e6-4b7c7831f3a4", "Public Safety Discovery", "Common 700 and 800 MHz receive segments", "Public safety",
		ScanRange{ID: NewID(), Name: "700 MHz", StartHz: 769e6, EndHz: 775e6, StepHz: 12_500, DwellMilliseconds: 180, PreferredMode: "digital", Enabled: true},
		ScanRange{ID: NewID(), Name: "800 MHz", StartHz: 851e6, EndHz: 869e6, StepHz: 12_500, DwellMilliseconds: 180, PreferredMode: "digital", Enabled: true})
	profiles := []ScanProfile{discovery, gmrs, weather, murs, cb, fm, am, airband, marine, ham, publicSafety}
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
