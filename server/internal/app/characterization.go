package app

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CharacterizationRequest describes an ambient RF survey. It does not claim
// laboratory receiver sensitivity: that requires a calibrated signal source
// and known input level. The measurements remain useful for comparing antennas,
// gain settings, locations, and receiver coverage under repeatable conditions.
type CharacterizationRequest struct {
	DeviceIDs         []string `json:"deviceIDs"`
	Name              string   `json:"name"`
	AntennaLabel      string   `json:"antennaLabel"`
	RangeMode         string   `json:"rangeMode"`
	StartHz           float64  `json:"startHz"`
	EndHz             float64  `json:"endHz"`
	AntennaMinimumHz  float64  `json:"antennaMinimumHz,omitempty"`
	AntennaMaximumHz  float64  `json:"antennaMaximumHz,omitempty"`
	Points            int      `json:"points"`
	DwellMilliseconds int      `json:"dwellMilliseconds"`
	SampleRateHz      int      `json:"sampleRateHz"`
	UseCalibration    bool     `json:"useCalibration"`
	GainDB            float64  `json:"gainDB"`
	LNAGainDB         int      `json:"lnaGainDB"`
	VGAGainDB         int      `json:"vgaGainDB"`
	AmpEnabled        bool     `json:"ampEnabled"`
}

type CharacterizationPoint struct {
	FrequencyHz    float64   `json:"frequencyHz"`
	SignalDBFS     float64   `json:"signalDBFS"`
	NoiseDBFS      float64   `json:"noiseDBFS"`
	PeakDBFS       float64   `json:"peakDBFS"`
	SNRDB          float64   `json:"snrDB"`
	ClippedPercent float64   `json:"clippedPercent"`
	RelativeScore  float64   `json:"relativeScore"`
	SignalDetected bool      `json:"signalDetected"`
	CapturedAt     time.Time `json:"capturedAt"`
}

type DeviceCharacterization struct {
	DeviceID                string                  `json:"deviceID"`
	DeviceName              string                  `json:"deviceName"`
	DeviceKind              string                  `json:"deviceKind"`
	NominalMinimumHz        float64                 `json:"nominalMinimumHz,omitempty"`
	NominalMaximumHz        float64                 `json:"nominalMaximumHz,omitempty"`
	FrequencyRangeNote      string                  `json:"frequencyRangeNote,omitempty"`
	TestedMinimumHz         float64                 `json:"testedMinimumHz,omitempty"`
	TestedMaximumHz         float64                 `json:"testedMaximumHz,omitempty"`
	CurrentFrequencyHz      float64                 `json:"currentFrequencyHz,omitempty"`
	CompletedPoints         int                     `json:"completedPoints"`
	TotalPoints             int                     `json:"totalPoints"`
	BestObservedFrequencyHz float64                 `json:"bestObservedFrequencyHz,omitempty"`
	QuietestFrequencyHz     float64                 `json:"quietestFrequencyHz,omitempty"`
	StrongestFrequencyHz    float64                 `json:"strongestFrequencyHz,omitempty"`
	AverageNoiseDBFS        float64                 `json:"averageNoiseDBFS,omitempty"`
	AverageSNRDB            float64                 `json:"averageSNRDB,omitempty"`
	DetectedPoints          int                     `json:"detectedPoints"`
	OverloadedPoints        int                     `json:"overloadedPoints"`
	Recommendation          string                  `json:"recommendation,omitempty"`
	Error                   string                  `json:"error,omitempty"`
	Points                  []CharacterizationPoint `json:"points"`
}

type CharacterizationStatus struct {
	Running              bool                     `json:"running"`
	StartedAt            *time.Time               `json:"startedAt,omitempty"`
	CompletedAt          *time.Time               `json:"completedAt,omitempty"`
	ExpectedCompletionAt *time.Time               `json:"expectedCompletionAt,omitempty"`
	Request              CharacterizationRequest  `json:"request"`
	Results              []DeviceCharacterization `json:"results"`
	LastError            string                   `json:"lastError,omitempty"`
	MeasurementNotice    string                   `json:"measurementNotice"`
}

type CharacterizationManager struct {
	mu     sync.RWMutex
	path   string
	status CharacterizationStatus
	stop   chan struct{}
}

