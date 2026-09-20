package app

import (
	"math"
	"testing"
	"time"
)

func TestDefaultChannelDwellAllowsFiniteFileDecodersToObserveFrames(t *testing.T) {
	for decoder, minimum := range map[string]time.Duration{
		"dump1090": 2 * time.Second,
		"rtl-433":  3 * time.Second,
		"ais":      3 * time.Second,
	} {
		value := decoder
		if got := defaultChannelDwell(&value); got < minimum {
			t.Fatalf("%s dwell %s is shorter than %s", decoder, got, minimum)
		}
	}
	if got := defaultChannelDwell(nil); got != 450*time.Millisecond {
		t.Fatalf("analog default dwell changed: %s", got)
	}
}

func TestMeasureSurveyTargetRejectsNoiseOnlyCapture(t *testing.T) {
	const rate = 1_000_000
	data := deterministicIQNoise(32_768, .08)
	level, measured, err := measureSurveyTarget(data, ComplexSigned8, rate, 155_000_000, surveyTarget{FrequencyHz: 155_000_000, BandwidthHz: 12_500})
	if err != nil {
		t.Fatal(err)
	}
	if !measured {
		t.Fatal("expected the target channel to be measured")
	}
	if snr := level.SignalDB - level.NoiseDB; snr >= 6 {
		t.Fatalf("noise-only capture must not open a 6 dB squelch; measured %.2f dB: %+v", snr, level)
	}
}

func TestMeasureSurveyTargetAcceptsCenteredCarrier(t *testing.T) {
	const rate = 1_000_000
	data := deterministicIQNoise(32_768, .025)
	for index := 0; index < len(data)/2; index++ {
		phase := 2 * math.Pi * 2_000 * float64(index) / rate
		i := float64(int8(data[index*2]))/128 + .45*math.Cos(phase)
		q := float64(int8(data[index*2+1]))/128 + .45*math.Sin(phase)
		data[index*2] = byte(int8(math.Round(clamp(i, -.99, .99) * 127)))
		data[index*2+1] = byte(int8(math.Round(clamp(q, -.99, .99) * 127)))
	}
	level, measured, err := measureSurveyTarget(data, ComplexSigned8, rate, 155_000_000, surveyTarget{FrequencyHz: 155_000_000, BandwidthHz: 12_500})
	if err != nil {
		t.Fatal(err)
	}
	if !measured || level.SignalDB-level.NoiseDB < 12 {
		t.Fatalf("expected centered carrier to clear the detector: %+v", level)
	}
}

func TestMeasureSurveyTargetAcceptsSparseEnergyInWideChannel(t *testing.T) {
	const rate = 2_000_000
	data := deterministicIQNoise(65_536, .04)
	for index := 0; index < len(data)/2; index++ {
		phase := 2 * math.Pi * 18_000 * float64(index) / rate
		i := float64(int8(data[index*2]))/128 + .16*math.Cos(phase)
		q := float64(int8(data[index*2+1]))/128 + .16*math.Sin(phase)
		data[index*2] = byte(int8(math.Round(clamp(i, -.99, .99) * 127)))
		data[index*2+1] = byte(int8(math.Round(clamp(q, -.99, .99) * 127)))
	}
	level, measured, err := measureSurveyTarget(data, ComplexSigned8, rate, 99_700_000,
		surveyTarget{FrequencyHz: 99_700_000, BandwidthHz: 180_000, Mode: "wfm"})
	if err != nil {
		t.Fatal(err)
	}
	if !measured || level.SignalDB-level.NoiseDB < 6 {
		t.Fatalf("expected sparse wide-channel energy to clear the detector: %+v", level)
	}
}

func TestSurveyTuningOffsetMovesTargetAwayFromReceiverDC(t *testing.T) {
	target := surveyTarget{FrequencyHz: 400_000_000, BandwidthHz: 12_500}
	offset := surveyTuningOffset(2_000_000, target)
	if offset < 200_000 || offset > 400_000 {
		t.Fatalf("unexpected 2 MHz scan offset: %.0f Hz", offset)
	}
}

func TestSurveyTuningOffsetKeepsWideFMNoiseWindowAwayFromDC(t *testing.T) {
	target := surveyTarget{FrequencyHz: 99_700_000, BandwidthHz: 180_000, Mode: "wfm"}
	offset := surveyTuningOffset(2_000_000, target)
	if offset < 400_000 || offset > 700_000 {
		t.Fatalf("wide-FM target leaves its local noise window on receiver DC: %.0f Hz", offset)
	}
}

