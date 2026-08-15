package app

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runtime struct {
	mu             sync.RWMutex
	Profiles       *ProfileStore
	Events         *EventStore
	devices        []SDRDevice
	decoders       []DecoderDescriptor
	mixer          []MixerChannel
	plan           []ReceiverPlanItem
	running        bool
	startedAt      *time.Time
	active         *ScanProfile
	stop           chan struct{}
	demo           bool
	webAddress     string
	dataDirectory  string
	transcriber    *Transcriber
	lastError      *string
	droppedSamples uint64
	op25           *OP25Manager
	radioReference *radioReferenceClient
	audioHub       *AudioHub
	installer      *Installer
	spectrum       SpectrumSnapshot
	tuning         bool
}

func NewRuntime(dataDirectory, webAddress string, demo bool) (*Runtime, error) {
	profiles, err := NewProfileStore(filepath.Join(dataDirectory, "Profiles"))
	if err != nil {
		return nil, err
	}
	events, err := NewEventStore(filepath.Join(dataDirectory, "Data"))
	if err != nil {
		return nil, err
	}
	runtimeState := &Runtime{Profiles: profiles, Events: events, devices: DiscoverDevices(demo), decoders: DiscoverDecoders(),
		demo: demo, webAddress: webAddress, dataDirectory: dataDirectory, transcriber: NewTranscriber(), op25: &OP25Manager{},
		radioReference: newRadioReferenceClient(), audioHub: NewAudioHub()}
	runtimeState.installer = NewInstaller(runtimeState.Refresh)
	return runtimeState, nil
}

func (r *Runtime) Refresh() {
	devices := DiscoverDevices(r.demo)
	decoders := DiscoverDecoders()
	transcriber := NewTranscriber()
	radioReference := newRadioReferenceClient()
	r.mu.Lock()
	r.devices = devices
	r.decoders = decoders
	r.transcriber = transcriber
	r.radioReference = radioReference
	r.mu.Unlock()
}
func (r *Runtime) Devices() []SDRDevice {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]SDRDevice(nil), r.devices...)
}
func (r *Runtime) Decoders() []DecoderDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]DecoderDescriptor(nil), r.decoders...)
}
func (r *Runtime) Setup() SetupOverview {
	if r.installer == nil {
		return SetupOverview{Platform: "unknown", PackageManager: "manual"}
	}
	return r.installer.Overview()
}
func (r *Runtime) Install(componentID string) (InstallJob, error) {
	if r.installer == nil {
		return InstallJob{}, errors.New("component installer is unavailable")
	}
	return r.installer.Start(componentID)
}
func (r *Runtime) Mixer() []MixerChannel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]MixerChannel{}, r.mixer...)
}
func (r *Runtime) Plan() []ReceiverPlanItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]ReceiverPlanItem{}, r.plan...)
}

func (r *Runtime) Status() RuntimeStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := RuntimeStatus{Running: r.running, Mode: "Idle", StartedAt: r.startedAt, ConnectedDeviceCount: 0,
		EventCount: r.Events.Count(), WebAddress: r.webAddress, SimulatorEnabled: r.demo, Version: Version,
		LastError: r.lastError, DroppedSamples: r.droppedSamples}
	for _, d := range r.devices {
		if d.Connected && d.Kind != "Simulator" {
			status.ConnectedDeviceCount++
		}
	}
	if r.running {
		status.Mode = "Survey"
		if r.demo {
			status.Mode = "Survey · Demo source"
		}
		if r.active != nil {
			status.ActiveProfileID = &r.active.ID
			status.ActiveProfileName = &r.active.Name
			if len(enabledP25Systems(*r.active)) > 0 && !r.demo {
				status.Mode = "P25 trunk follow"
			} else if r.tuning {
				status.Mode = "Tuner · " + strings.ToUpper(firstChannelMode(*r.active))
			}
		}
	}
	return status
}

func (r *Runtime) Start(profileID string) error {
	profile, ok := r.Profiles.Get(profileID)
	if !ok {
		return ErrNotFound
	}
	return r.startProfile(profile, nil)
}

