package app

import (
	"math"
	"strings"
)

// mapperAdaptiveTuning owns the RF settings for one Mapper job. HackRF has no
// hardware AGC, so GP-SDR adjusts its RF, IF, and baseband stages between
// bounded captures. RTL-SDR and compatible Soapy devices use their driver's
// automatic tuner gain when Auto is selected.
type mapperAdaptiveTuning struct {
	deviceKind      string
	mode            string
	ampMode         string
	sensitivity     string
	gainDB          float64
	rtlGainDB       float64
	lnaGainDB       int
	vgaGainDB       int
	ampEnabled      bool
	quietCaptures   int
	stableCaptures  int
	lastDecision    string
	lastRMSDBFS     float64
	lastPeakDBFS    float64
	lastClipped     float64
	lastNoiseMargin float64
}

func newMapperAdaptiveTuning(device SDRDevice, config MapperConfig) *mapperAdaptiveTuning {
	state := &mapperAdaptiveTuning{deviceKind: device.Kind, mode: config.GainMode, ampMode: config.AmpMode,
		sensitivity: config.Sensitivity, gainDB: config.GainDB, rtlGainDB: 1.5, lnaGainDB: config.LNAGainDB,
		vgaGainDB: config.VGAGainDB, lastNoiseMargin: config.NoiseMarginDB}
	if state.mode == "" {
		state.mode = "auto"
	}
	if state.ampMode == "" {
		state.ampMode = "auto"
	}
	if state.sensitivity == "" {
		state.sensitivity = "auto"
	}
	if strings.EqualFold(device.Kind, "HackRF") {
		if state.mode != "manual" {
			state.lnaGainDB, state.vgaGainDB = 16, 16
			if device.Calibration != nil {
				if device.Calibration.LNAGainDB > 0 {
					state.lnaGainDB = device.Calibration.LNAGainDB
				}
				if device.Calibration.VGAGainDB > 0 {
					state.vgaGainDB = device.Calibration.VGAGainDB
				}
			}
		}
		if state.ampMode == "on" {
			state.ampEnabled = true
		} else if state.ampMode == "off" {
			state.ampEnabled = false
		} else if state.mode == "saved" && device.Calibration != nil {
			state.ampEnabled = device.Calibration.AmpEnabled
		}
	} else if strings.EqualFold(device.Kind, "RTL-SDR") && config.GainMode == "saved" && device.Calibration != nil && device.Calibration.VGAGainDB > 0 {
		// Older calibration files had no dedicated RTL gain field. Preserve a
		// useful saved tuner setting when one is available, but never fall back
		// to rtl_sdr's hardware-auto mode (which can clip strong FM carriers).
		state.rtlGainDB = float64(device.Calibration.VGAGainDB)
	}
	return state
}

func (state *mapperAdaptiveTuning) apply(spec CaptureSpec) CaptureSpec {
	if state == nil {
		return spec
	}
	switch state.mode {
	case "auto":
		if strings.EqualFold(state.deviceKind, "HackRF") {
			spec.LNAGainDB, spec.VGAGainDB, spec.AmpEnabled = state.lnaGainDB, state.vgaGainDB, state.ampEnabled
		} else if strings.EqualFold(state.deviceKind, "RTL-SDR") {
			// rtl_sdr's driver-auto gain is often maximally sensitive and can
			// saturate on nearby FM or paging transmitters. Use the safe first
			// manual step and adapt it from capture telemetry instead.
			spec.AutoGain = false
			spec.GainDB = state.rtlGainDB
		} else {
			spec.AutoGain = true
		}
	case "manual":
		spec.AutoGain = false
		spec.GainDB, spec.LNAGainDB, spec.VGAGainDB = state.gainDB, state.lnaGainDB, state.vgaGainDB
	case "saved":
		if strings.EqualFold(state.deviceKind, "RTL-SDR") {
			spec.AutoGain = false
			spec.GainDB = state.rtlGainDB
		}
	}
	if strings.EqualFold(state.deviceKind, "HackRF") {
		switch state.ampMode {
		case "on":
			spec.AmpEnabled = true
		case "off":
			spec.AmpEnabled = false
		case "auto":
			spec.AmpEnabled = state.ampEnabled
		}
	}
	return spec
}

