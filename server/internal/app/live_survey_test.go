package app

import (
	"math"
	"testing"
)

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
	if rate := liveSampleRate(device, surveyTarget{Mode: "wfm"}); rate != 2_000_000 {
		t.Fatalf("HackRF WFM survey must use 2 Msps, got %d", rate)
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
