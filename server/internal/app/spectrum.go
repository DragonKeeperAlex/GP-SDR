package app

import (
	"errors"
	"math"
	"math/cmplx"
	"time"
)

type ChannelSpectrumLevel struct {
	SignalDB float64
	NoiseDB  float64
	PeakDB   float64
}

func MeasureChannelSpectrum(data []byte, format SampleFormat, sampleRate int, centerFrequency float64, channels []ChannelDefinition) (map[string]ChannelSpectrumLevel, error) {
	const fftSize = 4096
	sampleCount := len(data) / 2
	if sampleRate <= 0 || sampleCount < fftSize {
		return nil, errors.New("capture is too short for spectrum analysis")
	}
	windowCount := sampleCount / fftSize
	if windowCount > 8 {
		windowCount = 8
	}
	accumulated := make([]float64, fftSize)
	buffer := make([]complex128, fftSize)
	for window := 0; window < windowCount; window++ {
		startSample := 0
		if windowCount > 1 {
			startSample = (sampleCount - fftSize) * window / (windowCount - 1)
		}
		for index := 0; index < fftSize; index++ {
			i, q := iqSample(data[(startSample+index)*2], data[(startSample+index)*2+1], format)
			hann := .5 - .5*math.Cos(2*math.Pi*float64(index)/float64(fftSize-1))
			buffer[index] = complex(i*hann, q*hann)
		}
		fftInPlace(buffer)
		for index, value := range buffer {
			power := cmplx.Abs(value)
			accumulated[index] += power * power / (fftSize * fftSize)
		}
	}
	for index := range accumulated {
		accumulated[index] /= float64(windowCount)
	}
	binWidth := float64(sampleRate) / fftSize
	levels := make(map[string]ChannelSpectrumLevel, len(channels))
	for _, channel := range channels {
		offset := channel.FrequencyHz - centerFrequency
		if math.Abs(offset) >= float64(sampleRate)/2 {
			continue
		}
		centerBin := int(math.Round(offset/binWidth)) % fftSize
		if centerBin < 0 {
			centerBin += fftSize
		}
		halfWidth := int(math.Ceil(channel.BandwidthHz / (2 * binWidth)))
		if halfWidth < 2 {
			halfWidth = 2
		}
		signalPower, peakPower, signalBins := 0.0, 0.0, 0
		for delta := -halfWidth; delta <= halfWidth; delta++ {
			power := accumulated[wrappedBin(centerBin+delta, fftSize)]
			signalPower += power
			if power > peakPower {
				peakPower = power
			}
			signalBins++
		}
		noisePower, noiseBins := 0.0, 0
		for _, direction := range []int{-1, 1} {
			for distance := halfWidth * 2; distance <= halfWidth*4; distance++ {
				noisePower += accumulated[wrappedBin(centerBin+direction*distance, fftSize)]
				noiseBins++
			}
		}
		signalPower /= float64(signalBins)
		noisePower /= float64(maxInt(noiseBins, 1))
		levels[channel.ID] = ChannelSpectrumLevel{SignalDB: 10 * math.Log10(math.Max(signalPower, 1e-15)),
			NoiseDB: 10 * math.Log10(math.Max(noisePower, 1e-15)), PeakDB: 10 * math.Log10(math.Max(peakPower, 1e-15))}
	}
	return levels, nil
}

func BuildSpectrumSnapshot(spec CaptureSpec, data []byte, format SampleFormat) (SpectrumSnapshot, error) {
	const fftSize = 8192
	const outputBins = 4096
	if spec.SampleRateHz <= 0 || len(data)/2 < fftSize {
		return SpectrumSnapshot{}, errors.New("capture is too short for waterfall analysis")
	}
	values := make([]complex128, fftSize)
	for index := range values {
		i, q := iqSample(data[index*2], data[index*2+1], format)
		window := .5 - .5*math.Cos(2*math.Pi*float64(index)/float64(fftSize-1))
		values[index] = complex(i*window, q*window)
	}
	fftInPlace(values)
	bins := make([]float64, outputBins)
	group := fftSize / outputBins
	for output := range bins {
		power := 0.0
		for offset := 0; offset < group; offset++ {
			shifted := (output*group + offset + fftSize/2) % fftSize
			magnitude := cmplx.Abs(values[shifted])
			power += magnitude * magnitude / float64(fftSize*fftSize)
		}
		bins[output] = 10 * math.Log10(math.Max(power/float64(group), 1e-15))
	}
	halfSpan := float64(spec.SampleRateHz) / 2
	return SpectrumSnapshot{CenterFrequencyHz: float64(spec.CenterFrequencyHz), StartFrequencyHz: float64(spec.CenterFrequencyHz) - halfSpan,
		EndFrequencyHz: float64(spec.CenterFrequencyHz) + halfSpan, SampleRateHz: spec.SampleRateHz, BinsDBFS: bins, CapturedAt: time.Now()}, nil
}

func (r *Runtime) updateSpectrum(spec CaptureSpec, data []byte, format SampleFormat) {
	snapshot, err := BuildSpectrumSnapshot(spec, data, format)
	if err != nil {
		return
	}
	r.mu.Lock()
	r.spectrum = snapshot
	r.mu.Unlock()
}

func wrappedBin(index, size int) int {
	index %= size
	if index < 0 {
		index += size
	}
	return index
}

func fftInPlace(values []complex128) {
	size := len(values)
	for i, j := 1, 0; i < size; i++ {
		bit := size >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			values[i], values[j] = values[j], values[i]
		}
	}
	for length := 2; length <= size; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		step := complex(math.Cos(angle), math.Sin(angle))
		for start := 0; start < size; start += length {
			rotation := complex(1.0, 0)
			for offset := 0; offset < length/2; offset++ {
				even := values[start+offset]
				odd := values[start+offset+length/2] * rotation
				values[start+offset] = even + odd
				values[start+offset+length/2] = even - odd
				rotation *= step
			}
		}
	}
}