func NewCharacterizationManager(dataDirectory string) *CharacterizationManager {
	m := &CharacterizationManager{path: filepath.Join(dataDirectory, "Data", "receiver-characterization.json")}
	m.status.MeasurementNotice = characterizationNotice
	if data, err := os.ReadFile(m.path); err == nil {
		_ = json.Unmarshal(data, &m.status)
		m.status.Running = false
		m.status.ExpectedCompletionAt = nil
		m.status.MeasurementNotice = characterizationNotice
	}
	return m
}

const characterizationNotice = "Ambient results show observed response for this receiver, antenna, gain, place, and time. True sensitivity requires a calibrated RF signal source."

func (m *CharacterizationManager) Status() CharacterizationStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneCharacterizationStatus(m.status)
}

func cloneCharacterizationStatus(status CharacterizationStatus) CharacterizationStatus {
	copyStatus := status
	copyStatus.Request.DeviceIDs = append([]string(nil), status.Request.DeviceIDs...)
	copyStatus.Results = append([]DeviceCharacterization(nil), status.Results...)
	for index := range copyStatus.Results {
		copyStatus.Results[index].Points = append([]CharacterizationPoint(nil), status.Results[index].Points...)
	}
	return copyStatus
}

func validateCharacterizationRequest(request CharacterizationRequest) (CharacterizationRequest, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.AntennaLabel = strings.TrimSpace(request.AntennaLabel)
	request.RangeMode = strings.ToLower(strings.TrimSpace(request.RangeMode))
	if request.RangeMode == "" {
		request.RangeMode = "custom"
	}
	if request.RangeMode != "receiver" && request.RangeMode != "antenna" && request.RangeMode != "custom" {
		return request, errors.New("choose receiver range, antenna range, or custom range")
	}
	if len(request.DeviceIDs) == 0 {
		return request, errors.New("select at least one connected receiver")
	}
	if request.RangeMode == "antenna" {
		request.StartHz, request.EndHz = request.AntennaMinimumHz, request.AntennaMaximumHz
	}
	if request.RangeMode != "receiver" && (request.StartHz <= 0 || request.EndHz <= request.StartHz) {
		return request, errors.New("enter a valid characterization frequency range")
	}
	if request.Points == 0 {
		request.Points = 96
	}
	if request.Points < 8 || request.Points > 512 {
		return request, errors.New("characterization points must be between 8 and 512")
	}
	if request.DwellMilliseconds == 0 {
		request.DwellMilliseconds = 180
	}
	if request.DwellMilliseconds < 80 || request.DwellMilliseconds > 2000 {
		return request, errors.New("characterization dwell must be between 80 ms and 2 seconds")
	}
	if request.SampleRateHz != 0 && !supportedUserSampleRate(request.SampleRateHz) {
		return request, errors.New("choose Auto or a supported characterization sample rate")
	}
	if request.GainDB < 0 || request.GainDB > 62 {
		return request, errors.New("receiver gain must be between 0 and 62 dB")
	}
	if request.LNAGainDB < 0 || request.LNAGainDB > 40 || request.LNAGainDB%8 != 0 {
		return request, errors.New("HackRF LNA gain must be 0 to 40 dB in 8 dB steps")
	}
	if request.VGAGainDB < 0 || request.VGAGainDB > 62 || request.VGAGainDB%2 != 0 {
		return request, errors.New("HackRF VGA gain must be 0 to 62 dB in 2 dB steps")
	}
	return request, nil
}