func firstChannelMode(profile ScanProfile) string {
	if len(profile.Channels) == 0 || strings.TrimSpace(profile.Channels[0].Mode) == "" {
		return "NFM"
	}
	return profile.Channels[0].Mode
}

func (r *Runtime) Tune(request TunerRequest) error {
	request.Mode = strings.ToLower(strings.TrimSpace(request.Mode))
	if request.Mode == "fm" {
		request.Mode = "wfm"
	}
	if request.Mode != "am" && request.Mode != "nfm" && request.Mode != "wfm" {
		return errors.New("Tuner mode must be AM, NFM, or WFM.")
	}
	if !isFinitePositive(request.FrequencyHz) {
		return errors.New("Enter a valid positive tuner frequency.")
	}
	if request.BandwidthHz <= 0 {
		if request.Mode == "wfm" {
			request.BandwidthHz = 180_000
		} else {
			request.BandwidthHz = 12_500
		}
	}
	if request.GainDB < 0 || request.GainDB > 62 {
		return errors.New("Tuner gain must be between 0 and 62 dB.")
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	assignment := DeviceAssignment{ID: NewID(), Role: "tuner", Target: ptr("Quick Tune")}
	if deviceID != "" {
		assignment.DeviceID = &deviceID
	}
	profile := ScanProfile{SchemaVersion: 1, ID: "quick-tune", Name: "Quick Tune", Summary: "Direct receiver tuning",
		Channels: []ChannelDefinition{{ID: "quick-tune-channel", Name: "Tuned audio", FrequencyHz: request.FrequencyHz,
			BandwidthHz: request.BandwidthHz, Mode: request.Mode, Enabled: true, Priority: 10}},
		Ranges: []ScanRange{}, DeviceAssignments: []DeviceAssignment{assignment}, P25Systems: []P25SystemConfig{},
		Settings: SurveySettings{NoiseMarginDB: 6}}
	return r.startProfile(profile, &request)
}

func isFinitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (r *Runtime) Spectrum() SpectrumSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot := r.spectrum
	snapshot.BinsDBFS = append([]float64(nil), snapshot.BinsDBFS...)
	return snapshot
}

func (r *Runtime) startProfile(profile ScanProfile, tuner *TunerRequest) error {
	r.Stop()
	plan := r.buildPlan(profile)
	var liveDevice *SDRDevice
	useP25 := len(enabledP25Systems(profile)) > 0
	useWideband := false
	if !r.demo {
		for _, item := range plan {
			if item.DeviceID == nil || item.State != "assigned" {
				continue
			}
			for index := range r.devices {
				if r.devices[index].ID == *item.DeviceID && r.devices[index].Connected {
					device := r.devices[index]
					liveDevice = &device
					break
				}
			}
			if liveDevice != nil {
				break
			}
		}
		if liveDevice == nil {
			return errors.New("No SDR is connected. Connect a receiver and press Refresh, or launch with --demo.")
		}
		if !useP25 && len(surveyTargets(profile)) == 0 {
			return errors.New("This profile has no AM, NFM, WFM, or automatic scan targets.")
		}
		if !useP25 && hasReceiverRole(profile, "channelBank") {
			_, _, useWideband = widebandSpec(profile, *liveDevice)
		}
		if useP25 {
			if err := r.op25.Start(profile, plan, r.devices, r.dataDirectory); err != nil {
				return err
			}
		}
	}
	now := time.Now()
	mixer := make([]MixerChannel, 0, len(profile.Channels)+64)
	for _, ch := range profile.Channels {
		mixer = append(mixer, MixerChannel{ID: ch.ID, Kind: "channel", Channel: ch, Volume: .8})
	}
	if useP25 {
		mixer = append(mixer, p25ProfileMixer(profile)...)
	}
	r.mu.Lock()
	r.running = true
	r.startedAt = &now
	r.active = &profile
	r.mixer = mixer
	r.plan = plan
	r.stop = make(chan struct{})
	r.lastError = nil
	r.tuning = tuner != nil
	stop := r.stop
	demo := r.demo
	r.mu.Unlock()
	if demo {
		go r.simulationLoop(stop)
	} else if useP25 {
		go r.p25MonitorLoop(stop, profile)
	} else if tuner != nil {
		go r.tunerLoop(stop, profile, *liveDevice, *tuner)
	} else if !useP25 && useWideband {
		go r.widebandBankLoop(stop, profile, *liveDevice)
	} else if !useP25 {
		go r.liveSurveyLoop(stop, profile, *liveDevice)
	}
	return nil
}

