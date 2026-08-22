package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type surveyTarget struct {
	FrequencyHz float64
	BandwidthHz float64
	Mode        string
	Label       string
	Dwell       time.Duration
	Decoder     *string
}

type surveyTargetBatch struct {
	Targets    []surveyTarget
	SampleRate int
}

type mapperRunContext struct {
	JobID     string
	SessionID uint64
	Config    MapperConfig
}

func surveyTargets(profile ScanProfile) []surveyTarget {
	targets := make([]surveyTarget, 0, len(profile.Channels)+256)
	for _, channel := range profile.Channels {
		if !channel.Enabled {
			continue
		}
		mode := strings.ToLower(channel.Mode)
		if mode == "digital" || mode == "p25" || mode == "dmr" {
			continue
		}
		targets = append(targets, surveyTarget{FrequencyHz: channel.FrequencyHz, BandwidthHz: channel.BandwidthHz,
			Mode: mode, Label: channel.Name, Dwell: 450 * time.Millisecond, Decoder: channel.Decoder})
	}
	for _, scanRange := range profile.Ranges {
		if !scanRange.Enabled {
			continue
		}
		for frequency := scanRange.StartHz; frequency <= scanRange.EndHz; frequency += scanRange.StepHz {
			dwell := time.Duration(scanRange.DwellMilliseconds) * time.Millisecond
			if dwell < 200*time.Millisecond {
				dwell = 200 * time.Millisecond
			}
			targets = append(targets, surveyTarget{FrequencyHz: frequency, BandwidthHz: maxFloat(scanRange.StepHz, 12_500),
				Mode: scanRange.PreferredMode, Label: scanRange.Name, Dwell: dwell, Decoder: scanRange.Decoder})
		}
	}
	return targets
}

func liveSampleRate(device SDRDevice, target surveyTarget) int {
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		// HackRF's documented sampling path is best behaved at 8 MS/s or above;
		// GP-SDR digitally narrows the requested channel after capture.
		return 8_000_000
	}
	if canonicalDecoderID(stringValue(target.Decoder)) == "dump1090" {
		return 2_400_000
	}
	if target.Mode == "wfm" || target.Mode == "fm" {
		return 1_000_000
	}
	return 1_000_000
}

func automaticTunerSampleRate(device SDRDevice, request TunerRequest, fallback int) int {
	if request.HardwareCenterHz <= 0 {
		return fallback
	}
	required := (math.Abs(request.FrequencyHz-request.HardwareCenterHz) + request.BandwidthHz/2) / .44
	minimum, maximum := 225_000, 20_000_000
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		minimum = 2_000_000
	}
	if device.Kind == "RTL-SDR" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		maximum = 3_200_000
	}
	if device.SampleRateLimit != nil {
		maximum = int(math.Min(float64(maximum), *device.SampleRateLimit))
	}
	for _, rate := range []int{1_000_000, 2_000_000, 2_400_000, 3_200_000, 4_000_000, 8_000_000, 10_000_000, 12_000_000, 16_000_000, 20_000_000} {
		if rate >= minimum && rate <= maximum && float64(rate) >= required {
			return rate
		}
	}
	return fallback
}

func supportedUserSampleRate(rate int) bool {
	for _, supported := range []int{0, 1_000_000, 2_000_000, 2_400_000, 3_200_000, 4_000_000, 5_000_000, 8_000_000, 10_000_000, 12_000_000, 16_000_000, 20_000_000} {
		if rate == supported {
			return true
		}
	}
	return false
}

func compatibleUserSampleRate(device SDRDevice, requested, automatic int) int {
	if requested == 0 {
		return automatic
	}
	maximum := 20_000_000
	if device.Kind == "RTL-SDR" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		maximum = 3_200_000
	}
	if device.SampleRateLimit != nil {
		maximum = int(math.Min(float64(maximum), *device.SampleRateLimit))
	}
	if requested > maximum || (device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") && requested < 2_000_000) {
		return automatic
	}
	return requested
}

func surveyCaptureSpec(device SDRDevice, target surveyTarget, sampleRate int) CaptureSpec {
	tuningOffset := surveyTuningOffset(sampleRate, target)
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(target.FrequencyHz + tuningOffset)), SampleRateHz: sampleRate, GainDB: 20}
	// HackRF needs both IF (LNA) and baseband (VGA) gain for useful unattended
	// discovery. VGA-only scanning was about 24 dB less sensitive than the tuner
	// whenever the user had not saved a device calibration yet.
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		spec.LNAGainDB = 24
		spec.VGAGainDB = 20
	}
	if calibration := device.Calibration; calibration != nil {
		spec.PPMCorrection = calibration.PPMCorrection
		spec.LNAGainDB, spec.VGAGainDB = calibration.LNAGainDB, calibration.VGAGainDB
		spec.AmpEnabled = calibration.AmpEnabled
	}
	return spec
}

func mapperBatchSampleRate(device SDRDevice, config MapperConfig, targets []surveyTarget) int {
	automatic := 1_000_000
	for _, target := range targets {
		if candidate := liveSampleRate(device, target); candidate > automatic {
			automatic = candidate
		}
	}
	return compatibleUserSampleRate(device, config.SampleRateHz, automatic)
}

func mapperTargetsFitSampleWindow(targets []surveyTarget, sampleRate int) bool {
	if len(targets) == 0 || sampleRate <= 0 {
		return false
	}
	minimum, maximum := math.MaxFloat64, -math.MaxFloat64
	for _, target := range targets {
		bandwidth := math.Max(target.BandwidthHz, 12_500)
		// Local activity detection samples noise as far as two channel widths
		// from the target, so that evidence must fit in the usable passband too.
		minimum = math.Min(minimum, target.FrequencyHz-bandwidth*2)
		maximum = math.Max(maximum, target.FrequencyHz+bandwidth*2)
	}
	return maximum-minimum <= float64(sampleRate)*.84
}

func mapperJobTargetBatches(job MapperJob, device SDRDevice, targets []surveyTarget) []surveyTargetBatch {
	limit := job.Config.ConcurrentChannels
	if limit == 0 {
		limit = defaultMapperConcurrentChannels(job.Config.Mode)
	}
	limit = maxInt(1, minInt(32, limit))
	ordered := append([]surveyTarget(nil), targets...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].FrequencyHz < ordered[j].FrequencyHz })
	batches := make([]surveyTargetBatch, 0, (len(ordered)+limit-1)/limit)
	for len(ordered) > 0 {
		count := 1
		for count < len(ordered) && count < limit {
			candidate := ordered[:count+1]
			rate := mapperBatchSampleRate(device, job.Config, candidate)
			if !mapperTargetsFitSampleWindow(candidate, rate) {
				break
			}
			count++
		}
		group := append([]surveyTarget(nil), ordered[:count]...)
		batches = append(batches, surveyTargetBatch{Targets: group, SampleRate: mapperBatchSampleRate(device, job.Config, group)})
		ordered = ordered[count:]
	}
	return batches
}

