package app

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const narrowbandAudioRate = 16_000

type DemodulationResult struct {
	Audio        []int16
	AudioRateHz  int
	SignalDBFS   float64
	NoiseDBFS    float64
	PeakDBFS     float64
	DroppedBytes int
}

// DemodulateIQ converts an interleaved 8-bit complex capture into mono PCM.
// AM and phase-discriminator FM are deliberately implemented in the service so
// basic analog reception never depends on a third-party decoder process.
func DemodulateIQ(data []byte, format SampleFormat, inputRate int, offsetHz float64, mode string) (DemodulationResult, error) {
	if inputRate < narrowbandAudioRate || len(data) < 4 {
		return DemodulationResult{}, errors.New("IQ capture is too small or has an invalid sample rate")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	outputRate := narrowbandAudioRate
	deviation := 5_000.0
	deemphasis := 75e-6
	if mode == "wfm" || mode == "fm" {
		outputRate = 48_000
		deviation = 75_000
	}
	if mode == "auto" || mode == "" {
		mode = "nfm"
	}
	if mode != "nfm" && mode != "wfm" && mode != "fm" && mode != "am" {
		return DemodulationResult{}, errors.New("built-in DSP supports AM, NFM, and WFM")
	}

	samples := len(data) / 2
	decimation := inputRate / outputRate
	if decimation < 1 {
		decimation = 1
		outputRate = inputRate
	}
	audio := make([]int16, 0, samples/decimation+1)
	phaseStep := -2 * math.Pi * offsetHz / float64(inputRate)
	oscillatorI, oscillatorQ := 1.0, 0.0
	stepI, stepQ := math.Cos(phaseStep), math.Sin(phaseStep)
	previousI, previousQ := 0.0, 0.0
	havePrevious := false
	dc := 0.0
	dcAlpha := 1.0 / math.Max(float64(inputRate)*0.25, 1)
	deemphasisAlpha := (1 / float64(outputRate)) / (deemphasis + 1/float64(outputRate))
	filtered := 0.0
	accumulator := 0.0
	count := 0
	power := 0.0
	blockPower := 0.0
	blockCount := 0
	blockSize := maxInt(1, inputRate/100)
	blocks := make([]float64, 0, samples/blockSize+1)
	peakPower := 0.0

	for index := 0; index+1 < len(data); index += 2 {
		i, q := iqSample(data[index], data[index+1], format)
		samplePower := i*i + q*q
		power += samplePower
		blockPower += samplePower
		blockCount++
		if samplePower > peakPower {
			peakPower = samplePower
		}
		if blockCount == blockSize {
			blocks = append(blocks, blockPower/float64(blockCount))
			blockPower, blockCount = 0, 0
		}

		mixedI := i*oscillatorI - q*oscillatorQ
		mixedQ := i*oscillatorQ + q*oscillatorI
		nextI := oscillatorI*stepI - oscillatorQ*stepQ
		oscillatorQ = oscillatorI*stepQ + oscillatorQ*stepI
		oscillatorI = nextI
		if index&8191 == 0 {
			norm := math.Hypot(oscillatorI, oscillatorQ)
			if norm > 0 {
				oscillatorI, oscillatorQ = oscillatorI/norm, oscillatorQ/norm
			}
		}

		value := 0.0
		if mode == "am" {
			magnitude := math.Hypot(mixedI, mixedQ)
			dc += (magnitude - dc) * dcAlpha
			value = magnitude - dc
		} else if havePrevious {
			realPart := mixedI*previousI + mixedQ*previousQ
			imagPart := mixedQ*previousI - mixedI*previousQ
			value = math.Atan2(imagPart, realPart) * float64(inputRate) / (2 * math.Pi * deviation)
		}
		previousI, previousQ, havePrevious = mixedI, mixedQ, true
		accumulator += value
		count++
		if count >= decimation {
			sample := accumulator / float64(count)
			if mode != "am" {
				filtered += deemphasisAlpha * (sample - filtered)
				sample = filtered
			}
			audio = append(audio, floatPCM(sample))
			accumulator, count = 0, 0
		}
	}
	if blockCount > 0 {
		blocks = append(blocks, blockPower/float64(blockCount))
	}
	averagePower := power / float64(samples)
	noisePower := averagePower
	if len(blocks) > 0 {
		sort.Float64s(blocks)
		noisePower = blocks[len(blocks)/5]
	}
	return DemodulationResult{
		Audio: audio, AudioRateHz: outputRate,
		SignalDBFS: powerDBFS(averagePower), NoiseDBFS: powerDBFS(noisePower), PeakDBFS: powerDBFS(peakPower),
		DroppedBytes: len(data) % 2,
	}, nil
}

func iqSample(iByte, qByte byte, format SampleFormat) (float64, float64) {
	if format == ComplexUnsigned8 {
		return (float64(iByte) - 127.5) / 127.5, (float64(qByte) - 127.5) / 127.5
	}
	return float64(int8(iByte)) / 128.0, float64(int8(qByte)) / 128.0
}

func powerDBFS(power float64) float64 {
	if power <= 1e-15 {
		return -150
	}
	return 10 * math.Log10(power/2)
}

func floatPCM(value float64) int16 {
	value = math.Max(-1, math.Min(1, value))
	return int16(math.Round(value * 32767))
}

func WriteMonoWAV(path string, samples []int16, sampleRate int) error {
	if sampleRate <= 0 {
		return errors.New("WAV sample rate must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	dataSize := uint32(len(samples) * 2)
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)
	if _, err := file.Write(header); err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, samples)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