func TestLiveSampleRateHonorsHackRFMinimumForWFM(t *testing.T) {
	device := SDRDevice{Kind: "HackRF", Driver: "/opt/homebrew/bin/hackrf_transfer"}
	if rate := liveSampleRate(device, surveyTarget{Mode: "wfm"}); rate != 10_000_000 {
		t.Fatalf("HackRF WFM survey must use 10 MS/s, got %d", rate)
	}
}

func TestMapperAutomaticBandwidthDoesNotTruncateBroadcastFM(t *testing.T) {
	config := MapperConfig{PreferredMode: "auto", PreferredDecoder: "auto", StepHz: 12_500}
	if width := mapperDetectionBandwidth(config, 98_100_000); width != 200_000 {
		t.Fatalf("98.1 MHz automatic width = %.0f, want 200000", width)
	}
	if width := mapperDetectionBandwidth(config, 98_112_500); width != 25_000 {
		t.Fatalf("off-raster FM probe width = %.0f, want 25000 to avoid duplicate wide hits", width)
	}
	config.DetectionBandwidthHz = 180_000
	if width := mapperDetectionBandwidth(config, 450_000_000); width != 180_000 {
		t.Fatalf("explicit analysis width = %.0f, want 180000", width)
	}
}

func TestAutomaticTunerRateFitsSeparatedHackRFVFO(t *testing.T) {
	limit := 20_000_000.0
	device := SDRDevice{Kind: "HackRF", SampleRateLimit: &limit}
	request := TunerRequest{FrequencyHz: 100_100_000, HardwareCenterHz: 98_000_000, BandwidthHz: 180_000}
	if rate := automaticTunerSampleRate(device, request, 2_000_000); rate != 10_000_000 {
		t.Fatalf("automatic rate = %d, want 10000000", rate)
	}
}

func TestPlutoAcceptsDriverReportedHighSampleRates(t *testing.T) {
	limit := 61_440_000.0
	device := SDRDevice{Kind: "PlutoSDR", Driver: "SoapySDR:plutosdr", SampleRateLimit: &limit}
	if rate := compatibleUserSampleRate(device, 40_000_000, 2_000_000); rate != 40_000_000 {
		t.Fatalf("Pluto high-rate request was clamped to %d", rate)
	}
	if rate := compatibleUserSampleRate(device, 61_440_000, 2_000_000); rate != 61_440_000 {
		t.Fatalf("Pluto driver maximum was clamped to %d", rate)
	}
}

func TestSurveyCaptureSpecUsesUsableHackRFDefaultsAndSavedCalibration(t *testing.T) {
	target := surveyTarget{FrequencyHz: 99_700_000, BandwidthHz: 180_000, Mode: "wfm"}
	device := SDRDevice{Kind: "HackRF", Driver: "/opt/homebrew/bin/hackrf_transfer"}
	spec := surveyCaptureSpec(device, target, 2_000_000)
	if spec.LNAGainDB != 24 || spec.VGAGainDB != 20 || spec.AmpEnabled {
		t.Fatalf("unexpected HackRF scan defaults: %+v", spec)
	}
	device.Calibration = &DeviceCalibration{LNAGainDB: 32, VGAGainDB: 28, AmpEnabled: true, PPMCorrection: -2}
	spec = surveyCaptureSpec(device, target, 2_000_000)
	if spec.LNAGainDB != 32 || spec.VGAGainDB != 28 || !spec.AmpEnabled || spec.PPMCorrection != -2 {
		t.Fatalf("saved calibration did not override scan defaults: %+v", spec)
	}
}

func TestSequentialSurveyAppliesSavedReceiverControls(t *testing.T) {
	limit := 20_000_000.0
	device := SDRDevice{Kind: "HackRF", SampleRateLimit: &limit}
	target := surveyTarget{FrequencyHz: 98_100_000, BandwidthHz: 180_000, Mode: "wfm"}
	rate := compatibleUserSampleRate(device, 5_000_000, liveSampleRate(device, target))
	if rate != 5_000_000 {
		t.Fatalf("Band Monitor requested rate = %d, want 5000000", rate)
	}
	lna, vga, amp := 16, 8, false
	spec := applySurveyProfileControls(surveyCaptureSpec(device, target, rate), SurveySettings{LNAGainDB: &lna, VGAGainDB: &vga, AmpEnabled: &amp})
	if spec.SampleRateHz != 5_000_000 || spec.LNAGainDB != lna || spec.VGAGainDB != vga || spec.AmpEnabled {
		t.Fatalf("sequential controls not applied: %+v", spec)
	}
}

func deterministicIQNoise(samples int, amplitude float64) []byte {
	data := make([]byte, samples*2)
	seed := uint32(1)
	for index := range data {
		seed = seed*1664525 + 1013904223
		value := (float64(int((seed>>24)&255)-128) / 128) * amplitude
		data[index] = byte(int8(math.Round(value * 127)))
	}
	return data
}