func mapperBatchCaptureSpec(device SDRDevice, batch surveyTargetBatch) (CaptureSpec, bool) {
	if len(batch.Targets) == 0 || batch.SampleRate <= 0 {
		return CaptureSpec{}, false
	}
	if len(batch.Targets) == 1 {
		return surveyCaptureSpec(device, batch.Targets[0], batch.SampleRate), true
	}
	minimum, maximum := math.MaxFloat64, -math.MaxFloat64
	for _, target := range batch.Targets {
		bandwidth := math.Max(target.BandwidthHz, 12_500)
		minimum = math.Min(minimum, target.FrequencyHz-bandwidth*2)
		maximum = math.Max(maximum, target.FrequencyHz+bandwidth*2)
	}
	halfUsable := float64(batch.SampleRate) * .42
	lowerCenter, upperCenter := maximum-halfUsable, minimum+halfUsable
	if lowerCenter > upperCenter {
		return CaptureSpec{}, false
	}
	clampCenter := func(value float64) float64 { return math.Max(lowerCenter, math.Min(upperCenter, value)) }
	midpoint := (minimum + maximum) / 2
	candidates := []float64{lowerCenter, upperCenter, midpoint, clampCenter(midpoint - float64(batch.SampleRate)*.12), clampCenter(midpoint + float64(batch.SampleRate)*.12)}
	for _, target := range batch.Targets {
		guard := math.Max(50_000, target.BandwidthHz*2.5)
		candidates = append(candidates, clampCenter(target.FrequencyHz-guard), clampCenter(target.FrequencyHz+guard))
	}
	center, bestDistance := midpoint, -1.0
	for _, candidate := range candidates {
		nearest := math.MaxFloat64
		for _, target := range batch.Targets {
			nearest = math.Min(nearest, math.Abs(candidate-target.FrequencyHz))
		}
		if nearest > bestDistance {
			center, bestDistance = candidate, nearest
		}
	}
	spec := surveyCaptureSpec(device, batch.Targets[0], batch.SampleRate)
	spec.CenterFrequencyHz = int64(math.Round(center))
	return spec, true
}

// measureSurveyTarget compares power inside the channel being tested with
// nearby spectrum. Using raw capture power (or the largest individual IQ
// sample) makes normal wideband noise look active and attributes one signal to
// every tuning step covered by the SDR's much wider capture bandwidth.
func surveyTuningOffset(sampleRate int, target surveyTarget) float64 {
	// Direct-conversion receivers commonly have a fixed center spur. Keep the
	// desired channel one eighth of the sampled span away from DC, leaving ample
	// room for both the signal and adjacent-noise windows.
	offset := float64(sampleRate) / 8
	// The local noise estimator samples from one to two channel bandwidths on
	// either side of the target. Keep receiver DC beyond that entire window;
	// otherwise a wide-FM target at the old 1/8-span offset treated the center
	// spur as noise and could never clear squelch.
	minimum := target.BandwidthHz * 2.5
	if minimum < 50_000 {
		minimum = 50_000
	}
	if offset < minimum {
		offset = minimum
	}
	maximum := float64(sampleRate)*.42 - target.BandwidthHz/2
	if maximum > 0 && offset > maximum {
		offset = maximum
	}
	return offset
}

func measureSurveyTarget(data []byte, format SampleFormat, sampleRate int, centerFrequency float64, target surveyTarget) (ChannelSpectrumLevel, bool, error) {
	bandwidth := target.BandwidthHz
	if bandwidth <= 0 {
		bandwidth = 12_500
	}
	// Keep room on both sides for a local noise estimate. Wide steps describe
	// scan spacing, not permission to treat the entire SDR passband as a hit.
	maximumBandwidth := float64(sampleRate) / 4
	if bandwidth > maximumBandwidth {
		bandwidth = maximumBandwidth
	}
	const channelID = "survey-target"
	levels, err := MeasureChannelSpectrum(data, format, sampleRate, centerFrequency, []ChannelDefinition{{
		ID: channelID, FrequencyHz: target.FrequencyHz, BandwidthHz: bandwidth,
	}})
	if err != nil {
		return ChannelSpectrumLevel{}, false, err
	}
	level, measured := levels[channelID]
	// A real narrow carrier or FM pilot can occupy only a few FFT bins inside a
	// wide channel. Band-average power alone then misses it even when it is far
	// above the local floor. Discount the peak by a conservative 12 dB before
	// using it, which keeps random-noise peaks closed while retaining sparse RF.
	if peakAdjusted := level.PeakDB - 12; peakAdjusted > level.SignalDB {
		level.SignalDB = peakAdjusted
	}
	return level, measured, nil
}

func captureWindow(device SDRDevice, spec CaptureSpec, duration time.Duration, stop <-chan struct{}) ([]byte, SampleFormat, error) {
	stream, err := StartIQStream(device, spec)
	if err != nil {
		return nil, "", err
	}
	byteCount := int64(float64(spec.SampleRateHz*2) * duration.Seconds())
	if byteCount < 4096 {
		byteCount = 4096
	}
	data := make([]byte, byteCount)
	type result struct {
		count int
		err   error
	}
	done := make(chan result, 1)
	go func() {
		count, readErr := io.ReadFull(stream.Reader, data)
		done <- result{count: count, err: readErr}
	}()
	select {
	case <-stop:
		_ = stream.Close()
		return nil, stream.Format, fmt.Errorf("survey stopped")
	case outcome := <-done:
		_ = stream.Close()
		if outcome.err != nil && outcome.err != io.ErrUnexpectedEOF {
			return nil, stream.Format, outcome.err
		}
		return data[:outcome.count], stream.Format, nil
	case <-time.After(duration + 4*time.Second):
		_ = stream.Close()
		return nil, stream.Format, fmt.Errorf("receiver did not provide samples within %.1f seconds", (duration + 4*time.Second).Seconds())
	}
}

