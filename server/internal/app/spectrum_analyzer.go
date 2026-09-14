package app

import (
	"errors"
	"math"
	"sync"
	"time"
)

type SpectrumAnalyzerRequest struct {
	DeviceIDs    []string `json:"deviceIDs"`
	StartHz      float64  `json:"startHz"`
	EndHz        float64  `json:"endHz"`
	SampleRateHz int      `json:"sampleRateHz"`
	BinCount     int      `json:"binCount"`
}

type SpectrumAnalyzerStatus struct {
	Running   bool      `json:"running"`
	StartHz   float64   `json:"startHz"`
	EndHz     float64   `json:"endHz"`
	BinsDBFS  []float64 `json:"binsDBFS"`
	Sweeps    uint64    `json:"sweeps"`
	Slices    uint64    `json:"slices"`
	DeviceIDs []string  `json:"deviceIDs"`
	UpdatedAt time.Time `json:"updatedAt"`
	LastError string    `json:"lastError,omitempty"`
}

type spectrumAnalyzerState struct {
	mu sync.RWMutex
	SpectrumAnalyzerStatus
	stop chan struct{}
}

func (r *Runtime) SpectrumAnalyzerStatus() SpectrumAnalyzerStatus {
	if r.spectrumAnalyzer == nil {
		return SpectrumAnalyzerStatus{}
	}
	r.spectrumAnalyzer.mu.RLock()
	defer r.spectrumAnalyzer.mu.RUnlock()
	status := r.spectrumAnalyzer.SpectrumAnalyzerStatus
	status.BinsDBFS = append([]float64(nil), status.BinsDBFS...)
	status.DeviceIDs = append([]string(nil), status.DeviceIDs...)
	return status
}

func (r *Runtime) StartSpectrumAnalyzer(request SpectrumAnalyzerRequest) (SpectrumAnalyzerStatus, error) {
	if !(request.StartHz > 0) || request.EndHz <= request.StartHz {
		return SpectrumAnalyzerStatus{}, errors.New("a valid spectrum range is required")
	}
	devices := r.Devices()
	selected := make([]SDRDevice, 0, len(request.DeviceIDs))
	for _, id := range request.DeviceIDs {
		for _, device := range devices {
			if device.ID == id && device.Connected && device.Available {
				selected = append(selected, device)
				break
			}
		}
	}
	if len(selected) == 0 {
		return SpectrumAnalyzerStatus{}, errors.New("select at least one connected receiver")
	}
	if request.BinCount < 1024 {
		request.BinCount = 4096
	}
	if request.BinCount > 32768 {
		request.BinCount = 32768
	}
	r.StopSpectrumAnalyzer()
	state := &spectrumAnalyzerState{SpectrumAnalyzerStatus: SpectrumAnalyzerStatus{Running: true, StartHz: request.StartHz, EndHz: request.EndHz, DeviceIDs: append([]string(nil), request.DeviceIDs...), BinsDBFS: make([]float64, request.BinCount)}, stop: make(chan struct{})}
	for i := range state.BinsDBFS {
		state.BinsDBFS[i] = -120
	}
	r.mu.Lock()
	r.spectrumAnalyzer = state
	r.mu.Unlock()
	go r.runSpectrumAnalyzer(state, request, selected)
	return r.SpectrumAnalyzerStatus(), nil
}

func (r *Runtime) StopSpectrumAnalyzer() SpectrumAnalyzerStatus {
	r.mu.RLock()
	state := r.spectrumAnalyzer
	r.mu.RUnlock()
	if state == nil {
		return SpectrumAnalyzerStatus{}
	}
	state.mu.Lock()
	if state.Running {
		close(state.stop)
		state.Running = false
	}
	state.mu.Unlock()
	return r.SpectrumAnalyzerStatus()
}

func (r *Runtime) ClearSpectrumAnalyzer() SpectrumAnalyzerStatus {
	r.mu.RLock()
	state := r.spectrumAnalyzer
	r.mu.RUnlock()
	if state == nil {
		return SpectrumAnalyzerStatus{}
	}
	state.mu.Lock()
	for i := range state.BinsDBFS {
		state.BinsDBFS[i] = -120
	}
	state.Sweeps = 0
	state.Slices = 0
	state.UpdatedAt = time.Time{}
	state.LastError = ""
	state.mu.Unlock()
	return r.SpectrumAnalyzerStatus()
}

