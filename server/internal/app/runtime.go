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
	mu                sync.RWMutex
	Profiles          *ProfileStore
	Events            *EventStore
	devices           []SDRDevice
	decoders          []DecoderDescriptor
	mixer             []MixerChannel
	plan              []ReceiverPlanItem
	running           bool
	startedAt         *time.Time
	active            *ScanProfile
	stop              chan struct{}
	demo              bool
	webAddress        string
	dataDirectory     string
	transcriber       *Transcriber
	lastError         *string
	droppedSamples    uint64
	op25              *OP25Manager
	radioReference    *radioReferenceClient
	audioHub          *AudioHub
	installer         *Installer
	rangeSync         *RangeSyncManager
	calibrations      *CalibrationStore
	mapper            *MapperManager
	mapperJobs        map[string]*mapperJobRuntime
	remoteReceivers   *RemoteReceiverStore
	localDatabase     *LocalDatabaseManager
	spectrum          SpectrumSnapshot
	tuning            bool
	tunerUpdates      chan TunerRequest
	tunerHardware     *TunerRequest
	lastAnalysis      *SignalIntelligence
	receiverTelemetry *ReceiverTelemetry
	storage           StorageStatus
	storageRefreshing bool
}

type mapperJobRuntime struct {
	deviceID string
	stop     chan struct{}
	stopping bool
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
	calibrations, err := NewCalibrationStore(filepath.Join(dataDirectory, "Data", "device-calibrations.json"))
	if err != nil {
		return nil, err
	}
	remoteReceivers, err := NewRemoteReceiverStore(filepath.Join(dataDirectory, "Data", "remote-receivers.json"))
	if err != nil {
		return nil, err
	}
	runtimeState := &Runtime{Profiles: profiles, Events: events, devices: DiscoverDevices(demo), decoders: DiscoverDecoders(), remoteReceivers: remoteReceivers,
		demo: demo, webAddress: webAddress, dataDirectory: dataDirectory, transcriber: NewTranscriber(dataDirectory), op25: &OP25Manager{}, mapperJobs: make(map[string]*mapperJobRuntime),
		radioReference: newRadioReferenceClient(), audioHub: NewAudioHub(), calibrations: calibrations}
	runtimeState.devices = append(runtimeState.devices, remoteDevices(remoteReceivers.List())...)
	runtimeState.attachCalibrations()
	runtimeState.installer = NewInstaller(dataDirectory, runtimeState.Refresh)
	runtimeState.mapper = NewMapperManager(dataDirectory, events)
	runtimeState.localDatabase = NewLocalDatabaseManager(dataDirectory, profiles)
	runtimeState.rangeSync, err = NewRangeSyncManager(dataDirectory, profiles)
	if err != nil {
		return nil, err
	}
	return runtimeState, nil
}

func (r *Runtime) RangeSyncStatus() RangeSyncStatus { return r.rangeSync.Status() }
func (r *Runtime) UpdateRangeSync(config RangeSyncConfig) (RangeSyncStatus, error) {
	return r.rangeSync.Update(config)
}
func (r *Runtime) SyncRangesNow() RangeSyncStatus { return r.rangeSync.SyncNow() }
func (r *Runtime) MapperStatus() MapperStatus     { return r.mapper.Status() }
func (r *Runtime) MapperProgress() MapperProgress { return r.mapper.Progress() }
func (r *Runtime) MapperJobs() []MapperJob        { return r.mapper.Jobs() }
func (r *Runtime) MapperJob(id string) (MapperJob, bool) {
	return r.mapper.Job(id)
}
func (r *Runtime) LocalDatabaseStatus() LocalDatabaseStatus {
	return r.localDatabase.Status()
}
func (r *Runtime) SetLocalDatabaseFolder(folder string) (LocalDatabaseStatus, error) {
	return r.localDatabase.SetFolder(folder)
}
func (r *Runtime) ScanLocalDatabase() LocalDatabaseStatus { return r.localDatabase.Scan() }
func (r *Runtime) UpdateMapper(config MapperConfig) (MapperStatus, error) {
	return r.mapper.Update(config)
}
func (r *Runtime) SaveMapperJob(job MapperJob) (MapperJob, error) { return r.mapper.SaveJob(job) }
func (r *Runtime) DeleteMapperJob(id string) error {
	r.mu.RLock()
	_, running := r.mapperJobs[id]
	r.mu.RUnlock()
	if running {
		return errors.New("stop this Mapper job before deleting it")
	}
	return r.mapper.DeleteJob(id)
}