func (r *Runtime) setRuntimeError(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	r.mu.Lock()
	r.lastError = &message
	r.mu.Unlock()
}

func (r *Runtime) clearRuntimeError() {
	r.mu.Lock()
	r.lastError = nil
	r.mu.Unlock()
}

func (r *Runtime) updateMixerActivity(frequencyHz, level float64, active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.mixer {
		match := math.Abs(r.mixer[index].Channel.FrequencyHz-frequencyHz) < 1
		if match {
			r.mixer[index].Active = active
			r.mixer[index].Level = level
		}
	}
}

func hasReceiverRole(profile ScanProfile, role string) bool {
	for _, assignment := range profile.DeviceAssignments {
		if assignment.Role == role {
			return true
		}
	}
	return false
}
func (r *Runtime) Stop() {
	r.mu.Lock()
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
	r.running = false
	r.tuning = false
	r.startedAt = nil
	for i := range r.mixer {
		r.mixer[i].Active = false
		r.mixer[i].Level = 0
	}
	r.mu.Unlock()
	if r.op25 != nil {
		r.op25.Stop()
	}
}

func (r *Runtime) P25Status() P25Status {
	if r.op25 == nil {
		return P25Status{State: "setup", Note: "OP25 manager is unavailable."}
	}
	return r.op25.Status()
}

func (r *Runtime) TranscriptionStatus() TranscriptionStatus {
	r.mu.RLock()
	transcriber := r.transcriber
	r.mu.RUnlock()
	if transcriber == nil {
		return TranscriptionStatus{State: "setup", Note: "Offline transcription is unavailable."}
	}
	return transcriber.Status()
}

func (r *Runtime) buildPlan(profile ScanProfile) []ReceiverPlanItem {
	connected := make([]SDRDevice, 0)
	for _, d := range r.devices {
		if d.Connected {
			connected = append(connected, d)
		}
	}
	used := make(map[string]bool)
	items := make([]ReceiverPlanItem, 0, len(profile.DeviceAssignments))
	for _, assignment := range profile.DeviceAssignments {
		item := ReceiverPlanItem{AssignmentID: assignment.ID, Role: assignment.Role, Target: assignment.Target, DeviceID: assignment.DeviceID, State: "waiting", Note: "No compatible receiver is connected."}
		if assignment.DeviceID != nil {
			for _, d := range connected {
				if d.ID == *assignment.DeviceID {
					name := d.Name
					item.DeviceName = &name
					item.State = "assigned"
					item.Note = "Pinned by profile."
					used[d.ID] = true
					break
				}
			}
		} else {
			for _, d := range connected {
				if !used[d.ID] {
					id, name := d.ID, d.Name
					item.DeviceID = &id
					item.DeviceName = &name
					item.State = "assigned"
					item.Note = "Assigned automatically."
					used[d.ID] = true
					break
				}
			}
		}
		items = append(items, item)
	}
	return items
}

func (r *Runtime) UpdateMixer(id string, muted, solo *bool, volume, pan *float64) (MixerChannel, error) {
	r.mu.Lock()
	for i := range r.mixer {
		if r.mixer[i].ID == id {
			if muted != nil {
				r.mixer[i].Muted = *muted
			}
			if solo != nil {
				r.mixer[i].Solo = *solo
			}
			if volume != nil {
				r.mixer[i].Volume = clamp(*volume, 0, 1)
			}
			if pan != nil {
				r.mixer[i].Pan = clamp(*pan, -1, 1)
			}
			updated := r.mixer[i]
			r.mu.Unlock()
			if updated.TalkgroupID != nil && muted != nil && r.op25 != nil {
				if err := r.op25.UpdateTalkgroup(*updated.TalkgroupID, map[string]any{"mute": *muted}); err != nil {
					return updated, err
				}
			}
			return updated, nil
		}
	}
	r.mu.Unlock()
	return MixerChannel{}, ErrNotFound
}

