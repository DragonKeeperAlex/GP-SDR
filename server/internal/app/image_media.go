package app

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
)

// NOAAAPTImage is a receive-only image result. It deliberately does not use
// DecoderMessage: an image is media evidence, not a packet-decoder claim.
type NOAAAPTImage struct {
	PNG             []byte
	Width           int
	Height          int
	Lines           int
	SampleRateHz    int
	Complete        bool
	Quality         float64
	ProcessingError string
}

const (
	noaaAPTLineSamples = 2080 // 0.5 s at the nominal 4160 Hz APT line rate
	noaaAPTChannelPix  = 909
)

// DecodeNOAAAPT converts demodulated NOAA APT audio into a two-channel
// grayscale PNG. It is intentionally conservative: input must contain whole
// APT lines and incomplete input is reported as incomplete rather than being
// presented as a successful image.
func DecodeNOAAAPT(pcm []int16, sampleRateHz int) (NOAAAPTImage, error) {
	result := NOAAAPTImage{SampleRateHz: sampleRateHz}
	if sampleRateHz <= 0 || len(pcm) < noaaAPTLineSamples {
		return result, errors.New("NOAA APT audio is too short or has an invalid sample rate")
	}
	lineCount := len(pcm) / noaaAPTLineSamples
	if lineCount < 2 {
		return result, errors.New("NOAA APT audio does not contain enough complete lines")
	}
	gray := image.NewGray(image.Rect(0, 0, noaaAPTChannelPix*2, lineCount))
	qualityTotal := 0.0
	for line := 0; line < lineCount; line++ {
		start := line * noaaAPTLineSamples
		// APT is an AM subcarrier after FM demodulation. Absolute value is a
		// bounded envelope approximation for the receive-only first pass.
		envelope := make([]float64, noaaAPTLineSamples)
		mean := 0.0
		for n := range envelope {
			envelope[n] = math.Abs(float64(pcm[start+n]))
			mean += envelope[n]
		}
		mean /= float64(len(envelope))
		variance := 0.0
		for _, value := range envelope {
			delta := value - mean
			variance += delta * delta
		}
		qualityTotal += math.Sqrt(variance/float64(len(envelope))) / math.Max(mean, 1)
		for pixel := 0; pixel < noaaAPTChannelPix; pixel++ {
			left := averageAPTWindow(envelope, pixel*2, pixel*2+1)
			right := averageAPTWindow(envelope, noaaAPTLineSamples/2+pixel, noaaAPTLineSamples/2+pixel+1)
			gray.SetGray(pixel, line, color.Gray{Y: aptLevel(left, mean)})
			gray.SetGray(noaaAPTChannelPix+pixel, line, color.Gray{Y: aptLevel(right, mean)})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, gray); err != nil {
		return result, err
	}
	result.PNG = encoded.Bytes()
	result.Width, result.Height, result.Lines = gray.Bounds().Dx(), gray.Bounds().Dy(), lineCount
	result.Quality = qualityTotal / float64(lineCount)
	result.Complete = len(pcm)%noaaAPTLineSamples == 0
	if !result.Complete {
		result.ProcessingError = "capture ended partway through an APT line"
	}
	return result, nil
}

func averageAPTWindow(values []float64, start, end int) float64 {
	if start < 0 {
		start = 0
	}
	if end > len(values) {
		end = len(values)
	}
	if start >= end {
		return 0
	}
	total := 0.0
	for _, value := range values[start:end] {
		total += value
	}
	return total / float64(end-start)
}

func aptLevel(value, mean float64) uint8 {
	if mean <= 0 {
		return 0
	}
	level := math.Round(255 * value / (mean * 2))
	if level < 0 {
		return 0
	}
	if level > 255 {
		return 255
	}
	return uint8(level)
}

// imageArtifactPNG validates that a media artifact is a real PNG before it is
// persisted by a future image-media store. Keeping this check local prevents
// partial decoder output from being mistaken for an accepted image.
func imageArtifactPNG(data []byte) error {
	if len(data) < 24 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return errors.New("image artifact is not a PNG")
	}
	if binary.BigEndian.Uint32(data[16:20]) == 0 || binary.BigEndian.Uint32(data[20:24]) == 0 {
		return errors.New("image artifact has invalid dimensions")
	}
	return nil
}
