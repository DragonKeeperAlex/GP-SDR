package app

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type CalibrationStore struct {
	mu    sync.RWMutex
	path  string
	items map[string]DeviceCalibration
}

func NewCalibrationStore(path string) (*CalibrationStore, error) {
	store := &CalibrationStore{path: path, items: make(map[string]DeviceCalibration)}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &store.items); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *CalibrationStore) All() []DeviceCalibration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]DeviceCalibration, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DeviceID < items[j].DeviceID })
	return items
}

func (s *CalibrationStore) Get(deviceID string) (DeviceCalibration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[deviceID]
	return item, ok
}

func (s *CalibrationStore) Save(item DeviceCalibration) error {
	if item.DeviceID == "" || item.IQGain < .5 || item.IQGain > 1.5 || item.IQPhase < -20 || item.IQPhase > 20 || item.PPMCorrection < -200 || item.PPMCorrection > 200 {
		return errors.New("calibration values are outside the supported range")
	}
	s.mu.Lock()
	s.items[item.DeviceID] = item
	err := s.writeLocked()
	s.mu.Unlock()
	return err
}

func (s *CalibrationStore) Delete(deviceID string) error {
	s.mu.Lock()
	delete(s.items, deviceID)
	err := s.writeLocked()
	s.mu.Unlock()
	return err
}

func (s *CalibrationStore) writeLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}

func (r *Runtime) Calibrations() []DeviceCalibration { return r.calibrations.All() }

func (r *Runtime) SaveCalibration(item DeviceCalibration) (DeviceCalibration, error) {
	if item.MeasuredAt.IsZero() {
		item.MeasuredAt = time.Now()
	}
	if item.Source == "" {
		item.Source = "manual"
	}
	if item.IQGain == 0 {
		item.IQGain = 1
	}
	if err := r.calibrations.Save(item); err != nil {
		return DeviceCalibration{}, err
	}
	r.attachCalibrations()
	return item, nil
}

func (r *Runtime) DeleteCalibration(deviceID string) error {
	if err := r.calibrations.Delete(deviceID); err != nil {
		return err
	}
	r.attachCalibrations()
	return nil
}

func (r *Runtime) AutoCalibrate(request CalibrationRequest) (DeviceCalibration, error) {
	if request.ReferenceHz <= 0 {
		return DeviceCalibration{}, errors.New("enter a known active reference frequency")
	}
	r.mu.RLock()
	busy := r.running || len(r.mapperJobs) > 0
	var device *SDRDevice
	for index := range r.devices {
		if r.devices[index].ID == request.DeviceID && r.devices[index].Connected {
			copy := r.devices[index]
			device = &copy
			break
		}
	}
	r.mu.RUnlock()
	if busy || r.characterization != nil && r.characterization.Status().Running {
		return DeviceCalibration{}, errors.New("stop Tuner, scanning, P25, Mapper, and antenna tests before auto calibration")
	}
	if device == nil {
		return DeviceCalibration{}, errors.New("the selected receiver is not connected")
	}
	rate := request.SampleRateHz
	if rate == 0 {
		rate = 2_400_000
	}
	if device.Kind == "HackRF" && rate < 2_000_000 {
		rate = 2_000_000
	}
	spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(request.ReferenceHz)), SampleRateHz: rate,
		LNAGainDB: request.LNAGainDB, VGAGainDB: request.VGAGainDB}
	stream, err := StartIQStream(*device, spec)
	if err != nil {
		return DeviceCalibration{}, err
	}
	defer stream.Close()
	data := make([]byte, rate*2)
	if _, err := io.ReadFull(stream.Reader, data); err != nil {
		return DeviceCalibration{}, err
	}
	format := DetectSampleFormat(data, stream.Format)
	calibration, err := analyzeCalibration(data, format, rate, request.ReferenceHz)
	if err != nil {
		return DeviceCalibration{}, err
	}
	calibration.DeviceID = device.ID
	calibration.DeviceKind = device.Kind
	calibration.LNAGainDB = request.LNAGainDB
	calibration.VGAGainDB = request.VGAGainDB
	if device.Serial != nil {
		calibration.Serial = *device.Serial
	}
	if err := r.calibrations.Save(calibration); err != nil {
		return DeviceCalibration{}, err
	}
	r.attachCalibrations()
	return calibration, nil
}