func (m *CharacterizationManager) Start(devices []SDRDevice, request CharacterizationRequest, calibrations *CalibrationStore) (CharacterizationStatus, error) {
	request, err := validateCharacterizationRequest(request)
	if err != nil {
		return m.Status(), err
	}
	selected := make([]SDRDevice, 0, len(request.DeviceIDs))
	for _, id := range request.DeviceIDs {
		for _, device := range devices {
			if device.ID == id && device.Connected {
				selected = append(selected, device)
				break
			}
		}
	}
	if len(selected) != len(request.DeviceIDs) {
		return m.Status(), errors.New("one or more selected receivers are unavailable")
	}
	m.mu.Lock()
	if m.status.Running {
		m.mu.Unlock()
		return m.Status(), errors.New("a receiver characterization is already running")
	}
	now := time.Now()
	estimatedSeconds := float64(request.Points) * (float64(request.DwellMilliseconds)/1000 + .45)
	estimated := now.Add(time.Duration(estimatedSeconds * float64(time.Second)))
	results := make([]DeviceCharacterization, len(selected))
	for index, device := range selected {
		start, end := characterizationRange(device, request)
		results[index] = DeviceCharacterization{DeviceID: device.ID, DeviceName: device.Name, DeviceKind: device.Kind,
			NominalMinimumHz: device.FrequencyMinimumHz, NominalMaximumHz: device.FrequencyMaximumHz, FrequencyRangeNote: device.FrequencyRangeNote,
			TestedMinimumHz: start, TestedMaximumHz: end, TotalPoints: request.Points, Points: []CharacterizationPoint{}}
	}
	m.stop = make(chan struct{})
	m.status = CharacterizationStatus{Running: true, StartedAt: &now, ExpectedCompletionAt: &estimated, Request: request,
		Results: results, MeasurementNotice: characterizationNotice}
	stop := m.stop
	m.mu.Unlock()

	go m.run(selected, request, calibrations, stop)
	return m.Status(), nil
}

func characterizationRange(device SDRDevice, request CharacterizationRequest) (float64, float64) {
	start, end := request.StartHz, request.EndHz
	if request.RangeMode == "receiver" {
		start, end = device.FrequencyMinimumHz, device.FrequencyMaximumHz
	}
	if device.FrequencyMinimumHz > 0 && start < device.FrequencyMinimumHz {
		start = device.FrequencyMinimumHz
	}
	if device.FrequencyMaximumHz > 0 && (end == 0 || end > device.FrequencyMaximumHz) {
		end = device.FrequencyMaximumHz
	}
	return start, end
}

func (m *CharacterizationManager) run(devices []SDRDevice, request CharacterizationRequest, calibrations *CalibrationStore, stop <-chan struct{}) {
	var workers sync.WaitGroup
	for index, device := range devices {
		workers.Add(1)
		go func(resultIndex int, receiver SDRDevice) {
			defer workers.Done()
			m.runDevice(resultIndex, receiver, request, calibrations, stop)
		}(index, device)
	}
	workers.Wait()
	m.mu.Lock()
	now := time.Now()
	m.status.Running = false
	m.status.CompletedAt = &now
	m.status.ExpectedCompletionAt = nil
	m.stop = nil
	m.persistLocked()
	m.mu.Unlock()
}

func (m *CharacterizationManager) runDevice(resultIndex int, device SDRDevice, request CharacterizationRequest, calibrations *CalibrationStore, stop <-chan struct{}) {
	start, end := characterizationRange(device, request)
	if start <= 0 || end <= start {
		m.setDeviceError(resultIndex, "the selected range does not overlap this receiver's nominal tuning range")
		return
	}
	frequencies := spacedFrequencies(start, end, request.Points)
	for _, frequency := range frequencies {
		select {
		case <-stop:
			m.setDeviceError(resultIndex, "stopped")
			return
		default:
		}
		m.mu.Lock()
		m.status.Results[resultIndex].CurrentFrequencyHz = frequency
		m.mu.Unlock()
		point, err := characterizeFrequency(device, frequency, request, calibrations, stop)
		if err != nil {
			m.setDeviceError(resultIndex, err.Error())
			return
		}
		m.mu.Lock()
		result := &m.status.Results[resultIndex]
		result.Points = append(result.Points, point)
		result.CompletedPoints = len(result.Points)
		updateCharacterizationSummary(result)
		m.mu.Unlock()
	}
	m.mu.Lock()
	m.status.Results[resultIndex].CurrentFrequencyHz = 0
	m.persistLocked()
	m.mu.Unlock()
}

func spacedFrequencies(start, end float64, points int) []float64 {
	if points <= 1 {
		return []float64{(start + end) / 2}
	}
	values := make([]float64, points)
	step := (end - start) / float64(points-1)
	for index := range values {
		values[index] = start + float64(index)*step
	}
	return values
}