func (r *Runtime) liveSurveyLoop(stop <-chan struct{}, profile ScanProfile, device SDRDevice) {
	targets := surveyTargets(profile)
	if len(targets) == 0 {
		r.setRuntimeError("This profile has no analog channels or enabled scan frequencies.")
		return
	}
	isMapper := profile.ID == "mapper-session" && r.mapper != nil
	mapperConfig := MapperConfig{}
	var mapperSessionID uint64
	if isMapper {
		mapperConfig = r.mapper.Config()
		mapperSessionID = r.mapper.BeginSession(mapperConfig.Mode, len(targets))
		defer r.mapper.EndSession(mapperSessionID)
	}
	for {
		for index, target := range targets {
			select {
			case <-stop:
				return
			default:
			}
			listenFor := time.Duration(0)
			if isMapper && mapperConfig.Mode == "decipher" {
				listenFor = time.Duration(mapperConfig.DecipherListenSeconds) * time.Second
				// Whisper needs a meaningful speech window. Keep each capture bounded
				// for responsive Stop behavior while allowing up to 15 seconds of
				// continuous radio audio for local transcription and callsign parsing.
				target.Dwell = minDuration(listenFor, 15*time.Second)
			}
			if isMapper {
				r.mapper.BeginTarget(mapperSessionID, index, len(targets), target.FrequencyHz, target.Label, listenFor)
			}
			deadline := time.Now().Add(listenFor)
			for {
				var mapperRun *mapperRunContext
				if isMapper {
					mapperRun = &mapperRunContext{SessionID: mapperSessionID, Config: mapperConfig}
				}
				if !r.processSurveyTarget(stop, profile, device, target, mapperRun) {
					return
				}
				if listenFor == 0 || !time.Now().Before(deadline) {
					break
				}
			}
		}
		if isMapper {
			r.mapper.CompletePass(mapperSessionID)
		}
	}
}

func mapperJobTargets(job MapperJob, records []MapperFrequencyRecord) ([]surveyTarget, error) {
	config := job.Config
	if config.Mode == "decipher" {
		targets := make([]surveyTarget, 0, len(records))
		for _, record := range records {
			if record.Hits == 0 || config.StartHz > 0 && record.FrequencyHz < config.StartHz || config.EndHz > 0 && record.FrequencyHz > config.EndHz {
				continue
			}
			mode := strings.ToLower(strings.TrimSpace(record.Modulation))
			if config.PreferredMode != "" && config.PreferredMode != "auto" {
				mode = config.PreferredMode
			}
			if mode != "am" && mode != "wfm" && mode != "nfm" {
				mode = "auto"
			}
			bandwidth := 12_500.0
			if mode == "wfm" {
				bandwidth = 180_000
			}
			label := firstNonEmpty(record.Name, record.ProtocolName, "Mapped frequency")
			var decoder *string
			if record.CandidateDecoder != "" {
				value := record.CandidateDecoder
				decoder = &value
			}
			targets = append(targets, surveyTarget{FrequencyHz: record.FrequencyHz, BandwidthHz: bandwidth, Mode: mode, Label: label, Dwell: 15 * time.Second, Decoder: decoder})
		}
		if len(targets) == 0 {
			return nil, errors.New("Identify has no discovered frequencies in this range. Run Discovery first or widen its filter.")
		}
		return targets, nil
	}
	if config.StepHz <= 0 || config.EndHz < config.StartHz {
		return nil, errors.New("the Mapper discovery range is invalid")
	}
	count := int(math.Floor((config.EndHz-config.StartHz)/config.StepHz)) + 1
	if count <= 0 || count > 1_000_000 {
		return nil, errors.New("this Mapper job exceeds one million tuning steps; increase the step size or split the range between receivers")
	}
	targets := make([]surveyTarget, 0, count)
	dwell := time.Duration(config.DwellMilliseconds) * time.Millisecond
	for frequency := config.StartHz; frequency <= config.EndHz+config.StepHz*.001; frequency += config.StepHz {
		targets = append(targets, surveyTarget{FrequencyHz: frequency, BandwidthHz: maxFloat(config.StepHz, 12_500), Mode: firstNonEmpty(config.PreferredMode, "auto"), Label: job.Name, Dwell: dwell})
	}
	return targets, nil
}

func (r *Runtime) mapperJobLoop(job MapperJob, device SDRDevice, handle *mapperJobRuntime) {
	targets, err := mapperJobTargets(job, r.mapper.Status().Records)
	if err != nil {
		r.finishMapperJob(job.ID, handle, 0, err)
		return
	}
	config := job.Config
	batches := mapperJobTargetBatches(job, device, targets)
	if len(batches) == 0 {
		r.finishMapperJob(job.ID, handle, 0, errors.New("Mapper could not create a receiver pass for these frequencies"))
		return
	}
	sessionID := r.mapper.BeginJobSession(job.ID, len(targets), len(batches))
	profile := ScanProfile{ID: "mapper-job-" + job.ID, Name: job.Name, Settings: SurveySettings{NoiseMarginDB: 6, RecordAudio: config.Mode == "decipher", RecordIQForUnknown: true, TranscribeVoice: config.Transcribe}}
	for {
		targetIndex := 0
		for batchIndex, batch := range batches {
			select {
			case <-handle.stop:
				r.finishMapperJob(job.ID, handle, sessionID, nil)
				return
			default:
			}
			listenFor := time.Duration(0)
			if config.Mode == "decipher" {
				listenFor = time.Duration(config.DecipherListenSeconds) * time.Second
				for index := range batch.Targets {
					batch.Targets[index].Dwell = minDuration(listenFor, 15*time.Second)
				}
			}
			frequencies := make([]float64, 0, len(batch.Targets))
			for _, target := range batch.Targets {
				frequencies = append(frequencies, target.FrequencyHz)
			}
			label := batch.Targets[0].Label
			if len(batch.Targets) > 1 {
				label = fmt.Sprintf("%d channels in %.1f MHz capture", len(batch.Targets), float64(batch.SampleRate)/1e6)
			}
			r.mapper.BeginJobBatch(job.ID, sessionID, batchIndex, len(batches), targetIndex, len(targets), frequencies, label, listenFor)
			deadline := time.Now().Add(listenFor)
			for {
				if device.Kind == "Simulator" {
					wait := batch.Targets[0].Dwell
					if wait > 500*time.Millisecond {
						wait = 500 * time.Millisecond
					}
					select {
					case <-handle.stop:
						r.finishMapperJob(job.ID, handle, sessionID, nil)
						return
					case <-time.After(wait):
					}
					// Every simulated receiver exposes one stable active target per
					// pass so concurrent-job and provenance behavior is testable even
					// when a job contains fewer than four frequencies.
					for offset, target := range batch.Targets {
						active := targetIndex+offset == int(sessionID%uint64(len(targets)))
						name, mode, protocol, confidence, source := r.identifyMapperFrequency(target.FrequencyHz)
						r.mapper.ObserveJob(job.ID, device.ID, config, target.FrequencyHz, active, -46+rand.Float64()*14, -84+rand.Float64()*4, mode, protocol, name, "")
						if active {
							r.mapper.SetIdentification(target.FrequencyHz, source, confidence)
						}
					}
				} else if !r.processSurveyBatch(handle.stop, profile, device, batch, &mapperRunContext{JobID: job.ID, SessionID: sessionID, Config: config}) {
					r.finishMapperJob(job.ID, handle, sessionID, nil)
					return
				}
				if listenFor == 0 || !time.Now().Before(deadline) {
					break
				}
			}
			targetIndex += len(batch.Targets)
		}
		r.mapper.CompleteJobPass(job.ID, sessionID)
	}
}