func (r *Runtime) StartMapperJob(id string) (MapperStatus, error) {
	job, ok := r.mapper.Job(strings.TrimSpace(id))
	if !ok {
		return r.mapper.Status(), ErrNotFound
	}
	r.mu.Lock()
	if _, running := r.mapperJobs[job.ID]; running {
		r.mu.Unlock()
		return r.mapper.Status(), errors.New("this Mapper job is already running")
	}
	var device SDRDevice
	for _, candidate := range r.devices {
		if candidate.ID == job.Config.DeviceID && candidate.Connected && candidate.Available {
			device = candidate
			break
		}
	}
	if device.ID == "" {
		r.mu.Unlock()
		return r.mapper.Status(), errors.New("the assigned Mapper receiver is not connected or available")
	}
	for otherID, running := range r.mapperJobs {
		if running.deviceID == device.ID {
			r.mu.Unlock()
			return r.mapper.Status(), fmt.Errorf("%s is already assigned to Mapper job %s", device.Name, otherID)
		}
	}
	if r.running && r.profileUsesDeviceLocked(device.ID) {
		r.mu.Unlock()
		return r.mapper.Status(), fmt.Errorf("%s is already in use by the Live or Tuner workspace", device.Name)
	}
	handle := &mapperJobRuntime{deviceID: device.ID, stop: make(chan struct{})}
	r.mapperJobs[job.ID] = handle
	r.mu.Unlock()
	go r.mapperJobLoop(job, device, handle)
	return r.mapper.Status(), nil
}

func (r *Runtime) StopMapperJob(id string) MapperStatus {
	r.mu.Lock()
	if running, ok := r.mapperJobs[id]; ok {
		if !running.stopping {
			running.stopping = true
			r.mapper.MarkJobStopping(id)
			close(running.stop)
		}
	}
	r.mu.Unlock()
	return r.mapper.Status()
}

func (r *Runtime) StopAllMapperJobs() MapperStatus {
	r.mu.Lock()
	for id, running := range r.mapperJobs {
		if !running.stopping {
			running.stopping = true
			r.mapper.MarkJobStopping(id)
			close(running.stop)
		}
	}
	r.mu.Unlock()
	return r.mapper.Status()
}

func (r *Runtime) profileUsesDeviceLocked(deviceID string) bool {
	if !r.running || r.active == nil {
		return false
	}
	for _, item := range r.plan {
		if item.DeviceID != nil && *item.DeviceID == deviceID && item.State == "assigned" {
			return true
		}
	}
	return false
}
func (r *Runtime) UploadMapperNow() MapperStatus { return r.mapper.UploadNow() }
func (r *Runtime) UploadMapperFrequency(frequencyHz float64) MapperStatus {
	return r.mapper.UploadFrequency(frequencyHz)
}
func (r *Runtime) ClearMapperRecords() MapperStatus { return r.mapper.ClearRecords() }
func (r *Runtime) MapperCSV() ([]byte, int, error)  { return r.mapper.CSV() }
func (r *Runtime) SaveMapperCSV() (MapperExportResult, error) {
	return r.mapper.SaveCSV()
}
func (r *Runtime) SaveRadioReferenceCredentials(credentials RadioReferenceCredentialUpdate) (RadioReferenceStatus, error) {
	if err := saveRadioReferenceCredentials(credentials); err != nil {
		return RadioReferenceStatus{}, err
	}
	client := newRadioReferenceClient()
	r.mu.Lock()
	r.radioReference = client
	r.mu.Unlock()
	return client.Status(), nil
}
func (r *Runtime) ClearRadioReferenceCredentials() (RadioReferenceStatus, error) {
	if err := clearRadioReferenceCredentials(); err != nil {
		return RadioReferenceStatus{}, err
	}
	client := newRadioReferenceClient()
	r.mu.Lock()
	r.radioReference = client
	r.mu.Unlock()
	return client.Status(), nil
}
func (r *Runtime) RemoteReceivers() []RemoteReceiver { return r.remoteReceivers.List() }
func (r *Runtime) SaveRemoteReceiver(item RemoteReceiver) (RemoteReceiver, error) {
	saved, err := r.remoteReceivers.Save(item)
	if err == nil {
		r.Refresh()
	}
	return saved, err
}
func (r *Runtime) DeleteRemoteReceiver(id string) error {
	err := r.remoteReceivers.Delete(id)
	if err == nil {
		r.Refresh()
	}
	return err
}