func characterizeFrequency(device SDRDevice, frequency float64, request CharacterizationRequest, calibrations *CalibrationStore, stop <-chan struct{}) (CharacterizationPoint, error) {
	if device.Kind == "Simulator" {
		return simulatedCharacterizationPoint(device.ID, frequency, request), nil
	}
	rate := request.SampleRateHz
	if rate == 0 {
		if device.Kind == "HackRF" {
			rate = 10_000_000
		} else {
			rate = 2_400_000
		}
	}
	if device.SampleRateLimit != nil && float64(rate) > *device.SampleRateLimit {
		rate = int(*device.SampleRateLimit)
	}
	if device.Kind == "HackRF" && rate < 2_000_000 {
		rate = 2_000_000
	}
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(frequency)), SampleRateHz: rate, GainDB: request.GainDB,
		LNAGainDB: request.LNAGainDB, VGAGainDB: request.VGAGainDB, AmpEnabled: request.AmpEnabled}
	if request.UseCalibration && calibrations != nil {
		if calibration, ok := calibrations.Get(device.ID); ok {
			spec.PPMCorrection = calibration.PPMCorrection
			spec.LNAGainDB = calibration.LNAGainDB
			spec.VGAGainDB = calibration.VGAGainDB
			spec.AmpEnabled = calibration.AmpEnabled
		}
	}
	if device.Kind == "HackRF" && spec.LNAGainDB == 0 && spec.VGAGainDB == 0 {
		spec.LNAGainDB, spec.VGAGainDB = 16, 16
	}
	if device.Kind != "HackRF" && spec.GainDB == 0 {
		spec.GainDB = 20
	}
	data, format, err := captureWindow(device, spec, time.Duration(request.DwellMilliseconds)*time.Millisecond, stop)
	if err != nil {
		return CharacterizationPoint{}, err
	}
	format = DetectSampleFormat(data, format)
	snapshot, err := BuildSpectrumSnapshot(spec, data, format)
	if err != nil {
		return CharacterizationPoint{}, err
	}
	return summarizeCharacterizationSpectrum(frequency, snapshot.BinsDBFS, clippedIQPercent(data, format)), nil
}

func summarizeCharacterizationSpectrum(frequency float64, bins []float64, clipped float64) CharacterizationPoint {
	usable := make([]float64, 0, len(bins))
	middle := len(bins) / 2
	dcGuard := maxInt(2, len(bins)/100)
	for index, value := range bins {
		if absInt(index-middle) <= dcGuard || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		usable = append(usable, value)
	}
	if len(usable) == 0 {
		usable = []float64{-150}
	}
	sort.Float64s(usable)
	noise := percentile(usable, .25)
	signal := percentile(usable, .95)
	peak := usable[len(usable)-1]
	snr := math.Max(0, signal-noise)
	score := clamp((snr-2)/20, 0, 1) * 100
	if clipped >= .5 {
		score *= .25
	}
	return CharacterizationPoint{FrequencyHz: frequency, SignalDBFS: signal, NoiseDBFS: noise, PeakDBFS: peak, SNRDB: snr,
		ClippedPercent: clipped, RelativeScore: score, SignalDetected: snr >= 7 && clipped < .5, CapturedAt: time.Now()}
}

func percentile(sorted []float64, position float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Round(clamp(position, 0, 1) * float64(len(sorted)-1)))
	return sorted[index]
}

func simulatedCharacterizationPoint(deviceID string, frequency float64, request CharacterizationRequest) CharacterizationPoint {
	phase := frequency/53e6 + float64(len(deviceID))*0.7
	noise := -102 + 7*math.Sin(phase*.37)
	snr := math.Max(0, 5+14*math.Sin(phase)*math.Sin(phase)+4*math.Cos(phase*.23))
	clipped := 0.0
	if request.AmpEnabled && math.Sin(phase*.13) > .92 {
		clipped = 1.1
	}
	return CharacterizationPoint{FrequencyHz: frequency, SignalDBFS: noise + snr, NoiseDBFS: noise, PeakDBFS: noise + snr + 4,
		SNRDB: snr, ClippedPercent: clipped, RelativeScore: clamp((snr-2)/20, 0, 1) * 100,
		SignalDetected: snr >= 7 && clipped < .5, CapturedAt: time.Now()}
}