func analyzeCalibration(data []byte, format SampleFormat, sampleRate int, referenceHz float64) (DeviceCalibration, error) {
	if len(data) < 8192 || sampleRate <= 0 {
		return DeviceCalibration{}, errors.New("calibration capture is too short")
	}
	count := len(data) / 2
	meanI, meanQ := 0.0, 0.0
	for n := 0; n < count; n++ {
		i, q := iqSample(data[n*2], data[n*2+1], format)
		meanI += i
		meanQ += q
	}
	meanI /= float64(count)
	meanQ /= float64(count)
	varI, varQ, covariance := 0.0, 0.0, 0.0
	for n := 0; n < count; n++ {
		i, q := iqSample(data[n*2], data[n*2+1], format)
		i, q = i-meanI, q-meanQ
		varI += i * i
		varQ += q * q
		covariance += i * q
	}
	varI /= float64(count)
	varQ /= float64(count)
	covariance /= float64(count)
	phase := -math.Atan2(covariance, math.Max(varI, 1e-12))
	c, s := math.Cos(phase), math.Sin(phase)
	correctedQVariance := varQ*c*c + varI*s*s + 2*covariance*c*s
	qGain := math.Sqrt(varI / math.Max(correctedQVariance, 1e-12))
	qGain = clamp(qGain, .5, 1.5)
	phaseDegrees := clamp(phase*180/math.Pi, -20, 20)

	offsetHz, snr := calibrationCarrierOffset(data, format, sampleRate, meanI, meanQ)
	if snr < 6 {
		return DeviceCalibration{}, errors.New("reference signal is too weak for a reliable calibration; choose a stronger known carrier")
	}
	ppm := int(math.Round(offsetHz / referenceHz * 1e6))
	// Modulated broadcast signals are suitable for DC and I/Q balance but their
	// short-term spectral centroid is not a trustworthy frequency standard.
	if snr < 12 {
		ppm = 0
	}
	if ppm < -200 {
		ppm = -200
	} else if ppm > 200 {
		ppm = 200
	}
	confidence := clamp((snr-3)/17, 0, 1)
	return DeviceCalibration{ReferenceHz: referenceHz, PPMCorrection: ppm, IQGain: qGain, IQPhase: phaseDegrees,
		DCRemoval: true, Confidence: confidence, SignalToNoiseDB: snr, MeasuredAt: time.Now(), Source: "automatic"}, nil
}

func calibrationCarrierOffset(data []byte, format SampleFormat, sampleRate int, meanI, meanQ float64) (float64, float64) {
	const size = 8192
	values := make([]complex128, size)
	for index := range values {
		i, q := iqSample(data[index*2], data[index*2+1], format)
		window := .5 - .5*math.Cos(2*math.Pi*float64(index)/float64(size-1))
		values[index] = complex((i-meanI)*window, (q-meanQ)*window)
	}
	fftInPlace(values)
	binWidth := float64(sampleRate) / size
	halfBins := int(math.Min(100_000/binWidth, float64(size/2-1)))
	noiseTotal, noiseCount := 0.0, 0
	for delta := -halfBins * 3; delta <= halfBins*3; delta++ {
		if absInt(delta) <= halfBins {
			continue
		}
		power := cmplx.Abs(values[wrappedBin(delta, size)])
		noiseTotal += power * power
		noiseCount++
	}
	noise := noiseTotal / float64(maxInt(noiseCount, 1))
	weighted, total, signalPower := 0.0, 0.0, 0.0
	for delta := -halfBins; delta <= halfBins; delta++ {
		if absInt(delta) <= 1 {
			continue
		}
		magnitude := cmplx.Abs(values[wrappedBin(delta, size)])
		power := math.Max(magnitude*magnitude-noise, 0)
		weighted += float64(delta) * binWidth * power
		total += power
		signalPower += magnitude * magnitude
	}
	offset := 0.0
	if total > 0 {
		offset = weighted / total
	}
	snr := 10 * math.Log10(math.Max(signalPower/float64(halfBins*2)/math.Max(noise, 1e-15), 1e-15))
	return offset, snr
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