func (r *Runtime) finishMapperJob(id string, handle *mapperJobRuntime, sessionID uint64, runError error) {
	if sessionID != 0 {
		r.mapper.EndJobSession(id, sessionID)
	}
	if runError != nil {
		r.mapper.SetJobError(id, runError.Error())
	}
	r.mu.Lock()
	if current, ok := r.mapperJobs[id]; ok && current == handle {
		delete(r.mapperJobs, id)
	}
	r.mu.Unlock()
}

// processSurveyTarget captures one short window. Identify mode calls this
// repeatedly for the selected per-channel listen period, keeping memory use
// bounded and Stop responsive even when the period is measured in days.
func (r *Runtime) processSurveyTarget(stop <-chan struct{}, profile ScanProfile, device SDRDevice, target surveyTarget, mapperRun *mapperRunContext) bool {
	rate := liveSampleRate(device, target)
	if mapperRun != nil {
		rate = compatibleUserSampleRate(device, mapperRun.Config.SampleRateHz, rate)
	}
	return r.processSurveyBatch(stop, profile, device, surveyTargetBatch{Targets: []surveyTarget{target}, SampleRate: rate}, mapperRun)
}

// processSurveyBatch captures one IQ window and fans it out through a bounded
// number of software VFOs. Every target therefore observes the same instant in
// time without opening competing receiver processes or multiplying USB load.
func (r *Runtime) processSurveyBatch(stop <-chan struct{}, profile ScanProfile, device SDRDevice, batch surveyTargetBatch, mapperRun *mapperRunContext) bool {
	spec, ok := mapperBatchCaptureSpec(device, batch)
	if !ok {
		r.setRuntimeError("Mapper channels do not fit inside the selected receiver bandwidth")
		return true
	}
	duration := time.Duration(0)
	for _, target := range batch.Targets {
		if target.Dwell > duration {
			duration = target.Dwell
		}
	}
	data, format, err := captureWindow(device, spec, duration, stop)
	if err != nil {
		select {
		case <-stop:
			return false
		default:
		}
		r.setRuntimeError(err.Error())
		time.Sleep(350 * time.Millisecond)
		return true
	}
	format = DetectSampleFormat(data, format)
	// DC removal is the safe scan default, even before a receiver-specific
	// calibration has been saved. It suppresses the HackRF/RTL center spur.
	removeDC, iqGain, iqPhase, iqSwap := true, 1.0, 0.0, false
	if calibration := device.Calibration; calibration != nil {
		removeDC, iqGain, iqPhase, iqSwap = calibration.DCRemoval, calibration.IQGain, calibration.IQPhase, calibration.IQSwap
	}
	ApplyIQCorrection(data, format, removeDC, iqGain, iqPhase, iqSwap)
	r.updateSpectrum(spec, data, format)
	for _, target := range batch.Targets {
		select {
		case <-stop:
			return false
		default:
		}
		if !r.processSurveyTargetCapture(stop, profile, device, target, mapperRun, spec, data, format) {
			return false
		}
	}
	return true
}