func updateCharacterizationSummary(result *DeviceCharacterization) {
	if len(result.Points) == 0 {
		return
	}
	best, quietest, strongest := result.Points[0], result.Points[0], result.Points[0]
	noiseTotal, snrTotal := 0.0, 0.0
	result.DetectedPoints, result.OverloadedPoints = 0, 0
	for _, point := range result.Points {
		noiseTotal += point.NoiseDBFS
		snrTotal += point.SNRDB
		if point.RelativeScore > best.RelativeScore {
			best = point
		}
		if point.NoiseDBFS < quietest.NoiseDBFS {
			quietest = point
		}
		if point.PeakDBFS > strongest.PeakDBFS {
			strongest = point
		}
		if point.SignalDetected {
			result.DetectedPoints++
		}
		if point.ClippedPercent >= .5 {
			result.OverloadedPoints++
		}
	}
	result.BestObservedFrequencyHz = best.FrequencyHz
	result.QuietestFrequencyHz = quietest.FrequencyHz
	result.StrongestFrequencyHz = strongest.FrequencyHz
	result.AverageNoiseDBFS = noiseTotal / float64(len(result.Points))
	result.AverageSNRDB = snrTotal / float64(len(result.Points))
	switch {
	case result.OverloadedPoints > 0:
		result.Recommendation = "Overload was observed. Repeat with the RF amplifier off or lower gain before comparing antennas."
	case result.DetectedPoints == 0:
		result.Recommendation = "No clear ambient signals were observed. Use a narrower band with known local carriers or a calibrated signal source."
	default:
		result.Recommendation = fmt.Sprintf("Best observed response was near %.3f MHz. Repeat with identical settings before comparing another antenna.", best.FrequencyHz/1e6)
	}
}

func (m *CharacterizationManager) setDeviceError(index int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index >= 0 && index < len(m.status.Results) {
		m.status.Results[index].Error = message
		m.status.Results[index].CurrentFrequencyHz = 0
	}
}

func (m *CharacterizationManager) Stop() CharacterizationStatus {
	m.mu.Lock()
	if m.status.Running && m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.mu.Unlock()
	return m.Status()
}

func (m *CharacterizationManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.Running {
		return errors.New("stop the characterization before clearing its results")
	}
	m.status = CharacterizationStatus{MeasurementNotice: characterizationNotice, Results: []DeviceCharacterization{}}
	if err := os.Remove(m.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *CharacterizationManager) persistLocked() {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		m.status.LastError = err.Error()
		return
	}
	data, err := json.MarshalIndent(m.status, "", "  ")
	if err != nil {
		m.status.LastError = err.Error()
		return
	}
	if err := os.WriteFile(m.path, data, 0o600); err != nil {
		m.status.LastError = err.Error()
	}
}

func (m *CharacterizationManager) CSV() ([]byte, error) {
	status := m.Status()
	var output strings.Builder
	writer := csv.NewWriter(&output)
	_ = writer.Write([]string{"Receiver", "Kind", "Antenna", "Frequency Hz", "Signal dBFS", "Noise dBFS", "SNR dB", "Peak dBFS", "Clipped %", "Relative score", "Detected", "Captured at"})
	for _, result := range status.Results {
		for _, point := range result.Points {
			_ = writer.Write([]string{result.DeviceName, result.DeviceKind, status.Request.AntennaLabel,
				strconv.FormatFloat(point.FrequencyHz, 'f', 0, 64), strconv.FormatFloat(point.SignalDBFS, 'f', 2, 64),
				strconv.FormatFloat(point.NoiseDBFS, 'f', 2, 64), strconv.FormatFloat(point.SNRDB, 'f', 2, 64),
				strconv.FormatFloat(point.PeakDBFS, 'f', 2, 64), strconv.FormatFloat(point.ClippedPercent, 'f', 3, 64),
				strconv.FormatFloat(point.RelativeScore, 'f', 1, 64), strconv.FormatBool(point.SignalDetected), point.CapturedAt.Format(time.RFC3339)})
		}
	}
	writer.Flush()
	return []byte(output.String()), writer.Error()
}
