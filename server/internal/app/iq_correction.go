package app

import "math"

// DetectSampleFormat protects the receiver from HackRF-compatible firmware that
// exposes offset-binary samples even though hackrf_transfer normally emits
// signed bytes. A wrong interpretation looks like both I and Q are pinned near
// full scale and hides every real signal beneath a large DC component.
func DetectSampleFormat(data []byte, nominal SampleFormat) SampleFormat {
	if nominal != ComplexSigned8 || len(data) < 512 {
		return nominal
	}
	limit := len(data)
	if limit > 32768 {
		limit = 32768
	}
	signedMean, unsignedMean := 0.0, 0.0
	for _, raw := range data[:limit] {
		signedMean += float64(int8(raw))
		unsignedMean += float64(raw) - 127.5
	}
	signedMean = math.Abs(signedMean / float64(limit))
	unsignedMean = math.Abs(unsignedMean / float64(limit))
	if signedMean > 80 && unsignedMean < 40 {
		return ComplexUnsigned8
	}
	return nominal
}

// ApplyIQCorrection performs in-place DC removal, Q gain/phase balance, and I/Q swap.
func ApplyIQCorrection(data []byte, format SampleFormat, removeDC bool, qGain, phaseDegrees float64, swap bool) {
	if len(data) < 2 {
		return
	}
	if qGain == 0 {
		qGain = 1
	}
	meanI, meanQ := 0.0, 0.0
	if removeDC {
		for n := 0; n+1 < len(data); n += 2 {
			i, q := iqSample(data[n], data[n+1], format)
			meanI += i
			meanQ += q
		}
		count := float64(len(data) / 2)
		meanI /= count
		meanQ /= count
	}
	phase := phaseDegrees * math.Pi / 180
	c, s := math.Cos(phase), math.Sin(phase)
	for n := 0; n+1 < len(data); n += 2 {
		i, q := iqSample(data[n], data[n+1], format)
		i -= meanI
		q -= meanQ
		q = (q*c + i*s) * qGain
		if swap {
			i, q = q, i
		}
		if format == ComplexUnsigned8 {
			data[n] = byte(clampIQ(i*128 + 127.5))
			data[n+1] = byte(clampIQ(q*128 + 127.5))
		} else {
			data[n] = byte(int8(clampSigned(i * 128)))
			data[n+1] = byte(int8(clampSigned(q * 128)))
		}
	}
}

func clampIQ(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}
func clampSigned(v float64) int8 {
	if v < -128 {
		return -128
	}
	if v > 127 {
		return 127
	}
	return int8(math.Round(v))
}