func p25MixerID(talkgroupID uint32) string {
	return fmt.Sprintf("p25:%d", talkgroupID)
}

func p25ProfileMixer(profile ScanProfile) []MixerChannel {
	items := make([]MixerChannel, 0)
	seen := make(map[uint32]bool)
	for _, system := range enabledP25Systems(profile) {
		for _, talkgroup := range system.Talkgroups {
			id := uint32(talkgroup.ID)
			if id == 0 || seen[id] {
				continue
			}
			seen[id] = true
			name := talkgroup.Name
			if strings.TrimSpace(name) == "" {
				name = fmt.Sprintf("Talkgroup %d", id)
			}
			channel := ChannelDefinition{ID: p25MixerID(id), Name: name, Mode: "p25", Enabled: talkgroup.Enabled, Priority: 5, BandwidthHz: 12_500}
			systemName, talkgroupID := system.Name, id
			items = append(items, MixerChannel{ID: channel.ID, Kind: "talkgroup", Channel: channel, SystemName: &systemName,
				TalkgroupID: &talkgroupID, Encrypted: talkgroup.Encrypted, Muted: talkgroup.Encrypted || !talkgroup.Enabled, Volume: .8})
		}
	}
	return items
}

func (r *Runtime) p25MonitorLoop(stop <-chan struct{}, profile ScanProfile) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	previous := make(map[string]P25ActiveCall)
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			talkgroups, talkgroupErr := r.op25.Talkgroups()
			calls, callsErr := r.op25.ActiveCalls()
			if talkgroupErr != nil || callsErr != nil {
				continue
			}
			r.syncP25Mixer(profile, talkgroups, calls)
			current := make(map[string]P25ActiveCall)
			for _, call := range calls {
				key := p25CallKey(call)
				current[key] = call
			}
			for key, call := range previous {
				if _, active := current[key]; active {
					continue
				}
				label := fmt.Sprintf("Talkgroup %d", call.Grant.GroupID)
				if call.Talkgroup != nil && call.Talkgroup.AlphaTag != "" {
					label = call.Talkgroup.AlphaTag
				}
				protocol := "P25"
				if strings.Contains(strings.ToLower(call.Grant.Protocol), "phase2") {
					protocol = "P25 Phase 2"
				}
				duration := call.LastHeardAt.Sub(call.StartedAt).Seconds()
				if duration < 0 {
					duration = 0
				}
				event := TransmissionEvent{ID: NewID(), StartedAt: call.StartedAt, DurationSeconds: duration,
					FrequencyHz: float64(call.Grant.FrequencyHz), BandwidthHz: 12_500, SignalDBFS: -30, NoiseDBFS: -80,
					Modulation: "Digital", ProtocolName: &protocol, Label: &label, DeviceID: call.DeviceSerial, Confidence: .99}
				_ = r.Events.Append(event)
			}
			previous = current
		}
	}
}

func p25CallKey(call P25ActiveCall) string {
	return fmt.Sprintf("%s:%d:%d:%d", call.Grant.System, call.Grant.GroupID, call.Grant.FrequencyHz, call.StartedAt.UnixNano())
}

