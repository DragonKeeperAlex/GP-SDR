package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SignalFixtureRequest struct {
	Kind              string  `json:"kind"`
	Payload           string  `json:"payload,omitempty"`
	SymbolRate        float64 `json:"symbolRate,omitempty"`
	ToneHz            float64 `json:"toneHz,omitempty"`
	SNRDB             float64 `json:"snrDB,omitempty"`
	FrequencyOffsetHz float64 `json:"frequencyOffsetHz,omitempty"`
	DriftHzPerSecond  float64 `json:"driftHzPerSecond,omitempty"`
	IQGainError       float64 `json:"iqGainError,omitempty"`
	IQPhaseErrorDeg   float64 `json:"iqPhaseErrorDeg,omitempty"`
	DCOffset          float64 `json:"dcOffset,omitempty"`
	ClipLevel         float64 `json:"clipLevel,omitempty"`
}

type SignalFixtureManifest struct {
	SchemaVersion       int                 `json:"schemaVersion"`
	GeneratorVersion    string              `json:"generatorVersion"`
	NoiseSeed           int64               `json:"noiseSeed"`
	TrainingEligibility string              `json:"trainingEligibility"`
	ID                  string              `json:"id"`
	CreatedAt           time.Time           `json:"createdAt"`
	Kind                string              `json:"kind"`
	Description         string              `json:"description"`
	SampleRateHz        int                 `json:"sampleRateHz"`
	SampleCount         int                 `json:"sampleCount"`
	DurationSeconds     float64             `json:"durationSeconds"`
	SymbolRate          float64             `json:"symbolRate,omitempty"`
	PayloadUTF8         string              `json:"payloadUTF8,omitempty"`
	PayloadHex          string              `json:"payloadHex,omitempty"`
	ExpectedSymbols     int                 `json:"expectedSymbols,omitempty"`
	OccupiedBandwidthHz float64             `json:"occupiedBandwidthHz"`
	ConfiguredSNRDB     float64             `json:"configuredSNRDB,omitempty"`
	MeasuredEVMPercent  float64             `json:"measuredEVMPercent"`
	FrequencyOffsetHz   float64             `json:"frequencyOffsetHz,omitempty"`
	DriftHzPerSecond    float64             `json:"driftHzPerSecond,omitempty"`
	IQGainError         float64             `json:"iqGainError,omitempty"`
	IQPhaseErrorDeg     float64             `json:"iqPhaseErrorDeg,omitempty"`
	DCOffset            float64             `json:"dcOffset,omitempty"`
	ClipLevel           float64             `json:"clipLevel,omitempty"`
	IQPath              string              `json:"iqPath"`
	SHA256              string              `json:"sha256"`
	Source              string              `json:"source"`
	DecodeStatus        string              `json:"decodeStatus"`
	DecoderMessages     []DecoderMessage    `json:"decoderMessages,omitempty"`
	ReceiverMeasurement *FixtureMeasurement `json:"receiverMeasurement,omitempty"`
}

type FixtureMeasurement struct {
	ReceiverID       string  `json:"receiverID"`
	PeakDBFS         float64 `json:"peakDBFS"`
	NoiseDBFS        float64 `json:"noiseDBFS"`
	SNRDB            float64 `json:"snrDB"`
	FrequencyErrorHz float64 `json:"frequencyErrorHz"`
	LockMilliseconds int64   `json:"lockMilliseconds"`
	DroppedSamples   uint64  `json:"droppedSamples"`
}

func normalizeFixtureRequest(request SignalFixtureRequest) (SignalFixtureRequest, error) {
	for _, value := range []float64{request.SymbolRate, request.ToneHz, request.SNRDB, request.FrequencyOffsetHz, request.DriftHzPerSecond, request.IQGainError, request.IQPhaseErrorDeg, request.DCOffset, request.ClipLevel} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return request, errors.New("fixture parameters must be finite numbers")
		}
	}
	request.Kind = strings.ToLower(strings.TrimSpace(request.Kind))
	valid := map[string]bool{"cw": true, "am": true, "nfm": true, "wfm": true, "ook": true, "2fsk": true, "gfsk": true, "gmsk": true, "bpsk": true, "qpsk": true}
	if !valid[request.Kind] {
		return request, errors.New("choose CW, AM, NFM, WFM, OOK, 2-FSK, GFSK, GMSK, BPSK, or QPSK")
	}
	if request.SymbolRate <= 0 {
		request.SymbolRate = 1200
	}
	if request.SymbolRate > 250_000 {
		return request, errors.New("fixture symbol rate is limited to 250 ksym/s")
	}
	if request.ToneHz <= 0 {
		request.ToneHz = 1000
	}
	if request.SNRDB == 0 {
		request.SNRDB = 40
	}
	if request.SNRDB < -10 || request.SNRDB > 100 {
		return request, errors.New("SNR must be between -10 and 100 dB")
	}
	if request.ClipLevel == 0 {
		request.ClipLevel = 1
	}
	if request.ClipLevel < .05 || request.ClipLevel > 1 {
		return request, errors.New("clip level must be between 0.05 and 1.0")
	}
	if len(request.Payload) > 4096 {
		return request, errors.New("fixture payload is limited to 4096 characters")
	}
	if request.Payload == "" {
		request.Payload = "GP-SDR TEST"
	}
	return request, nil
}

