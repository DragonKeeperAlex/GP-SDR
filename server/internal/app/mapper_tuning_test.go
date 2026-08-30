package app

import (
	"strings"
	"testing"
)

func TestMapperAutoGainEnablesAndProtectsHackRFAmp(t *testing.T) {
	state := newMapperAdaptiveTuning(SDRDevice{Kind: "HackRF"}, MapperConfig{GainMode: "auto", AmpMode: "auto", Sensitivity: "auto", NoiseMarginDB: 6})
	quiet := make([]byte, 16_384)
	for index := range quiet {
		quiet[index] = 1
	}
	state.observe(quiet, ComplexSigned8)
	status := state.observe(quiet, ComplexSigned8)
	if !status.AmpEnabled || !strings.Contains(status.Decision, "RF amp on") {
		t.Fatalf("weak input should enable the HackRF RF amp after confirmation: %+v", status)
	}
	clipped := make([]byte, 16_384)
	for index := range clipped {
		clipped[index] = 127
	}
	status = state.observe(clipped, ComplexSigned8)
	if status.AmpEnabled || !strings.Contains(status.Decision, "overload protection") {
		t.Fatalf("clipping should remove the RF amp before other gain stages: %+v", status)
	}
}

func TestMapperManualAndDriverAutoGainModes(t *testing.T) {
	manual := newMapperAdaptiveTuning(SDRDevice{Kind: "HackRF"}, MapperConfig{GainMode: "manual", GainDB: 18, LNAGainDB: 32, VGAGainDB: 28, AmpMode: "on"})
	spec := manual.apply(CaptureSpec{GainDB: 20})
	if spec.GainDB != 18 || spec.LNAGainDB != 32 || spec.VGAGainDB != 28 || !spec.AmpEnabled || spec.AutoGain {
		t.Fatalf("manual HackRF controls were not applied: %+v", spec)
	}
	rtl := newMapperAdaptiveTuning(SDRDevice{Kind: "RTL-SDR"}, MapperConfig{GainMode: "auto", AmpMode: "auto"})
	spec = rtl.apply(CaptureSpec{GainDB: 20})
	if spec.AutoGain || spec.GainDB != 1.5 {
		t.Fatalf("RTL-SDR Auto should start at a safe manual tuner gain: %+v", spec)
	}
}

func TestMapperRTLAutomaticGainAvoidsClippingAndRaisesWeakInput(t *testing.T) {
	state := newMapperAdaptiveTuning(SDRDevice{Kind: "RTL-SDR"}, MapperConfig{GainMode: "auto", Sensitivity: "balanced"})
	clipped := make([]byte, 16_384)
	for index := range clipped {
		if index%2 == 0 {
			clipped[index] = 255
		} else {
			clipped[index] = 0
		}
	}
	status := state.observe(clipped, ComplexUnsigned8)
	if status.GainDB != 1.5 || !strings.Contains(status.Decision, "overload protection") {
		t.Fatalf("RTL clipping should hold the safe minimum gain: %+v", status)
	}
	quiet := make([]byte, 16_384)
	for index := range quiet {
		quiet[index] = 127
	}
	state.observe(quiet, ComplexUnsigned8)
	status = state.observe(quiet, ComplexUnsigned8)
	if status.GainDB <= 1.5 || !strings.Contains(status.Decision, "gain raised") {
		t.Fatalf("weak RTL input should raise gain after confirmation: %+v", status)
	}
}

func TestMapperAutomaticDetectionMarginTracksNoise(t *testing.T) {
	config := MapperConfig{Sensitivity: "auto", NoiseMarginDB: 6}
	quiet := mapperDetectionMargin(config, ChannelSpectrumLevel{NoiseDB: -110}, SignalIntelligence{})
	noisy := mapperDetectionMargin(config, ChannelSpectrumLevel{NoiseDB: -48}, SignalIntelligence{})
	if quiet != 4 || noisy != 10 {
		t.Fatalf("unexpected adaptive margins quiet=%.1f noisy=%.1f", quiet, noisy)
	}
	digital := mapperDetectionMargin(config, ChannelSpectrumLevel{NoiseDB: -80}, SignalIntelligence{Modulation: "DIGITAL", Confidence: .8})
	if digital != 5 {
		t.Fatalf("credible digital structure should receive a modest weak-signal allowance: %.1f", digital)
	}
}

func TestMapperTuningValidationDefaultsToAuto(t *testing.T) {
	config, err := validateMapperScanConfig(MapperConfig{Mode: "adaptive", DeviceID: "hackrf-0", StartHz: 100e6, EndHz: 101e6,
		StepHz: 12_500, DwellMilliseconds: 100, ConcurrentChannels: 4})
	if err != nil {
		t.Fatal(err)
	}
	if config.GainMode != "auto" || config.AmpMode != "auto" || config.Sensitivity != "auto" || config.NoiseMarginDB != 6 {
		t.Fatalf("new Mapper jobs should migrate to safe automatic controls: %+v", config)
	}
}