func (r *Runtime) Refresh() {
	r.mu.RLock()
	running := r.running
	devices := append([]SDRDevice(nil), r.devices...)
	r.mu.RUnlock()
	// Hardware enumeration tools open USB devices briefly. Keep the last known
	// inventory while receiving so a Hardware refresh cannot contend with the
	// active DSP or trunking engine.
	r.mu.RLock()
	mapperRunning := len(r.mapperJobs) > 0
	r.mu.RUnlock()
	if !running && !mapperRunning {
		devices = DiscoverDevices(r.demo)
		devices = append(devices, remoteDevices(r.remoteReceivers.List())...)
	}
	decoders := DiscoverDecoders()
	transcriber := NewTranscriber(r.dataDirectory)
	radioReference := newRadioReferenceClient()
	r.mu.Lock()
	r.devices = devices
	r.decoders = decoders
	r.transcriber = transcriber
	r.radioReference = radioReference
	r.mu.Unlock()
	r.attachCalibrations()
}

func (r *Runtime) attachCalibrations() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.devices {
		if calibration, ok := r.calibrations.Get(r.devices[index].ID); ok {
			copy := calibration
			r.devices[index].Calibration = &copy
		} else {
			r.devices[index].Calibration = nil
		}
	}
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
	r.refreshStorageStatus()
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := RuntimeStatus{Running: r.running, Mode: "Idle", StartedAt: r.startedAt, ConnectedDeviceCount: 0,
		EventCount: r.Events.Count(), WebAddress: r.webAddress, SimulatorEnabled: r.demo, Version: Version,
		LastError: r.lastError, DroppedSamples: r.droppedSamples, SignalAnalysis: r.lastAnalysis, ReceiverTelemetry: r.receiverTelemetry,
		Storage: r.storage, HealthNotices: r.healthNoticesLocked()}
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