func generateSignalFixture(request SignalFixtureRequest, sampleRate, sampleCount int) ([]byte, SignalFixtureManifest, error) {
	request, err := normalizeFixtureRequest(request)
	if err != nil {
		return nil, SignalFixtureManifest{}, err
	}
	if sampleRate <= 0 || sampleCount <= 0 || sampleCount > sampleRate*60 {
		return nil, SignalFixtureManifest{}, errors.New("invalid bounded fixture duration")
	}
	payload := []byte(request.Payload)
	bits := make([]byte, 0, len(payload)*8)
	for _, value := range payload {
		for bit := 7; bit >= 0; bit-- {
			bits = append(bits, (value>>bit)&1)
		}
	}
	if len(bits) == 0 {
		bits = []byte{0, 1}
	}
	rng := rand.New(rand.NewSource(0x4750534452))
	noiseSigma := math.Pow(10, -request.SNRDB/20) * .78 / math.Sqrt2
	phaseError := request.IQPhaseErrorDeg * math.Pi / 180
	output := make([]byte, sampleCount*2)
	errorPower, referencePower := 0.0, 0.0
	phase, symbolPhase := 0.0, 0.0
	samplesPerSymbol := max(1, int(math.Round(float64(sampleRate)/request.SymbolRate)))
	for i := 0; i < sampleCount; i++ {
		idealI, idealQ := 0.0, 0.0
		t := float64(i) / float64(sampleRate)
		symbol := i / samplesPerSymbol
		bit := bits[symbol%len(bits)]
		switch request.Kind {
		case "cw":
			idealI, idealQ = .78, 0
		case "am":
			amplitude := .52 + .25*math.Sin(2*math.Pi*request.ToneHz*t)
			idealI, idealQ = amplitude, 0
		case "nfm", "wfm":
			deviation := 2500.0
			if request.Kind == "wfm" {
				deviation = 75_000
			}
			phase += 2 * math.Pi * deviation * math.Sin(2*math.Pi*request.ToneHz*t) / float64(sampleRate)
			idealI, idealQ = .78*math.Cos(phase), .78*math.Sin(phase)
		case "ook":
			amplitude := .05
			if bit == 1 {
				amplitude = .78
			}
			idealI, idealQ = amplitude, 0
		case "2fsk", "gfsk", "gmsk":
			level := -1.0
			if bit == 1 {
				level = 1
			}
			if request.Kind != "2fsk" {
				fraction := float64(i%samplesPerSymbol) / float64(samplesPerSymbol)
				level *= .5 - .5*math.Cos(math.Pi*min(1.0, fraction*2))
			}
			deviation := request.SymbolRate * .5
			phase += 2 * math.Pi * deviation * level / float64(sampleRate)
			idealI, idealQ = .78*math.Cos(phase), .78*math.Sin(phase)
		case "bpsk":
			if bit == 1 {
				symbolPhase = math.Pi
			} else {
				symbolPhase = 0
			}
			idealI, idealQ = .78*math.Cos(symbolPhase), .78*math.Sin(symbolPhase)
		case "qpsk":
			second := bits[(symbol*2+1)%len(bits)]
			first := bits[(symbol*2)%len(bits)]
			symbolPhase = (float64(first*2+second) * math.Pi / 2) + math.Pi/4
			idealI, idealQ = .78*math.Cos(symbolPhase), .78*math.Sin(symbolPhase)
		}
		rotation := 2 * math.Pi * (request.FrequencyOffsetHz*t + .5*request.DriftHzPerSecond*t*t)
		iValue := idealI*(1+request.IQGainError) + request.DCOffset
		qValue := idealQ*(1-request.IQGainError) + request.DCOffset
		qValue = qValue*math.Cos(phaseError) + iValue*math.Sin(phaseError)
		rotI := iValue*math.Cos(rotation) - qValue*math.Sin(rotation)
		rotQ := iValue*math.Sin(rotation) + qValue*math.Cos(rotation)
		rotI += rng.NormFloat64() * noiseSigma
		rotQ += rng.NormFloat64() * noiseSigma
		rotI = max(-request.ClipLevel, min(request.ClipLevel, rotI))
		rotQ = max(-request.ClipLevel, min(request.ClipLevel, rotQ))
		output[2*i], output[2*i+1] = signedIQByte(rotI*127), signedIQByte(rotQ*127)
		errorPower += (rotI-idealI)*(rotI-idealI) + (rotQ-idealQ)*(rotQ-idealQ)
		referencePower += idealI*idealI + idealQ*idealQ
	}
	description, bandwidth := fixtureDescription(request.Kind), fixtureBandwidth(request)
	manifest := SignalFixtureManifest{SchemaVersion: 1, GeneratorVersion: "gpsdr-fixture-v1", NoiseSeed: 0x4750534452, TrainingEligibility: "synthetic-only; independent decoder and receiver validation required", ID: NewID(), CreatedAt: time.Now(), Kind: strings.ToUpper(request.Kind), Description: description,
		SampleRateHz: sampleRate, SampleCount: sampleCount, DurationSeconds: float64(sampleCount) / float64(sampleRate), SymbolRate: request.SymbolRate,
		PayloadUTF8: request.Payload, PayloadHex: hex.EncodeToString(payload), ExpectedSymbols: int(math.Ceil(float64(sampleCount) / float64(samplesPerSymbol))),
		OccupiedBandwidthHz: bandwidth, ConfiguredSNRDB: request.SNRDB, MeasuredEVMPercent: 100 * math.Sqrt(errorPower/maxFloat(referencePower, 1e-12)),
		FrequencyOffsetHz: request.FrequencyOffsetHz, DriftHzPerSecond: request.DriftHzPerSecond, IQGainError: request.IQGainError,
		IQPhaseErrorDeg: request.IQPhaseErrorDeg, DCOffset: request.DCOffset, ClipLevel: request.ClipLevel,
		Source: "deterministic synthetic fixture", DecodeStatus: "ground truth generated; no receiver decode has been claimed"}
	return output, manifest, nil
}

