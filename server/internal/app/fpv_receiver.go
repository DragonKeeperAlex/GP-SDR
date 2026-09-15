package app

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type FPVReceiverRequest struct {
	DeviceID     string  `json:"deviceID"`
	FrequencyHz  float64 `json:"frequencyHz"`
	SampleRateHz int     `json:"sampleRateHz"`
	Standard     string  `json:"standard"`
	GainDB       float64 `json:"gainDB"`
	LNAGainDB    float64 `json:"lnaGainDB"`
	VGAGainDB    float64 `json:"vgaGainDB"`
	AmpEnabled   bool    `json:"ampEnabled"`
	DCRemoval    bool    `json:"dcRemoval"`
}

type FPVReceiverStatus struct {
	Running      bool      `json:"running"`
	DeviceID     string    `json:"deviceID,omitempty"`
	FrequencyHz  float64   `json:"frequencyHz,omitempty"`
	Standard     string    `json:"standard,omitempty"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	FrameUpdated time.Time `json:"frameUpdated,omitempty"`
	FrameBytes   int64     `json:"frameBytes,omitempty"`
	BackendReady bool      `json:"backendReady"`
	LastError    string    `json:"lastError,omitempty"`
}

type fpvReceiverState struct {
	mu        sync.RWMutex
	status    FPVReceiverStatus
	command   *exec.Cmd
	framePath string
	stop      chan struct{}
}

func newFPVReceiverState(dataDirectory string) *fpvReceiverState {
	return &fpvReceiverState{framePath: filepath.Join(dataDirectory, "Cache", "fpv-live.png")}
}

func fpvBackend() (python, script, runtimeRoot string, ok bool) {
	if runtime.GOOS != "darwin" {
		return "", "", "", false
	}
	root := "/Applications/FPV Viewer.app/Contents/Resources"
	python, script = filepath.Join(root, "venv", "bin", "python"), filepath.Join(root, "fpv-sdr", "fpv_viewer.py")
	if _, err := os.Stat(python); err != nil {
		return "", "", "", false
	}
	if _, err := os.Stat(script); err != nil {
		return "", "", "", false
	}
	return python, script, root, true
}

func (r *Runtime) FPVStatus() FPVReceiverStatus {
	if r.fpv == nil {
		return FPVReceiverStatus{}
	}
	r.fpv.mu.Lock()
	defer r.fpv.mu.Unlock()
	_, _, _, r.fpv.status.BackendReady = fpvBackend()
	if info, err := os.Stat(r.fpv.framePath); err == nil {
		r.fpv.status.FrameUpdated, r.fpv.status.FrameBytes = info.ModTime(), info.Size()
	}
	return r.fpv.status
}

func (r *Runtime) StartFPV(request FPVReceiverRequest) (FPVReceiverStatus, error) {
	if r.fpv == nil {
		return FPVReceiverStatus{}, errors.New("FPV receiver is unavailable")
	}
	var device SDRDevice
	for _, candidate := range r.Devices() {
		if candidate.ID == request.DeviceID && candidate.Connected && candidate.Available {
			device = candidate
			break
		}
	}
	if device.ID == "" {
		return r.FPVStatus(), errors.New("select a connected receiver")
	}
	if request.FrequencyHz < 1e6 || request.FrequencyHz > 6e9 {
		return r.FPVStatus(), errors.New("enter a valid frequency supported by the receiver")
	}
	if (device.FrequencyMinimumHz > 0 && request.FrequencyHz < device.FrequencyMinimumHz) || (device.FrequencyMaximumHz > 0 && request.FrequencyHz > device.FrequencyMaximumHz) {
		return r.FPVStatus(), fmt.Errorf("%s cannot tune %.3f MHz", device.Name, request.FrequencyHz/1e6)
	}
	python, script, runtimeRoot, ok := fpvBackend()
	if !ok {
		return r.FPVStatus(), errors.New("FPV video backend is not installed; install FPV Viewer first")
	}
	r.StopFPV()
	if request.SampleRateHz <= 0 {
		request.SampleRateHz = 16_000_000
	}
	request.SampleRateHz = min(request.SampleRateHz, maximumCaptureRate(device))
	request.Standard = strings.ToLower(strings.TrimSpace(request.Standard))
	if request.Standard != "pal" {
		request.Standard = "ntsc"
	}
	if err := os.MkdirAll(filepath.Dir(r.fpv.framePath), 0o755); err != nil {
		return r.FPVStatus(), err
	}
	_ = os.Remove(r.fpv.framePath)
	driver := strings.ToLower(device.Kind)
	if strings.HasPrefix(device.Driver, "SoapySDR:") {
		driver = strings.TrimPrefix(device.Driver, "SoapySDR:")
	}
	switch device.Kind {
	case "HackRF":
		driver = "hackrf"
	case "PlutoSDR":
		driver = "pluto"
	case "RTL-SDR":
		driver = "rtlsdr"
	}
	if device.Kind == "RTL-TCP" || device.Driver == "Android USB" {
		return r.FPVStatus(), errors.New("this remote receiver does not expose the continuous local IQ source required by the FPV decoder")
	}
	args := []string{script, "--sdr", driver, "--freq", fmt.Sprintf("%.0f", request.FrequencyHz), "--samp-rate", fmt.Sprintf("%d", request.SampleRateHz), "--gain", fmt.Sprintf("%.1f", request.GainDB), "--standard", request.Standard, "--no-keys", "--no-window", "--frame-out", r.fpv.framePath}
	if device.Kind == "HackRF" {
		args = append(args, "--lna", fmt.Sprintf("%.0f", request.LNAGainDB), "--vga", fmt.Sprintf("%.0f", request.VGAGainDB))
		if request.AmpEnabled {
			args = append(args, "--amp")
		}
	}
	if !request.DCRemoval {
		args = append(args, "--no-dcblock")
	}
	cmd := exec.Command(python, args...)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(runtimeRoot, "runtime", "python"), "DYLD_LIBRARY_PATH="+filepath.Join(runtimeRoot, "runtime", "lib")+":/opt/homebrew/lib")
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		return r.FPVStatus(), err
	}
	r.fpv.mu.Lock()
	r.fpv.command = cmd
	r.fpv.stop = make(chan struct{})
	r.fpv.status = FPVReceiverStatus{Running: true, DeviceID: device.ID, FrequencyHz: request.FrequencyHz, Standard: request.Standard, StartedAt: time.Now(), BackendReady: true}
	stop := r.fpv.stop
	r.fpv.mu.Unlock()
	go func() {
		err := cmd.Wait()
		r.fpv.mu.Lock()
		defer r.fpv.mu.Unlock()
		if r.fpv.command != cmd {
			return
		}
		r.fpv.status.Running = false
		select {
		case <-stop:
		default:
			message := strings.TrimSpace(output.String())
			if len(message) > 1200 {
				message = message[len(message)-1200:]
			}
			if message == "" && err != nil {
				message = err.Error()
			}
			r.fpv.status.LastError = message
		}
	}()
	return r.FPVStatus(), nil
}

func (r *Runtime) StopFPV() FPVReceiverStatus {
	if r.fpv == nil {
		return FPVReceiverStatus{}
	}
	r.fpv.mu.Lock()
	cmd, stop := r.fpv.command, r.fpv.stop
	r.fpv.command = nil
	r.fpv.status.Running = false
	if stop != nil {
		close(stop)
		r.fpv.stop = nil
	}
	r.fpv.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
		time.AfterFunc(2*time.Second, func() { _ = cmd.Process.Kill() })
	}
	return r.FPVStatus()
}

func (r *Runtime) ServeFPVFrame(w http.ResponseWriter, request *http.Request) {
	if r.fpv == nil {
		http.NotFound(w, request)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, request, r.fpv.framePath)
}
