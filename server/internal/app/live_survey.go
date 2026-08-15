package app

import (
	"context"
	"fmt"
	"io"
	"math"
	"path/filepath"
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
		for frequency := scanRange.StartHz; frequency <= scanRange.EndHz && len(targets) < 20_000; frequency += scanRange.StepHz {
			dwell := time.Duration(scanRange.DwellMilliseconds) * time.Millisecond
			if dwell < 200*time.Millisecond {
				dwell = 200 * time.Millisecond
			}
			targets = append(targets, surveyTarget{FrequencyHz: frequency, BandwidthHz: maxFloat(scanRange.StepHz, 12_500),
				Mode: scanRange.PreferredMode, Label: scanRange.Name, Dwell: dwell})
		}
	}
	return targets
}

func liveSampleRate(device SDRDevice, target surveyTarget) int {
	if target.Mode == "wfm" || target.Mode == "fm" {
		return 1_000_000
	}
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") {
		return 2_000_000
	}
	return 1_000_000
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
	noiseFloor := make(map[string]float64)
	for {
		for _, target := range targets {
			select {
			case <-stop:
				return
			default:
			}
			rate := liveSampleRate(device, target)
			spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(target.FrequencyHz)), SampleRateHz: rate, GainDB: 20}
			data, format, err := captureWindow(device, spec, target.Dwell, stop)
			if err != nil {
				select {
				case <-stop:
					return
				default:
				}
				r.setRuntimeError(err.Error())
				time.Sleep(350 * time.Millisecond)
				continue
			}
			r.updateSpectrum(spec, data, format)
			result, err := DemodulateIQ(data, format, rate, 0, target.Mode)
			if err != nil {
				r.setRuntimeError(err.Error())
				continue
			}
			key := fmt.Sprintf("%.0f", target.FrequencyHz)
			baseline, established := noiseFloor[key]
			withinWindowBurst := result.PeakDBFS-result.NoiseDBFS >= profile.Settings.NoiseMarginDB+6
			active := established && result.SignalDBFS >= baseline+profile.Settings.NoiseMarginDB
			active = active || withinWindowBurst
			if !active {
				if !established {
					noiseFloor[key] = result.SignalDBFS
				} else {
					noiseFloor[key] = baseline*.9 + result.SignalDBFS*.1
				}
				r.updateMixerActivity(target.FrequencyHz, 0, false)
				continue
			}
			r.clearRuntimeError()
			label := target.Label
			mode := strings.ToUpper(target.Mode)
			if mode == "" || mode == "AUTO" {
				mode = "NFM"
			}
			event := TransmissionEvent{ID: NewID(), StartedAt: time.Now().Add(-target.Dwell), DurationSeconds: target.Dwell.Seconds(),
				FrequencyHz: target.FrequencyHz, BandwidthHz: target.BandwidthHz, SignalDBFS: result.SignalDBFS,
				NoiseDBFS: baseline, Modulation: mode, ProtocolName: target.Decoder, Label: &label,
				DeviceID: device.ID, Confidence: .72}
			if profile.Settings.RecordAudio && len(result.Audio) > 0 {
				filename := fmt.Sprintf("%s-%.0f-%s.wav", time.Now().UTC().Format("20060102T150405.000Z"), target.FrequencyHz, strings.ToLower(mode))
				path := filepath.Join(r.dataDirectory, "Recordings", time.Now().UTC().Format("2006-01-02"), filename)
				if err := WriteMonoWAV(path, result.Audio, result.AudioRateHz); err == nil {
					event.AudioPath = &path
				}
			}
			if err := r.Events.Append(event); err != nil {
				r.setRuntimeError(err.Error())
				continue
			}
			level := clamp((result.SignalDBFS-baseline)/30, .08, 1)
			r.updateMixerActivity(target.FrequencyHz, level, true)
			if channelID := r.mixerChannelID(target.FrequencyHz); channelID != "" && r.audioHub != nil {
				r.audioHub.Publish(AudioFrame{ChannelID: channelID, SampleRate: result.AudioRateHz, Samples: result.Audio})
			}
			if profile.Settings.TranscribeVoice && event.AudioPath != nil {
				go r.transcribeEvent(stop, event.ID, *event.AudioPath)
			}
		}
	}
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

func (r *Runtime) transcribeEvent(stop <-chan struct{}, eventID, wavPath string) {
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
		_ = r.Events.UpdateTranscript(eventID, transcript)
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
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
		minimumRate = 2_000_000
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
	return CaptureSpec{CenterFrequencyHz: int64(math.Round(center)), SampleRateHz: selectedRate, GainDB: 20}, channels, true
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
		r.updateSpectrum(spec, data, stream.Format)
		levels, err := MeasureChannelSpectrum(data, stream.Format, spec.SampleRateHz, float64(spec.CenterFrequencyHz), channels)
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
			result, err := DemodulateIQ(data, stream.Format, spec.SampleRateHz,
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

func (r *Runtime) tunerLoop(stop <-chan struct{}, profile ScanProfile, device SDRDevice, request TunerRequest) {
	rate := request.SampleRateHz
	if rate == 0 {
		rate = liveSampleRate(device, surveyTarget{Mode: request.Mode})
	}
	if device.Kind == "HackRF" && !strings.HasPrefix(device.Driver, "SoapySDR:") && rate < 2_000_000 {
		rate = 2_000_000
	}
	if device.Kind == "RTL-SDR" && !strings.HasPrefix(device.Driver, "SoapySDR:") && (rate < 225_000 || rate > 3_200_000) {
		rate = 1_000_000
	}
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(request.FrequencyHz)), SampleRateHz: rate, GainDB: request.GainDB}
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
	const frameDuration = 200 * time.Millisecond
	frameBytes := int(float64(spec.SampleRateHz*2) * frameDuration.Seconds())
	noiseFloor := -150.0
	for {
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
		r.updateSpectrum(spec, data, stream.Format)
		result, err := DemodulateIQ(data, stream.Format, spec.SampleRateHz, 0, request.Mode)
		if err != nil {
			r.setRuntimeError(err.Error())
			continue
		}
		if noiseFloor <= -149 {
			noiseFloor = result.SignalDBFS
		}
		active := result.SignalDBFS >= noiseFloor+profile.Settings.NoiseMarginDB || result.PeakDBFS-result.NoiseDBFS >= profile.Settings.NoiseMarginDB+6
		if !active {
			noiseFloor = noiseFloor*.96 + result.SignalDBFS*.04
			r.updateMixerActivity(request.FrequencyHz, 0, false)
			continue
		}
		r.clearRuntimeError()
		level := clamp((result.SignalDBFS-noiseFloor)/25, .08, 1)
		r.updateMixerActivity(request.FrequencyHz, level, true)
		if r.audioHub != nil {
			r.audioHub.Publish(AudioFrame{ChannelID: "quick-tune-channel", SampleRate: result.AudioRateHz, Samples: result.Audio})
		}
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
	if profile.Settings.TranscribeVoice && event.AudioPath != nil {
		go r.transcribeEvent(stop, event.ID, *event.AudioPath)
	}
}