func fixtureDescription(kind string) string {
	return map[string]string{"cw": "continuous carrier", "am": "AM 1 kHz tone", "nfm": "narrow FM 1 kHz tone", "wfm": "wide FM 1 kHz tone", "ook": "on-off keyed packet", "2fsk": "binary FSK packet", "gfsk": "Gaussian-shaped binary FSK packet", "gmsk": "Gaussian-shaped MSK-like packet", "bpsk": "binary phase-shift keyed packet", "qpsk": "quadrature phase-shift keyed packet"}[kind]
}

func fixtureBandwidth(request SignalFixtureRequest) float64 {
	switch request.Kind {
	case "cw":
		return 1
	case "am":
		return request.ToneHz * 2
	case "nfm":
		return 2 * (2500 + request.ToneHz)
	case "wfm":
		return 2 * (75000 + request.ToneHz)
	case "ook", "bpsk":
		return request.SymbolRate * 2
	case "qpsk":
		return request.SymbolRate
	default:
		return request.SymbolRate * 2
	}
}

func saveSignalFixture(dataDirectory string, iq []byte, manifest SignalFixtureManifest) (SignalFixtureManifest, error) {
	directory := filepath.Join(dataDirectory, "Transmit", "fixtures")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return manifest, err
	}
	// IDs prevent simultaneous same-kind benchmarks from overwriting each other.
	base := fmt.Sprintf("%s-%s-%s", manifest.CreatedAt.UTC().Format("20060102T150405.000Z"), strings.ToLower(manifest.Kind), NewID())
	manifest.IQPath = filepath.Join(directory, base+".cs8")
	digest := sha256.Sum256(iq)
	manifest.SHA256 = hex.EncodeToString(digest[:])
	if err := os.WriteFile(manifest.IQPath, iq, 0o600); err != nil {
		return manifest, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifest, err
	}
	if err := os.WriteFile(filepath.Join(directory, base+".truth.json"), append(data, '\n'), 0o600); err != nil {
		return manifest, err
	}
	return manifest, nil
}