func (r *Runtime) processSurveyTargetCapture(stop <-chan struct{}, profile ScanProfile, device SDRDevice, target surveyTarget, mapperRun *mapperRunContext, spec CaptureSpec, data []byte, format SampleFormat) bool {
	rate := spec.SampleRateHz
	analysis := AnalyzeSignalIQ(data, format, rate, target.FrequencyHz-float64(spec.CenterFrequencyHz), target.BandwidthHz)
	demodulationMode := target.Mode
	if strings.EqualFold(demodulationMode, "auto") || strings.TrimSpace(demodulationMode) == "" {
		demodulationMode = strings.ToLower(analysis.Modulation)
		if demodulationMode != "am" && demodulationMode != "wfm" && demodulationMode != "nfm" {
			demodulationMode = "nfm"
		}
	}
	result, err := DemodulateIQ(data, format, rate, target.FrequencyHz-float64(spec.CenterFrequencyHz), demodulationMode)
	if err != nil {
		r.setRuntimeError(err.Error())
		return true
	}
	level, measured, err := measureSurveyTarget(data, format, rate, float64(spec.CenterFrequencyHz), target)
	if err != nil {
		r.setRuntimeError(err.Error())
		return true
	}
	margin := profile.Settings.NoiseMarginDB
	if margin < 6 {
		margin = 6
	}
	snr := level.SignalDB - level.NoiseDB
	active := measured && snr >= margin
	if !active {
		r.updateMixerActivity(target.FrequencyHz, 0, false)
		if mapperRun != nil && r.mapper != nil {
			r.mapper.ObserveJob(mapperRun.JobID, device.ID, mapperRun.Config, target.FrequencyHz, false, level.SignalDB, level.NoiseDB, "", "", "", "")
		}
		return true
	}
	r.clearRuntimeError()
	label := target.Label
	mode := strings.ToUpper(target.Mode)
	if mode == "" || mode == "AUTO" {
		mode = analysis.Modulation
		if mode == "UNKNOWN" || mode == "CARRIER" {
			mode = strings.ToUpper(demodulationMode)
		}
	}
	protocol := target.Decoder
	confidence := .72
	identificationSource := ""
	candidate, hasCandidate := decoderCandidate(target.FrequencyHz, stringValue(target.Decoder))
	if mapperRun != nil {
		identifiedName, identifiedMode, identifiedProtocol, identifiedConfidence, source := r.identifyMapperFrequency(target.FrequencyHz)
		if identifiedName != "" {
			label = identifiedName
		}
		if identifiedMode != "" {
			mode = identifiedMode
		}
		if identifiedProtocol != "" {
			protocol = &identifiedProtocol
		}
		confidence = math.Max(confidence, identifiedConfidence)
		identificationSource = source
	}
	if hasCandidate {
		protocol = &candidate.Protocol
		if label == "" || label == target.Label {
			label = candidate.Label
		}
		if mode == "" || mode == "AUTO" || mode == "NFM" {
			mode = candidate.Mode
		}
		if identificationSource == "" {
			identificationSource = "Decoder target · " + candidate.DecoderID
		}
	}
	if mapperRun != nil && r.mapper != nil {
		protocolName := ""
		if protocol != nil {
			protocolName = *protocol
		}
		r.mapper.ObserveJob(mapperRun.JobID, device.ID, mapperRun.Config, target.FrequencyHz, true, level.SignalDB, level.NoiseDB, mode, protocolName, label, "")
		r.mapper.SetIdentification(target.FrequencyHz, identificationSource, confidence)
		r.mapper.SetSignalIntelligence(target.FrequencyHz, analysis)
		if hasCandidate {
			r.mapper.SetDecoderEvidence(target.FrequencyHz, candidate.DecoderID, "candidate", candidate.Reason, r.decoderReady(candidate.DecoderID))
		}
	}
	mixerLevel := clamp((snr-margin)/24+.1, .08, 1)
	r.updateMixerActivity(target.FrequencyHz, mixerLevel, true)
	if channelID := r.mixerChannelID(target.FrequencyHz); channelID != "" && r.audioHub != nil {
		r.audioHub.Publish(AudioFrame{ChannelID: channelID, SampleRate: result.AudioRateHz, Samples: result.Audio})
	}
	if mapperRun != nil && r.mapper != nil && !r.mapper.ShouldArchive(target.FrequencyHz, 30*time.Second) {
		return true
	}
	event := TransmissionEvent{ID: NewID(), StartedAt: time.Now().Add(-target.Dwell), DurationSeconds: target.Dwell.Seconds(),
		FrequencyHz: target.FrequencyHz, BandwidthHz: target.BandwidthHz, SignalDBFS: level.SignalDB,
		NoiseDBFS: level.NoiseDB, Modulation: mode, ProtocolName: protocol, Label: &label,
		DeviceID: device.ID, Confidence: math.Max(confidence, analysis.Confidence), Analysis: &analysis}
	if (mode == "NFM" || mode == "FM") && len(result.Audio) > 0 {
		if tone, _, detected := DetectCTCSS(result.Audio, result.AudioRateHz); detected {
			event.CTCSSHz = &tone
		}
	}
	if mapperRun != nil && r.mapper != nil {
		event.Location = observationLocation(mapperRun.Config)
	}
	if profile.Settings.RecordAudio && len(result.Audio) > 0 {
		filename := fmt.Sprintf("%s-%.0f-%s.wav", time.Now().UTC().Format("20060102T150405.000Z"), target.FrequencyHz, strings.ToLower(mode))
		path := filepath.Join(r.dataDirectory, "Recordings", time.Now().UTC().Format("2006-01-02"), filename)
		if err := WriteMonoWAV(path, result.Audio, result.AudioRateHz); err == nil {
			event.AudioPath = &path
		}
	}
	unknownProtocol := protocol == nil || strings.Contains(strings.ToLower(stringValue(protocol)), "candidate") || analysis.Modulation == "UNKNOWN"
	if profile.Settings.RecordIQForUnknown && unknownProtocol && len(data) > 0 {
		maximumBytes := spec.SampleRateHz * 2 * 2 // at most two seconds of interleaved 8-bit IQ
		if maximumBytes > len(data) {
			maximumBytes = len(data)
		}
		if path, writeErr := writeIQEvidence(r.dataDirectory, target.FrequencyHz, spec, format, data[:maximumBytes]); writeErr == nil {
			event.IQPath = &path
		}
	}
	if err := r.Events.Append(event); err != nil {
		r.setRuntimeError(err.Error())
		return true
	}
	if hasCandidate && r.decoderReady(candidate.DecoderID) {
		go r.decodeEvent(stop, event, candidate.DecoderID, result.Audio, result.AudioRateHz, spec)
	}
	if profile.Settings.TranscribeVoice && event.AudioPath != nil {
		go r.transcribeEvent(stop, event.ID, event.FrequencyHz, *event.AudioPath)
	}
	return true
}

func (r *Runtime) decodeEvent(stop <-chan struct{}, event TransmissionEvent, decoderID string, audio []int16, audioRate int, spec CaptureSpec) {
	decoderContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-decoderContext.Done():
		}
	}()
	iqPath := ""
	if event.IQPath != nil {
		iqPath = *event.IQPath
	}
	messages, _ := runCandidateDecoder(decoderContext, decoderID, audio, audioRate, iqPath, event.FrequencyHz, spec)
	// Several command-line decoders return a non-zero exit code after EOF even
	// when they emitted valid frames. Parsed frames are the authoritative result.
	if len(messages) == 0 {
		return
	}
	_ = r.Events.UpdateDecoderMessages(event.ID, messages)
	if r.mapper != nil {
		r.mapper.SetDecodedMessages(event.FrequencyHz, decoderID, messages)
	}
}

func (r *Runtime) identifyMapperFrequency(frequencyHz float64) (name, mode, protocol string, confidence float64, source string) {
	for _, profile := range r.Profiles.All() {
		if profile.ID == "mapper-session" {
			continue
		}
		profileSource := "Saved profile · " + profile.Name
		if strings.HasPrefix(profile.ID, "localdb-") {
			profileSource = "Local database · " + profile.Name
		} else if strings.Contains(strings.ToLower(profile.Summary), "radioreference") {
			profileSource = "RadioReference import · " + profile.Name
		}
		for _, channel := range profile.Channels {
			if math.Abs(channel.FrequencyHz-frequencyHz) > 1 {
				continue
			}
			channelMode := strings.ToUpper(channel.Mode)
			channelProtocol := ""
			if channel.Decoder != nil {
				channelProtocol = *channel.Decoder
				if candidate, ok := decoderCandidate(frequencyHz, channelProtocol); ok {
					return firstNonEmpty(channel.Name, candidate.Label), candidate.Mode, candidate.Protocol, .64, "Decoder target · " + candidate.DecoderID
				}
			}
			if channelProtocol == "" {
				if channelMode == "AM" {
					channelProtocol = "Analog AM"
				} else if channelMode == "WFM" || channelMode == "NFM" || channelMode == "FM" {
					channelProtocol = "Analog FM"
				}
			}
			return channel.Name, channelMode, channelProtocol, .98, profileSource
		}
		for _, system := range profile.P25Systems {
			for _, controlFrequency := range system.ControlChannelsHz {
				if math.Abs(controlFrequency-frequencyHz) <= 1 {
					return system.Name, "DIGITAL", "P25 trunked control", .99, profileSource
				}
			}
		}
	}
	name, mode, protocol, confidence = identifyMappedFrequency(frequencyHz)
	if candidate, ok := decoderCandidate(frequencyHz, ""); ok {
		if name == "" {
			name = candidate.Label
		}
		if mode == "" {
			mode = candidate.Mode
		}
		protocol = candidate.Protocol
		confidence = math.Max(confidence, .64)
		if source == "" {
			source = "Built-in decoder target · " + candidate.DecoderID
		}
	}
	if name != "" {
		if source == "" {
			source = "Built-in US band plan"
		}
	}
	return
}