func (r *Runtime) syncP25Mixer(profile ScanProfile, talkgroups []P25TalkgroupState, calls []P25ActiveCall) {
	active := make(map[uint32]P25ActiveCall)
	for _, call := range calls {
		if call.Grant.GroupID != 0 {
			active[call.Grant.GroupID] = call
		}
	}
	defaultSystem := "P25"
	if systems := enabledP25Systems(profile); len(systems) > 0 {
		defaultSystem = systems[0].Name
	}
	r.mu.Lock()
	byID := make(map[uint32]int)
	for index := range r.mixer {
		if r.mixer[index].TalkgroupID != nil {
			byID[*r.mixer[index].TalkgroupID] = index
		}
	}
	for _, talkgroup := range talkgroups {
		index, exists := byID[talkgroup.ID]
		if !exists {
			name := talkgroup.AlphaTag
			if name == "" {
				name = fmt.Sprintf("Talkgroup %d", talkgroup.ID)
			}
			id, systemName := talkgroup.ID, defaultSystem
			channel := ChannelDefinition{ID: p25MixerID(id), Name: name, Mode: "p25", Enabled: true, Priority: talkgroup.Priority, BandwidthHz: 12_500}
			r.mixer = append(r.mixer, MixerChannel{ID: channel.ID, Kind: "talkgroup", Channel: channel, SystemName: &systemName,
				TalkgroupID: &id, Discovered: talkgroup.Discovered, Muted: talkgroup.Mute || talkgroup.Lockout, Volume: .8})
			index = len(r.mixer) - 1
			byID[id] = index
		}
		if call, ok := active[talkgroup.ID]; ok {
			r.mixer[index].Active = true
			r.mixer[index].Level = .72
			r.mixer[index].Encrypted = call.Grant.Encrypted
			if call.Grant.System != "" {
				systemName := call.Grant.System
				r.mixer[index].SystemName = &systemName
			}
		} else {
			r.mixer[index].Active = false
			r.mixer[index].Level = 0
		}
	}
	feeds := make([]struct {
		id      uint32
		channel string
	}, 0, len(byID))
	for id, index := range byID {
		feeds = append(feeds, struct {
			id      uint32
			channel string
		}{id: id, channel: r.mixer[index].ID})
	}
	r.mu.Unlock()
	for _, feed := range feeds {
		r.op25.EnsureAudio(feed.id, feed.channel, r.audioHub)
	}
}
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (r *Runtime) simulationLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(2400 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			r.generateDemo()
		}
	}
}
func (r *Runtime) generateDemo() {
	r.mu.RLock()
	if !r.running || r.active == nil {
		r.mu.RUnlock()
		return
	}
	profile := *r.active
	r.mu.RUnlock()
	type choice struct {
		frequency       float64
		modulation      string
		protocol, label *string
	}
	choices := make([]choice, 0)
	for _, ch := range profile.Channels {
		mod := "NFM"
		switch ch.Mode {
		case "am":
			mod = "AM"
		case "wfm", "fm":
			mod = "WFM"
		case "digital", "p25", "dmr":
			mod = "Digital"
		}
		label := ch.Name
		choices = append(choices, choice{ch.FrequencyHz, mod, ch.Decoder, &label})
	}
	for _, scanRange := range profile.Ranges {
		for i := 1; i <= 3; i++ {
			mod := "NFM"
			var protocol *string
			if i == 3 {
				mod = "Digital"
				protocol = ptr("Unidentified digital")
			}
			label := scanRange.Name
			choices = append(choices, choice{scanRange.StartHz + (scanRange.EndHz-scanRange.StartHz)*float64(i)/4, mod, protocol, &label})
		}
	}
	if len(choices) == 0 {
		return
	}
	selected := choices[rand.Intn(len(choices))]
	signal := -48 + rand.Float64()*31
	event := TransmissionEvent{ID: NewID(), StartedAt: time.Now(), DurationSeconds: 1.5 + rand.Float64()*9.5, FrequencyHz: selected.frequency, BandwidthHz: 12500, SignalDBFS: signal, NoiseDBFS: -82 + rand.Float64()*14, Modulation: selected.modulation, ProtocolName: selected.protocol, Label: selected.label, DeviceID: "simulator-0", Confidence: .55 + rand.Float64()*.39, Simulated: true}
	if selected.modulation == "NFM" && rand.Intn(2) == 1 {
		event.Transcript = ptr("Demo transcript · awaiting live receiver audio")
	}
	_ = r.Events.Append(event)
	r.mu.Lock()
	for i := range r.mixer {
		active := r.mixer[i].Channel.FrequencyHz == event.FrequencyHz
		r.mixer[i].Active = active
		if active {
			r.mixer[i].Level = clamp((signal+60)/45, .08, 1)
		} else {
			r.mixer[i].Level = 0
		}
	}
	r.mu.Unlock()
}

var errInvalidRequest = errors.New("invalid request")