func (state *mapperAdaptiveTuning) observe(data []byte, format SampleFormat) MapperTuningStatus {
	if state == nil {
		return MapperTuningStatus{}
	}
	rms, peak, clipped := mapperCaptureLevels(data, format)
	state.lastRMSDBFS, state.lastPeakDBFS, state.lastClipped = rms, peak, clipped
	state.lastDecision = "Gain stable"
	if state.mode == "auto" && strings.EqualFold(state.deviceKind, "HackRF") {
		state.adjustHackRF(rms, peak, clipped)
	} else if state.mode == "auto" && strings.EqualFold(state.deviceKind, "RTL-SDR") {
		state.adjustRTL(rms, peak, clipped)
	} else if state.mode == "auto" {
		state.lastDecision = "Receiver tuner AGC"
	} else if state.mode == "saved" {
		state.lastDecision = "Saved calibration"
	} else {
		state.lastDecision = "Manual RF settings"
	}
	return state.status()
}

// RTL-SDR tuner gain is device-specific and the driver's automatic setting
// can clip an otherwise perfectly usable signal. These conservative steps are
// accepted by common E4000 and R820-family tuners; rtl_sdr rounds to the nearest
// supported value when a particular tuner omits one.
var rtlAutoGainSteps = []float64{1.5, 9, 14, 19, 24, 29, 34, 42}

func (state *mapperAdaptiveTuning) adjustRTL(rmsDBFS, peakDBFS, clipped float64) {
	// A strong carrier can have a high RMS without clipping. Treat RMS alone as
	// overload only near full scale; clipping and peak headroom remain decisive.
	overloaded := clipped >= .5 || peakDBFS > -1 || (rmsDBFS > -3 && clipped > .01)
	if overloaded {
		for index := len(rtlAutoGainSteps) - 1; index > 0; index-- {
			if state.rtlGainDB >= rtlAutoGainSteps[index]-0.01 {
				state.rtlGainDB = rtlAutoGainSteps[index-1]
				state.lastDecision = "RTL gain reduced · overload protection"
				return
			}
		}
		state.rtlGainDB = rtlAutoGainSteps[0]
		state.lastDecision = "RTL minimum gain · overload protection"
		return
	}
	target := -34.0
	switch state.sensitivity {
	case "weak":
		target = -28
	case "balanced":
		target = -34
	case "conservative":
		target = -40
	}
	if rmsDBFS < target-5 && peakDBFS < -9 {
		state.quietCaptures++
		state.stableCaptures = 0
	} else {
		state.quietCaptures = 0
		state.stableCaptures++
	}
	if state.quietCaptures < 2 {
		state.lastDecision = "RTL gain stable"
		return
	}
	state.quietCaptures = 0
	for index, step := range rtlAutoGainSteps {
		if state.rtlGainDB < step-0.01 {
			state.rtlGainDB = step
			state.lastDecision = "RTL gain raised · weak input"
			return
		}
		if index == len(rtlAutoGainSteps)-1 {
			state.lastDecision = "RTL maximum automatic gain"
		}
	}
}