func identifyMappedFrequency(frequencyHz float64) (name, mode, protocol string, confidence float64) {
	MHz := frequencyHz / 1e6
	switch {
	case MHz >= 88 && MHz <= 108:
		return "FM broadcast", "WFM", "Analog FM", .9
	case MHz >= 108 && MHz < 118:
		return "Aircraft navigation", "AM", "Aviation", .7
	case MHz >= 118 && MHz <= 137:
		return "Aircraft voice", "AM", "Aviation AM", .85
	case math.Abs(MHz-1090) <= .5:
		return "Aircraft transponder", "DIGITAL", "ADS-B / Mode S candidate", .72
	case MHz >= 162.4 && MHz <= 162.55:
		return "NOAA Weather Radio", "NFM", "Analog FM", .9
	case MHz >= 462.5 && MHz <= 467.75:
		return "GMRS / FRS", "NFM", "Analog or digital land mobile", .75
	case MHz >= 769 && MHz <= 775:
		return "700 MHz public safety", "DIGITAL", "Likely P25", .78
	case MHz >= 851 && MHz <= 869:
		return "800 MHz trunked radio", "DIGITAL", "P25 or land mobile", .72
	}
	return "", "", "", 0
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (r *Runtime) decoderReady(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, decoder := range r.decoders {
		if decoder.ID == id {
			return decoder.State == "ready"
		}
	}
	return false
}

func (r *Runtime) mixerChannelID(frequencyHz float64) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.mixer {
		if math.Abs(item.Channel.FrequencyHz-frequencyHz) < 1 {
			return item.ID
		}
	}
	return ""
}

func (r *Runtime) transcribeEvent(stop <-chan struct{}, eventID string, frequencyHz float64, wavPath string) {
	transcriptionContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-transcriptionContext.Done():
		}
	}()
	transcript, err := r.transcriber.Transcribe(transcriptionContext, wavPath)
	if err != nil {
		r.setRuntimeError("Transcription: " + err.Error())
		return
	}
	if transcript != "" {
		callsigns := ExtractCallsigns(transcript)
		_ = r.Events.UpdateTranscript(eventID, transcript)
		if r.mapper != nil {
			r.mapper.SetTranscript(frequencyHz, transcript)
			if len(callsigns) > 0 {
				r.mapper.SetSignalIntelligence(frequencyHz, SignalIntelligence{Engine: "whisper.cpp + GP-SDR callsign parser", Callsigns: callsigns,
					Summary: "Local speech transcription produced one or more callsign candidates.", Evidence: []string{"Callsign pattern found in offline transcript"}})
			}
		}
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

type widebandTransmission struct {
	channel   ChannelDefinition
	startedAt time.Time
	audio     []int16
	audioRate int
	signalDB  float64
	noiseDB   float64
}

func widebandSpec(profile ScanProfile, device SDRDevice) (CaptureSpec, []ChannelDefinition, bool) {
	channels := make([]ChannelDefinition, 0)
	minimum, maximum, widest := math.MaxFloat64, 0.0, 0.0
	for _, channel := range profile.Channels {
		mode := strings.ToLower(channel.Mode)
		if !channel.Enabled || mode == "digital" || mode == "p25" || mode == "dmr" {
			continue
		}
		channels = append(channels, channel)
		minimum = math.Min(minimum, channel.FrequencyHz)
		maximum = math.Max(maximum, channel.FrequencyHz)
		widest = math.Max(widest, channel.BandwidthHz)
	}
	if len(channels) < 2 {
		return CaptureSpec{}, nil, false
	}
	required := maximum - minimum + math.Max(widest*2, 50_000)
	rates := []int{1_000_000, 2_000_000, 2_400_000, 3_200_000, 4_000_000, 8_000_000, 10_000_000, 12_000_000, 16_000_000, 20_000_000}
	minimumRate, maximumRate := 225_000, 20_000_000
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		minimumRate = 8_000_000
	}
	if device.Kind == "RTL-SDR" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		maximumRate = 3_200_000
	}
	if device.SampleRateLimit != nil {
		maximumRate = int(math.Min(float64(maximumRate), *device.SampleRateLimit))
	}
	selectedRate := 0
	for _, rate := range rates {
		if rate >= minimumRate && rate <= maximumRate && float64(rate)*.86 >= required {
			selectedRate = rate
			break
		}
	}
	if selectedRate == 0 {
		return CaptureSpec{}, nil, false
	}
	center := (minimum + maximum) / 2
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(center)), SampleRateHz: selectedRate, GainDB: 20}
	if calibration := device.Calibration; calibration != nil {
		spec.PPMCorrection = calibration.PPMCorrection
		spec.LNAGainDB, spec.VGAGainDB = calibration.LNAGainDB, calibration.VGAGainDB
		spec.AmpEnabled = calibration.AmpEnabled
	}
	return spec, channels, true
}