func (r *Runtime) healthNoticesLocked() []HealthNotice {
	notices := make([]HealthNotice, 0, 3)
	if r.droppedSamples > 0 {
		notices = append(notices, HealthNotice{ID: "dropped-samples", Level: "warning", Message: fmt.Sprintf("Receiver dropped %d sample blocks; lower the sample rate or use a direct USB connection.", r.droppedSamples)})
	}
	if telemetry := r.receiverTelemetry; telemetry != nil {
		if telemetry.Overloaded {
			notices = append(notices, HealthNotice{ID: "overload", Level: "warning", Message: fmt.Sprintf("Receiver input is overloaded (%.1f%% clipped). Reduce gain or disable the RF amplifier.", telemetry.ClippedPercent)})
		} else if r.running && telemetry.SampleRateHz > 0 && !telemetry.SignalDetected && telemetry.SignalDBFS-telemetry.NoiseDBFS < 2 {
			notices = append(notices, HealthNotice{ID: "noise-only", Level: "info", Message: "No signal is above the measured noise floor on the current channel."})
		}
	}
	if r.storage.RecordingBytes+r.storage.IQBytes >= 25*1024*1024*1024 {
		notices = append(notices, HealthNotice{ID: "capture-storage", Level: "warning", Message: "GP-SDR recordings and IQ captures use more than 25 GB. Review profile retention settings."})
	}
	return notices
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
	if request.Mode != "auto" && request.Mode != "am" && request.Mode != "nfm" && request.Mode != "wfm" {
		return errors.New("Tuner mode must be Auto, AM, NFM, or WFM.")
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
	deviceID := strings.TrimSpace(request.DeviceID)
	if request.IQGain == 0 {
		request.IQGain = 1
	}
	if request.UseCalibration && deviceID != "" {
		if calibration, ok := r.calibrations.Get(deviceID); ok {
			request.PPMCorrection += calibration.PPMCorrection
			request.IQGain *= calibration.IQGain
			request.IQPhase += calibration.IQPhase
			request.IQSwap = request.IQSwap != calibration.IQSwap
			request.IQDCRemoval = request.IQDCRemoval || calibration.DCRemoval
			if request.LNAGainDB == 0 {
				request.LNAGainDB = calibration.LNAGainDB
			}
			if request.VGAGainDB == 0 {
				request.VGAGainDB = calibration.VGAGainDB
			}
			request.AmpEnabled = request.AmpEnabled || calibration.AmpEnabled
		}
	}
	if request.GainDB < 0 || request.GainDB > 62 {
		return errors.New("Tuner gain must be between 0 and 62 dB.")
	}
	if request.LNAGainDB < 0 || request.LNAGainDB > 40 || request.LNAGainDB%8 != 0 {
		return errors.New("HackRF LNA gain must be 0 to 40 dB in 8 dB steps.")
	}
	if request.VGAGainDB < 0 || request.VGAGainDB > 62 || request.VGAGainDB%2 != 0 {
		return errors.New("HackRF VGA gain must be 0 to 62 dB in 2 dB steps.")
	}
	if request.PPMCorrection < -200 || request.PPMCorrection > 200 {
		return errors.New("Frequency correction must be between -200 and 200 PPM.")
	}
	if request.IQGain < .5 || request.IQGain > 1.5 || request.IQPhase < -20 || request.IQPhase > 20 {
		return errors.New("IQ gain or phase correction is outside the supported range.")
	}
	if request.SquelchDB < 0 || request.SquelchDB > 60 {
		return errors.New("Squelch must be between 0 and 60 dB above the noise floor.")
	}
	if request.NoiseReduction != "" && request.NoiseReduction != "off" && request.NoiseReduction != "voice" && request.NoiseReduction != "strong" {
		return errors.New("Noise reduction must be Off, Voice, or Strong.")
	}
	if request.LockCenter {
		updated, err := r.trySoftwareTune(request)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
	}
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

func (r *Runtime) Spectrum(maxBins int) SpectrumSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot := r.spectrum
	snapshot.BinsDBFS = append([]float64(nil), snapshot.BinsDBFS...)
	if maxBins >= 64 && maxBins < len(snapshot.BinsDBFS) {
		group := len(snapshot.BinsDBFS) / maxBins
		bins := make([]float64, maxBins)
		for index := range bins {
			total := 0.0
			for offset := 0; offset < group; offset++ {
				total += snapshot.BinsDBFS[index*group+offset]
			}
			bins[index] = total / float64(group)
		}
		snapshot.BinsDBFS = bins
	}
	return snapshot
}

func (r *Runtime) startProfile(profile ScanProfile, tuner *TunerRequest) error {
	r.Stop()
	if profile.Settings.MaxRecordingDays > 0 {
		go func(days int) {
			if _, err := pruneExpiredRecordings(r.dataDirectory, days, time.Now()); err != nil {
				r.setRuntimeError("Recording cleanup: " + err.Error())
			}
		}(profile.Settings.MaxRecordingDays)
	}
	plan := r.buildPlan(profile)
	liveDevices := make([]SDRDevice, 0)
	useP25 := len(enabledP25Systems(profile)) > 0
	useWideband := false
	if !r.demo {
		seenDevices := make(map[string]bool)
		for _, item := range plan {
			if item.DeviceID == nil || item.State != "assigned" {
				continue
			}
			for index := range r.devices {
				if r.devices[index].ID == *item.DeviceID && r.devices[index].Connected && !seenDevices[r.devices[index].ID] {
					liveDevices = append(liveDevices, r.devices[index])
					seenDevices[r.devices[index].ID] = true
					break
				}
			}
		}
		if len(liveDevices) == 0 {
			return errors.New("No SDR is connected. Connect a receiver and press Refresh, or launch with --demo.")
		}
		r.mu.RLock()
		mapperOwners := make(map[string]string, len(r.mapperJobs))
		for jobID, runningJob := range r.mapperJobs {
			mapperOwners[runningJob.deviceID] = jobID
		}
		r.mu.RUnlock()
		for _, device := range liveDevices {
			if jobID := mapperOwners[device.ID]; jobID != "" {
				return fmt.Errorf("%s is assigned to Mapper job %s; stop that job or choose another receiver", device.Name, jobID)
			}
		}
		if !useP25 && len(surveyTargets(profile)) == 0 {
			return errors.New("This profile has no AM, NFM, WFM, or automatic scan targets.")
		}
		if !useP25 && hasReceiverRole(profile, "channelBank") {
			_, _, useWideband = widebandSpec(profile, liveDevices[0])
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
	r.lastAnalysis = nil
	r.receiverTelemetry = nil
	r.tuning = tuner != nil
	if tuner != nil {
		r.tunerUpdates = make(chan TunerRequest, 1)
		tunerCopy := *tuner
		r.tunerHardware = &tunerCopy
	} else {
		r.tunerUpdates = nil
		r.tunerHardware = nil
	}
	stop := r.stop
	tunerUpdates := r.tunerUpdates
	demo := r.demo
	r.mu.Unlock()
	if demo {
		go r.simulationLoop(stop)
	} else if useP25 {
		go r.p25MonitorLoop(stop, profile)
	} else if tuner != nil {
		go r.tunerLoop(stop, profile, liveDevices[0], *tuner, tunerUpdates)
	} else if !useP25 && len(liveDevices) > 1 {
		for index, device := range liveDevices {
			partition := partitionScanProfile(profile, index, len(liveDevices))
			if _, _, wide := widebandSpec(partition, device); useWideband && wide {
				go r.widebandBankLoop(stop, partition, device)
			} else {
				go r.liveSurveyLoop(stop, partition, device)
			}
		}
	} else if !useP25 && useWideband {
		go r.widebandBankLoop(stop, profile, liveDevices[0])
	} else if !useP25 {
		go r.liveSurveyLoop(stop, profile, liveDevices[0])
	}
	return nil
}

func partitionScanProfile(profile ScanProfile, index, total int) ScanProfile {
	if total <= 1 {
		return profile
	}
	partition := profile
	partition.ID = fmt.Sprintf("%s-receiver-%d", profile.ID, index+1)
	partition.Channels = nil
	partition.Ranges = nil
	if len(profile.Channels) > 0 {
		start := index * len(profile.Channels) / total
		end := (index + 1) * len(profile.Channels) / total
		partition.Channels = append([]ChannelDefinition(nil), profile.Channels[start:end]...)
	}
	for _, scanRange := range profile.Ranges {
		if scanRange.StepHz <= 0 {
			continue
		}
		steps := int(math.Floor((scanRange.EndHz-scanRange.StartHz)/scanRange.StepHz)) + 1
		if steps <= 0 {
			continue
		}
		startStep := index * steps / total
		endStep := (index+1)*steps/total - 1
		if endStep < startStep {
			continue
		}
		piece := scanRange
		piece.ID = fmt.Sprintf("%s-part-%d", scanRange.ID, index+1)
		piece.StartHz = scanRange.StartHz + float64(startStep)*scanRange.StepHz
		piece.EndHz = scanRange.StartHz + float64(endStep)*scanRange.StepHz
		partition.Ranges = append(partition.Ranges, piece)
	}
	partition.DeviceAssignments = nil
	return partition
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
	r.tunerUpdates = nil
	r.tunerHardware = nil
	r.lastAnalysis = nil
	r.receiverTelemetry = nil
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

func (r *Runtime) updateReceiverTelemetry(telemetry ReceiverTelemetry, analysis *SignalIntelligence) {
	r.mu.Lock()
	copy := telemetry
	r.receiverTelemetry = &copy
	if analysis != nil {
		analysisCopy := *analysis
		analysisCopy.Evidence = append([]string(nil), analysis.Evidence...)
		r.lastAnalysis = &analysisCopy
	}
	r.mu.Unlock()
}

func (r *Runtime) trySoftwareTune(request TunerRequest) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running || !r.tuning || r.active == nil || r.active.ID != "quick-tune" || r.tunerUpdates == nil || r.tunerHardware == nil || r.spectrum.SampleRateHz <= 0 {
		return false, nil
	}
	if !sameTunerHardware(*r.tunerHardware, request) {
		return false, nil
	}
	center := r.spectrum.CenterFrequencyHz
	if request.HardwareCenterHz > 0 && math.Abs(request.HardwareCenterHz-center) > 1 {
		return false, errors.New("The requested hardware center no longer matches the active capture.")
	}
	usableHalfSpan := float64(r.spectrum.SampleRateHz) * .44
	if math.Abs(request.FrequencyHz-center)+request.BandwidthHz/2 > usableHalfSpan {
		return false, fmt.Errorf("%.6f MHz is outside the locked %.3f MHz usable passband.", request.FrequencyHz/1e6, usableHalfSpan*2/1e6)
	}
	request.HardwareCenterHz = center
	select {
	case r.tunerUpdates <- request:
	default:
		select {
		case <-r.tunerUpdates:
		default:
		}
		r.tunerUpdates <- request
	}
	if len(r.active.Channels) > 0 {
		r.active.Channels[0].FrequencyHz = request.FrequencyHz
		r.active.Channels[0].BandwidthHz = request.BandwidthHz
		r.active.Channels[0].Mode = request.Mode
	}
	if len(r.mixer) > 0 {
		r.mixer[0].Channel.FrequencyHz = request.FrequencyHz
		r.mixer[0].Channel.BandwidthHz = request.BandwidthHz
		r.mixer[0].Channel.Mode = request.Mode
	}
	return true, nil
}

func sameTunerHardware(active, next TunerRequest) bool {
	return active.DeviceID == next.DeviceID && active.SampleRateHz == next.SampleRateHz && active.GainDB == next.GainDB &&
		active.LNAGainDB == next.LNAGainDB && active.VGAGainDB == next.VGAGainDB && active.PPMCorrection == next.PPMCorrection &&
		active.AmpEnabled == next.AmpEnabled && active.AntennaPower == next.AntennaPower && active.AutoGain == next.AutoGain &&
		active.IQDCRemoval == next.IQDCRemoval && active.IQGain == next.IQGain && active.IQPhase == next.IQPhase && active.IQSwap == next.IQSwap
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
			p25MuteUpdates := make(map[uint32]bool)
			if muted != nil {
				r.mixer[i].Muted = *muted
			}
			if solo != nil {
				r.mixer[i].Solo = *solo
				if r.mixer[i].TalkgroupID != nil {
					for index := range r.mixer {
						if r.mixer[index].TalkgroupID == nil {
							continue
						}
						r.mixer[index].Solo = *solo && index == i
						r.mixer[index].Muted = r.mixer[index].Encrypted || (*solo && index != i)
						p25MuteUpdates[*r.mixer[index].TalkgroupID] = r.mixer[index].Muted
					}
				}
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
			if r.op25 != nil {
				for talkgroupID, mute := range p25MuteUpdates {
					if err := r.op25.UpdateTalkgroup(talkgroupID, map[string]any{"mute": mute}); err != nil {
						return updated, err
					}
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
				systemName, talkgroupID := call.Grant.System, call.Grant.GroupID
				var sourceRadioID *uint32
				if call.Grant.SourceID != 0 {
					source := call.Grant.SourceID
					sourceRadioID = &source
				}
				event := TransmissionEvent{ID: NewID(), StartedAt: call.StartedAt, DurationSeconds: duration,
					FrequencyHz: float64(call.Grant.FrequencyHz), BandwidthHz: 12_500, SignalDBFS: -30, NoiseDBFS: -80,
					Modulation: "Digital", ProtocolName: &protocol, Label: &label, DeviceID: call.DeviceSerial, Confidence: .99,
					SystemName: &systemName, TalkgroupID: &talkgroupID, SourceRadioID: sourceRadioID, Encrypted: call.Grant.Encrypted,
					Analysis: &SignalIntelligence{Engine: "SDRTrunk frame decoder", Modulation: "DIGITAL", SignalFamily: protocol, Confidence: .99,
						Evidence: []string{fmt.Sprintf("Talkgroup %d", call.Grant.GroupID), fmt.Sprintf("Source radio %d", call.Grant.SourceID)}, Summary: "SDRTrunk confirmed a framed " + protocol + " call."}}
				_ = r.Events.Append(event)
				if !call.Grant.Encrypted {
					go r.attachP25Recording(stop, profile, call, event.ID)
				}
			}
			previous = current
		}
	}
}

func (r *Runtime) attachP25Recording(stop <-chan struct{}, profile ScanProfile, call P25ActiveCall, eventID string) {
	root := filepath.Join(r.dataDirectory, "Runtime", "P25", profile.ID, "home", "SDRTrunk", "recordings")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	for {
		if source := findP25Recording(root, call.Grant.GroupID, call.StartedAt); source != "" {
			path, err := archiveP25Recording(r.dataDirectory, source, call.Grant.GroupID, call.StartedAt)
			if err == nil {
				_ = r.Events.UpdateAudioPath(eventID, path)
				if profile.Settings.TranscribeVoice {
					go r.transcribeEvent(stop, eventID, float64(call.Grant.FrequencyHz), path)
				}
			}
			return
		}
		select {
		case <-stop:
			return
		case <-deadline.C:
			return
		case <-ticker.C:
		}
	}
}

func p25CallKey(call P25ActiveCall) string {
	return fmt.Sprintf("%s:%d:%d:%d", call.Grant.System, call.Grant.GroupID, call.Grant.FrequencyHz, call.StartedAt.UnixNano())
}

func (r *Runtime) syncP25Mixer(profile ScanProfile, talkgroups []P25TalkgroupState, calls []P25ActiveCall) {
	type talkgroupHistory struct {
		count int
		last  time.Time
	}
	history := make(map[uint32]talkgroupHistory)
	for _, event := range r.Events.Recent(2000) {
		if event.TalkgroupID == nil || *event.TalkgroupID == 0 {
			continue
		}
		item := history[*event.TalkgroupID]
		item.count++
		if event.StartedAt.After(item.last) {
			item.last = event.StartedAt
		}
		history[*event.TalkgroupID] = item
	}
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
		item := history[talkgroup.ID]
		r.mixer[index].EventCount = item.count
		if !item.last.IsZero() {
			last := item.last
			r.mixer[index].LastHeardAt = &last
		}
		if call, ok := active[talkgroup.ID]; ok {
			r.mixer[index].Active = true
			r.mixer[index].Level = .72
			r.mixer[index].Encrypted = call.Grant.Encrypted
			last := call.LastHeardAt
			if last.IsZero() {
				last = call.StartedAt
			}
			r.mixer[index].LastHeardAt = &last
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
	r.mu.RLock()
	profile := ScanProfile{}
	if r.active != nil {
		profile = *r.active
	}
	r.mu.RUnlock()
	mapperTargets := surveyTargets(profile)
	mapperConfig := MapperConfig{}
	mapperIndex := 0
	var mapperSessionID uint64
	var mapperTargetEndsAt time.Time
	if profile.ID == "mapper-session" && r.mapper != nil && len(mapperTargets) > 0 {
		mapperConfig = r.mapper.Config()
		mapperSessionID = r.mapper.BeginSession(mapperConfig.Mode, len(mapperTargets))
		defer r.mapper.EndSession(mapperSessionID)
	}
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if mapperSessionID != 0 {
				target := mapperTargets[mapperIndex]
				listenFor := time.Duration(0)
				if mapperConfig.Mode == "decipher" {
					listenFor = time.Duration(mapperConfig.DecipherListenSeconds) * time.Second
				}
				if mapperTargetEndsAt.IsZero() {
					r.mapper.BeginTarget(mapperSessionID, mapperIndex, len(mapperTargets), target.FrequencyHz, target.Label, listenFor)
					mapperTargetEndsAt = time.Now().Add(listenFor)
				}
				active := rand.Intn(4) == 0
				name, mode, protocol, confidence, source := r.identifyMapperFrequency(target.FrequencyHz)
				r.mapper.Observe(target.FrequencyHz, active, -40+rand.Float64()*12, -82+rand.Float64()*5, mode, protocol, name, "")
				if active {
					r.mapper.SetIdentification(target.FrequencyHz, source, confidence)
				}
				if listenFor == 0 || !time.Now().Before(mapperTargetEndsAt) {
					mapperIndex++
					mapperTargetEndsAt = time.Time{}
					if mapperIndex >= len(mapperTargets) {
						mapperIndex = 0
						r.mapper.CompletePass(mapperSessionID)
					}
				}
			}
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