func (r *Runtime) runSpectrumAnalyzer(state *spectrumAnalyzerState, request SpectrumAnalyzerRequest, devices []SDRDevice) {
	for {
		state.mu.RLock()
		beforeSlices := state.Slices
		state.mu.RUnlock()
		var workers sync.WaitGroup
		for _, device := range devices {
			workers.Add(1)
			go func(device SDRDevice) { defer workers.Done(); r.runSpectrumAnalyzerPass(state, request, device) }(device)
		}
		workers.Wait()
		select {
		case <-state.stop:
			return
		default:
		}
		state.mu.Lock()
		state.Sweeps++
		capturedNothing := state.Slices == beforeSlices
		state.mu.Unlock()
		// Missing drivers and disconnected receivers must not turn the analyzer
		// into a tight retry loop that consumes an entire CPU core.
		if capturedNothing {
			select {
			case <-state.stop:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

func (r *Runtime) runSpectrumAnalyzerPass(state *spectrumAnalyzerState, request SpectrumAnalyzerRequest, device SDRDevice) {
	if device.Kind == "Simulator" {
		r.mergeSimulatedSpectrum(state, request, device)
		select {
		case <-state.stop:
		case <-time.After(16 * time.Millisecond):
		}
		return
	}
	span := request.EndHz - request.StartHz
	rate := request.SampleRateHz
	maximum := maximumCaptureRate(device)
	if rate <= 0 || rate > maximum {
		rate = maximum
	}
	if rate < 225000 {
		rate = 225000
	}
	usable := float64(rate) * .82
	sliceCount := max(1, int(math.Ceil(span/usable)))
	for slice := 0; slice < sliceCount; slice++ {
		select {
		case <-state.stop:
			return
		default:
		}
		center := request.StartHz + (float64(slice)+.5)*span/float64(sliceCount)
		if (device.FrequencyMinimumHz > 0 && center < device.FrequencyMinimumHz) || (device.FrequencyMaximumHz > 0 && center > device.FrequencyMaximumHz) {
			continue
		}
		spec := CaptureSpec{CenterFrequencyHz: int64(math.Round(center)), SampleRateHz: rate, GainDB: 20, AutoGain: true}
		data, format, err := captureWindow(device, spec, 35*time.Millisecond, state.stop)
		if err != nil {
			state.mu.Lock()
			state.LastError = err.Error()
			state.mu.Unlock()
			continue
		}
		snapshot, err := BuildSpectrumSnapshot(spec, data, format)
		if err != nil {
			continue
		}
		r.updateSpectrum(device.ID, spec, data, format)
		state.mu.Lock()
		for i, value := range snapshot.BinsDBFS {
			frequency := snapshot.StartFrequencyHz + (snapshot.EndFrequencyHz-snapshot.StartFrequencyHz)*float64(i)/float64(max(1, len(snapshot.BinsDBFS)-1))
			if frequency < request.StartHz || frequency > request.EndHz {
				continue
			}
			index := int((frequency - request.StartHz) / span * float64(len(state.BinsDBFS)-1))
			if index >= 0 && index < len(state.BinsDBFS) && value > state.BinsDBFS[index] {
				state.BinsDBFS[index] = value
			}
		}
		state.Slices++
		state.UpdatedAt = time.Now()
		state.LastError = ""
		state.mu.Unlock()
	}
}

func (r *Runtime) mergeSimulatedSpectrum(state *spectrumAnalyzerState, request SpectrumAnalyzerRequest, device SDRDevice) {
	span := request.EndHz - request.StartHz
	now := time.Now()
	state.mu.Lock()
	for index := range state.BinsDBFS {
		frequency := request.StartHz + span*float64(index)/float64(max(1, len(state.BinsDBFS)-1))
		phase := frequency/1.7e6 + float64(now.UnixMilli()%10000)/1900
		value := -96 + 3*math.Sin(phase)
		for _, fraction := range []float64{.18, .43, .71, .86} {
			distance := math.Abs(frequency-(request.StartHz+span*fraction)) / maxFloat(span, 1)
			value = maxFloat(value, -34-distance*1800)
		}
		state.BinsDBFS[index] = value
	}
	state.Slices++
	state.UpdatedAt = now
	state.LastError = ""
	state.mu.Unlock()
	_ = device
}