func (r *Runtime) widebandBankLoop(stop <-chan struct{}, profile ScanProfile, device SDRDevice) {
	spec, channels, ok := widebandSpec(profile, device)
	if !ok {
		r.setRuntimeError("This channel bank does not fit inside the assigned receiver bandwidth.")
		return
	}
	stream, err := StartIQStream(device, spec)
	if err != nil {
		r.setRuntimeError(err.Error())
		return
	}
	defer stream.Close()
	closed := make(chan struct{})
	defer close(closed)
	go func() {
		select {
		case <-stop:
			_ = stream.Close()
		case <-closed:
		}
	}()
	const frameDuration = 250 * time.Millisecond
	frameBytes := int(float64(spec.SampleRateHz*2) * frameDuration.Seconds())
	active := make(map[string]*widebandTransmission)
	finishAll := func() {
		for id, transmission := range active {
			r.finishWidebandTransmission(stop, profile, device, transmission)
			delete(active, id)
		}
	}
	defer finishAll()
	for {
		data := make([]byte, frameBytes)
		if _, err := io.ReadFull(stream.Reader, data); err != nil {
			select {
			case <-stop:
				return
			default:
				r.setRuntimeError("Wideband receiver: " + err.Error())
				return
			}
		}
		format := DetectSampleFormat(data, stream.Format)
		if calibration := device.Calibration; calibration != nil {
			ApplyIQCorrection(data, format, calibration.DCRemoval, calibration.IQGain, calibration.IQPhase, calibration.IQSwap)
		}
		r.updateSpectrum(spec, data, format)
		levels, err := MeasureChannelSpectrum(data, format, spec.SampleRateHz, float64(spec.CenterFrequencyHz), channels)
		if err != nil {
			r.setRuntimeError(err.Error())
			continue
		}
		for _, channel := range channels {
			level, measured := levels[channel.ID]
			snr := level.SignalDB - level.NoiseDB
			isActive := measured && snr >= profile.Settings.NoiseMarginDB
			if !isActive {
				r.updateMixerActivity(channel.FrequencyHz, 0, false)
				if transmission := active[channel.ID]; transmission != nil {
					r.finishWidebandTransmission(stop, profile, device, transmission)
					delete(active, channel.ID)
				}
				continue
			}
			result, err := DemodulateIQ(data, format, spec.SampleRateHz,
				channel.FrequencyHz-float64(spec.CenterFrequencyHz), channel.Mode)
			if err != nil {
				r.setRuntimeError(err.Error())
				continue
			}
			transmission := active[channel.ID]
			if transmission == nil {
				transmission = &widebandTransmission{channel: channel, startedAt: time.Now().Add(-frameDuration),
					audioRate: result.AudioRateHz, signalDB: level.SignalDB, noiseDB: level.NoiseDB}
				active[channel.ID] = transmission
			}
			transmission.signalDB = math.Max(transmission.signalDB, level.SignalDB)
			transmission.noiseDB = level.NoiseDB
			if profile.Settings.RecordAudio || profile.Settings.TranscribeVoice {
				transmission.audio = append(transmission.audio, result.Audio...)
			}
			mixerLevel := clamp((snr-profile.Settings.NoiseMarginDB)/24+.1, .08, 1)
			r.updateMixerActivity(channel.FrequencyHz, mixerLevel, true)
			if r.audioHub != nil {
				r.audioHub.Publish(AudioFrame{ChannelID: channel.ID, SampleRate: result.AudioRateHz, Samples: result.Audio})
			}
			if time.Since(transmission.startedAt) >= 2*time.Minute {
				r.finishWidebandTransmission(stop, profile, device, transmission)
				delete(active, channel.ID)
			}
		}
		r.clearRuntimeError()
	}
}

func (r *Runtime) tunerLoop(stop <-chan struct{}, profile ScanProfile, device SDRDevice, request TunerRequest, updates <-chan TunerRequest) {
	rate := request.SampleRateHz
	if rate == 0 {
		rate = liveSampleRate(device, surveyTarget{Mode: request.Mode})
		rate = automaticTunerSampleRate(device, request, rate)
	}
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") && rate < 2_000_000 {
		rate = 2_000_000
	}
	if device.Kind == "RTL-SDR" && !strings.HasPrefix(device.Driver, "SoapySDR:") && (rate < 225_000 || rate > 3_200_000) {
		rate = 1_000_000
	}
	centerFrequency := request.HardwareCenterHz
	if centerFrequency <= 0 {
		centerFrequency = request.FrequencyHz + surveyTuningOffset(rate, surveyTarget{FrequencyHz: request.FrequencyHz, BandwidthHz: request.BandwidthHz, Mode: request.Mode})
	}
	if math.Abs(request.FrequencyHz-centerFrequency)+request.BandwidthHz/2 > float64(rate)*.44 {
		r.setRuntimeError(fmt.Sprintf("Listen VFO %.6f MHz is outside the usable passband around hardware center %.6f MHz.", request.FrequencyHz/1e6, centerFrequency/1e6))
		return
	}
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(centerFrequency)), SampleRateHz: rate, GainDB: request.GainDB,
		PPMCorrection: request.PPMCorrection, LNAGainDB: request.LNAGainDB, VGAGainDB: request.VGAGainDB,
		AmpEnabled: request.AmpEnabled, AntennaPower: request.AntennaPower, AutoGain: request.AutoGain}
	stream, err := StartIQStream(device, spec)
	if err != nil {
		r.setRuntimeError(err.Error())
		return
	}
	defer stream.Close()
	closed := make(chan struct{})
	defer close(closed)
	go func() {
		select {
		case <-stop:
			_ = stream.Close()
		case <-closed:
		}
	}()
	const frameDuration = 50 * time.Millisecond
	frameBytes := int(float64(spec.SampleRateHz*2) * frameDuration.Seconds())
	noiseFloor := -150.0
	analysisFrames := 0
	latestAnalysis := SignalIntelligence{}
	for {
		select {
		case next := <-updates:
			request.FrequencyHz, request.Mode, request.BandwidthHz = next.FrequencyHz, next.Mode, next.BandwidthHz
			request.SquelchDB, request.MonitorOpen, request.AutoGain = next.SquelchDB, next.MonitorOpen, next.AutoGain
			request.NoiseReduction = next.NoiseReduction
			noiseFloor = -150
		default:
		}
		data := make([]byte, frameBytes)
		if _, err := io.ReadFull(stream.Reader, data); err != nil {
			select {
			case <-stop:
				return
			default:
				r.setRuntimeError("Tuner: " + err.Error())
				return
			}
		}
		format := DetectSampleFormat(data, stream.Format)
		ApplyIQCorrection(data, format, request.IQDCRemoval, request.IQGain, request.IQPhase, request.IQSwap)
		r.updateSpectrum(spec, data, format)
		analysisFrames++
		if analysisFrames == 1 || analysisFrames >= 10 {
			latestAnalysis = AnalyzeSignalIQ(data, format, spec.SampleRateHz, request.FrequencyHz-float64(spec.CenterFrequencyHz), request.BandwidthHz)
			analysisFrames = 1
		}
		demodulationMode := request.Mode
		if demodulationMode == "auto" {
			demodulationMode = strings.ToLower(latestAnalysis.Modulation)
			if demodulationMode != "am" && demodulationMode != "nfm" && demodulationMode != "wfm" {
				demodulationMode = "nfm"
			}
		}
		result, err := DemodulateIQ(data, format, spec.SampleRateHz, request.FrequencyHz-float64(spec.CenterFrequencyHz), demodulationMode)
		if err != nil {
			r.setRuntimeError(err.Error())
			continue
		}
		if noiseFloor <= -149 {
			noiseFloor = result.SignalDBFS
		}
		squelch := request.SquelchDB
		if squelch == 0 {
			squelch = profile.Settings.NoiseMarginDB
		}
		detected := result.SignalDBFS >= noiseFloor+squelch || result.PeakDBFS-result.NoiseDBFS >= squelch+6
		active := request.MonitorOpen || detected
		clipped := clippedIQPercent(data, format)
		telemetry := ReceiverTelemetry{DeviceID: device.ID, HardwareCenterHz: float64(spec.CenterFrequencyHz), ListenFrequencyHz: request.FrequencyHz,
			SampleRateHz: spec.SampleRateHz, SignalDBFS: result.SignalDBFS, NoiseDBFS: noiseFloor, PeakDBFS: result.PeakDBFS,
			ClippedPercent: clipped, Overloaded: clipped >= .5, SignalDetected: detected, SquelchOpen: active, LNAGainDB: spec.LNAGainDB, VGAGainDB: spec.VGAGainDB, AmpEnabled: spec.AmpEnabled}
		r.updateReceiverTelemetry(telemetry, &latestAnalysis)
		if !active {
			noiseFloor = noiseFloor*.96 + result.SignalDBFS*.04
			r.updateMixerActivity(request.FrequencyHz, 0, false)
			continue
		}
		r.clearRuntimeError()
		level := clamp((result.SignalDBFS-noiseFloor)/25, .08, 1)
		if request.AutoGain {
			applyAudioAGC(result.Audio)
		}
		applyNoiseReduction(result.Audio, result.AudioRateHz, request.NoiseReduction)
		r.updateMixerActivity(request.FrequencyHz, level, true)
		if r.audioHub != nil {
			r.audioHub.Publish(AudioFrame{ChannelID: "quick-tune-channel", SampleRate: result.AudioRateHz, Samples: result.Audio})
		}
	}
}