func (state *mapperAdaptiveTuning) adjustHackRF(rmsDBFS, peakDBFS, clipped float64) {
	overloaded := clipped >= .05 || peakDBFS > -1 || rmsDBFS > -12
	if overloaded {
		state.quietCaptures, state.stableCaptures = 0, 0
		if state.ampMode == "auto" && state.ampEnabled {
			state.ampEnabled = false
			state.lastDecision = "RF amp off · overload protection"
			return
		}
		if state.vgaGainDB >= 4 {
			state.vgaGainDB -= 4
			state.lastDecision = "VGA reduced · overload protection"
			return
		}
		if state.lnaGainDB >= 8 {
			state.lnaGainDB -= 8
			state.lastDecision = "LNA reduced · overload protection"
		}
		return
	}
	target := -34.0
	switch state.sensitivity {
	case "weak":
		target = -28
	case "balanced":
		target = -34
	case "conservative":
		target = -40
	}
	if rmsDBFS < target-5 && peakDBFS < -9 {
		state.quietCaptures++
		state.stableCaptures = 0
	} else {
		state.quietCaptures = 0
		state.stableCaptures++
	}
	// Requiring two quiet captures avoids enabling the ~11 dB RF stage for a
	// momentary lull. The amp is the first large step, and is also the first
	// stage removed when headroom disappears.
	if state.quietCaptures < 2 {
		return
	}
	state.quietCaptures = 0
	if state.ampMode == "auto" && !state.ampEnabled {
		state.ampEnabled = true
		state.lastDecision = "RF amp on · weak input"
		return
	}
	if state.lnaGainDB < 40 && state.lnaGainDB <= state.vgaGainDB+8 {
		state.lnaGainDB += 8
		state.lastDecision = "LNA raised · weak input"
		return
	}
	if state.vgaGainDB < 62 {
		state.vgaGainDB = minInt(62, state.vgaGainDB+4)
		state.lastDecision = "VGA raised · weak input"
		return
	}
	state.lastDecision = "Maximum automatic gain"
}

func (state *mapperAdaptiveTuning) setNoiseMargin(value float64) MapperTuningStatus {
	if state != nil {
		state.lastNoiseMargin = value
		return state.status()
	}
	return MapperTuningStatus{}
}

func (state *mapperAdaptiveTuning) status() MapperTuningStatus {
	if state == nil {
		return MapperTuningStatus{}
	}
	gain := state.gainDB
	if strings.EqualFold(state.deviceKind, "RTL-SDR") && state.mode == "auto" {
		gain = state.rtlGainDB
	}
	return MapperTuningStatus{Mode: state.mode, GainDB: gain, LNAGainDB: state.lnaGainDB,
		VGAGainDB: state.vgaGainDB, AmpEnabled: state.ampEnabled, NoiseMarginDB: state.lastNoiseMargin,
		RMSDBFS: state.lastRMSDBFS, PeakDBFS: state.lastPeakDBFS, ClippedPercent: state.lastClipped,
		Decision: state.lastDecision}
}

func mapperCaptureLevels(data []byte, format SampleFormat) (rmsDBFS, peakDBFS, clippedPercent float64) {
	if len(data) < 2 {
		return -150, -150, 0
	}
	power, peak, clipped, count := 0.0, 0.0, 0, 0
	stride := maxInt(2, (len(data)/400_000)&^1)
	for index := 0; index+1 < len(data); index += stride {
		i, q := iqSample(data[index], data[index+1], format)
		magnitude := math.Hypot(i, q) / math.Sqrt2
		power += magnitude * magnitude
		if magnitude > peak {
			peak = magnitude
		}
		if math.Abs(i) >= .992 || math.Abs(q) >= .992 {
			clipped++
		}
		count++
	}
	if count == 0 {
		return -150, -150, 0
	}
	rms := math.Sqrt(power / float64(count))
	return 20 * math.Log10(math.Max(rms, 1e-9)), 20 * math.Log10(math.Max(peak, 1e-9)), float64(clipped) * 100 / float64(count)
}

func mapperDetectionMargin(config MapperConfig, level ChannelSpectrumLevel, analysis SignalIntelligence) float64 {
	switch config.Sensitivity {
	case "weak":
		return 4
	case "balanced":
		return 6
	case "conservative":
		return 10
	case "manual":
		return clamp(config.NoiseMarginDB, 3, 30)
	}
	margin := 6.0
	switch {
	case level.NoiseDB > -55:
		margin = 10
	case level.NoiseDB > -70:
		margin = 8
	case level.NoiseDB < -100:
		margin = 4
	}
	if analysis.Modulation == "DIGITAL" && analysis.Confidence >= .65 {
		margin--
	}
	return clamp(margin, 4, 12)
}