func clippedIQPercent(data []byte, format SampleFormat) float64 {
	if len(data) == 0 {
		return 0
	}
	clipped := 0
	for _, sample := range data {
		if format == ComplexUnsigned8 {
			if sample <= 1 || sample >= 254 {
				clipped++
			}
		} else {
			value := int8(sample)
			if value <= -127 || value >= 126 {
				clipped++
			}
		}
	}
	return float64(clipped) / float64(len(data)) * 100
}

// applyNoiseReduction is an entirely local, low-latency speech filter. It uses
// a gentle voice-band filter and downward expansion rather than a heavyweight
// ML runtime, so it remains responsive on both Intel and Apple Silicon Macs.
func applyNoiseReduction(samples []int16, sampleRate int, mode string) {
	if len(samples) == 0 || sampleRate <= 0 || mode == "" || mode == "off" {
		return
	}
	strength := 0.45
	if mode == "strong" {
		strength = 0.68
	}
	mean := 0.0
	for _, sample := range samples {
		mean += float64(sample)
	}
	mean /= float64(len(samples))
	rms := 0.0
	for _, sample := range samples {
		v := float64(sample) - mean
		rms += v * v
	}
	rms = math.Sqrt(rms / float64(len(samples)))
	gate := math.Max(90, rms*.18)
	alpha := math.Exp(-2 * math.Pi * 4200 / float64(sampleRate))
	low := 0.0
	for index, sample := range samples {
		v := float64(sample) - mean
		low = (1-alpha)*v + alpha*low
		if math.Abs(low) < gate {
			low *= 1 - strength
		}
		if low > 32767 {
			low = 32767
		}
		if low < -32768 {
			low = -32768
		}
		samples[index] = int16(low)
	}
}

func applyAudioAGC(samples []int16) {
	peak := 0
	for _, sample := range samples {
		value := int(sample)
		if value < 0 {
			value = -value
		}
		if value > peak {
			peak = value
		}
	}
	if peak < 256 {
		return
	}
	gain := 24000.0 / float64(peak)
	if gain > 8 {
		gain = 8
	}
	if gain < .25 {
		gain = .25
	}
	for index, sample := range samples {
		value := float64(sample) * gain
		if value > 32767 {
			value = 32767
		}
		if value < -32768 {
			value = -32768
		}
		samples[index] = int16(value)
	}
}

func (r *Runtime) finishWidebandTransmission(stop <-chan struct{}, profile ScanProfile, device SDRDevice, transmission *widebandTransmission) {
	if transmission == nil {
		return
	}
	duration := time.Since(transmission.startedAt).Seconds()
	if duration < .1 {
		duration = .1
	}
	mode := strings.ToUpper(transmission.channel.Mode)
	if mode == "" || mode == "AUTO" {
		mode = "NFM"
	}
	label := transmission.channel.Name
	event := TransmissionEvent{ID: NewID(), StartedAt: transmission.startedAt, DurationSeconds: duration,
		FrequencyHz: transmission.channel.FrequencyHz, BandwidthHz: transmission.channel.BandwidthHz,
		SignalDBFS: transmission.signalDB, NoiseDBFS: transmission.noiseDB, Modulation: mode,
		ProtocolName: transmission.channel.Decoder, Label: &label, DeviceID: device.ID, Confidence: .82}
	if profile.Settings.RecordAudio && len(transmission.audio) > 0 {
		filename := fmt.Sprintf("%s-%.0f-%s.wav", time.Now().UTC().Format("20060102T150405.000Z"), transmission.channel.FrequencyHz, strings.ToLower(mode))
		path := filepath.Join(r.dataDirectory, "Recordings", time.Now().UTC().Format("2006-01-02"), filename)
		if err := WriteMonoWAV(path, transmission.audio, transmission.audioRate); err == nil {
			event.AudioPath = &path
		}
	}
	if err := r.Events.Append(event); err != nil {
		r.setRuntimeError(err.Error())
		return
	}
	if transmission.channel.Decoder != nil && r.decoderReady(*transmission.channel.Decoder) {
		spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(transmission.channel.FrequencyHz)), SampleRateHz: transmission.audioRate}
		go r.decodeEvent(stop, event, *transmission.channel.Decoder, transmission.audio, transmission.audioRate, spec)
	}
	if profile.Settings.TranscribeVoice && event.AudioPath != nil {
		go r.transcribeEvent(stop, event.ID, event.FrequencyHz, *event.AudioPath)
	}
}
