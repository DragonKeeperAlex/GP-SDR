package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TransmitRequest is intentionally small and conservative. GP-SDR only
// exposes analog AM/NFM/WFM file playback for now; digital signalling and
// unattended/continuous transmission remain receive-side features.
type TransmitRequest struct {
	DeviceID       string                `json:"deviceID"`
	FrequencyHz    float64               `json:"frequencyHz"`
	Mode           string                `json:"mode"`
	AudioPath      string                `json:"audioPath"`
	DurationSecond float64               `json:"durationSeconds"`
	TXGainDB       int                   `json:"txGainDB"`
	Armed          bool                  `json:"armed"`
	DryRun         bool                  `json:"dryRun"`
	Fixture        *SignalFixtureRequest `json:"fixture,omitempty"`
}

type TransmitStatus struct {
	State       string                 `json:"state"`
	Note        string                 `json:"note"`
	DeviceID    string                 `json:"deviceID,omitempty"`
	FrequencyHz float64                `json:"frequencyHz,omitempty"`
	Mode        string                 `json:"mode,omitempty"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	FinishedAt  *time.Time             `json:"finishedAt,omitempty"`
	IQPath      string                 `json:"iqPath,omitempty"`
	Samples     int                    `json:"samples,omitempty"`
	DryRun      bool                   `json:"dryRun,omitempty"`
	Fixture     *SignalFixtureManifest `json:"fixture,omitempty"`
}

type transmitState struct {
	generation sync.Mutex
	mu         sync.Mutex
	status     TransmitStatus
	cancel     context.CancelFunc
}

func newTransmitState() *transmitState {
	return &transmitState{status: TransmitStatus{State: "idle", Note: "Transmit is receive-only until an armed transmit-capable SDR job is started."}}
}

func (r *Runtime) TransmitStatus() TransmitStatus {
	if r.transmit == nil {
		return TransmitStatus{State: "idle", Note: "Transmit is unavailable."}
	}
	r.transmit.mu.Lock()
	defer r.transmit.mu.Unlock()
	return r.transmit.status
}

// SaveTransmitAudio stores a user-selected WAV in the application data area.
// The HTTP handler uses this instead of accepting arbitrary paths from remote
// clients, keeping the transmit surface local and auditable.
func (r *Runtime) SaveTransmitAudio(name string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("the audio file is empty")
	}
	if len(data) > 50<<20 {
		return "", errors.New("audio files are limited to 50 MB")
	}
	if strings.ToLower(filepath.Ext(name)) != ".wav" {
		return "", errors.New("choose a PCM WAV file")
	}
	if _, _, err := decodePCM16WAV(data); err != nil {
		return "", fmt.Errorf("choose a valid 16-bit mono or stereo PCM WAV: %w", err)
	}
	directory := filepath.Join(r.dataDirectory, "Transmit", "audio")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, NewID()+".wav")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (r *Runtime) Transmit(request TransmitRequest) (TransmitStatus, error) {
	if r.transmit == nil {
		r.transmit = newTransmitState()
	}
	if !r.transmit.generation.TryLock() {
		return r.TransmitStatus(), errors.New("a transmit waveform is already being prepared")
	}
	defer r.transmit.generation.Unlock()
	r.transmit.mu.Lock()
	alreadyRunning := r.transmit.cancel != nil
	r.transmit.mu.Unlock()
	if alreadyRunning {
		return r.TransmitStatus(), errors.New("a transmit job is already running")
	}
	request.Mode = strings.ToLower(strings.TrimSpace(request.Mode))
	if request.Mode == "fm" {
		request.Mode = "nfm"
	}
	if request.Mode != "am" && request.Mode != "nfm" && request.Mode != "wfm" {
		return r.TransmitStatus(), errors.New("Transmit supports AM, NFM, and WFM audio only.")
	}
	if !isFinitePositive(request.FrequencyHz) || request.FrequencyHz < 1e6 || request.FrequencyHz > 6e9 {
		return r.TransmitStatus(), errors.New("Enter a frequency from 1 MHz to 6 GHz.")
	}
	if math.IsNaN(request.DurationSecond) || math.IsInf(request.DurationSecond, 0) {
		return r.TransmitStatus(), errors.New("transmit duration must be finite")
	}
	if request.DurationSecond <= 0 {
		request.DurationSecond = 5
	}
	if request.DurationSecond > 60 {
		return r.TransmitStatus(), errors.New("Transmit duration is limited to 60 seconds per job.")
	}
	if request.TXGainDB < 0 || request.TXGainDB > 89 {
		return r.TransmitStatus(), errors.New("TX gain must be between 0 and 89 dB.")
	}
	if !request.DryRun && !request.Armed {
		return r.TransmitStatus(), errors.New("Check the local RF safety confirmation before transmitting.")
	}
	r.mu.RLock()
	var device SDRDevice
	for _, candidate := range r.devices {
		if candidate.ID == strings.TrimSpace(request.DeviceID) {
			device = candidate
			break
		}
	}
	busy := r.running && r.profileUsesDeviceLocked(device.ID)
	mapperOwner := ""
	for id, job := range r.mapperJobs {
		if job.deviceID == device.ID {
			mapperOwner = id
			break
		}
	}
	r.mu.RUnlock()
	offlineFixture := request.DryRun && request.Fixture != nil
	transmitCapable := strings.EqualFold(device.Kind, "HackRF") || (strings.EqualFold(device.Kind, "PlutoSDR") && device.TransmitChannels > 0)
	if !offlineFixture && (device.ID == "" || !device.Connected || !device.Available || !transmitCapable) {
		return r.TransmitStatus(), errors.New("select a connected HackRF or PlutoSDR with an exposed TX channel; RTL-SDR cannot transmit")
	}
	if strings.EqualFold(device.Kind, "HackRF") && request.TXGainDB > 47 {
		return r.TransmitStatus(), errors.New("HackRF TX gain must be between 0 and 47 dB")
	}
	if (device.HealthWarning != "" || device.FirmwareSelfTestWarning) && !request.DryRun {
		return r.TransmitStatus(), errors.New("resolve the HackRF diagnostic warning before RF transmission; receive and transmit dry runs remain available")
	}
	if busy {
		return r.TransmitStatus(), fmt.Errorf("%s is already in use by Live, Tuner, or P25", device.Name)
	}
	if mapperOwner != "" && !request.DryRun {
		return r.TransmitStatus(), fmt.Errorf("%s is collecting data for Mapper job %s; choose a separate transmitter or stop that job", device.Name, mapperOwner)
	}
	if request.AudioPath == "" && request.Fixture == nil {
		return r.TransmitStatus(), errors.New("choose a WAV audio file")
	}
	// Generate a bounded IQ file before starting hardware. The dry-run path is
	// useful for validating modulation and file size without RF output.
	transmitRate := 2_000_000
	samples := int(math.Round(request.DurationSecond * float64(transmitRate)))
	if samples < 1 {
		samples = transmitRate
	}
	var iq []byte
	var fixture *SignalFixtureManifest
	if request.Fixture != nil {
		generated, manifest, fixtureErr := generateSignalFixture(*request.Fixture, transmitRate, samples)
		if fixtureErr != nil {
			return r.TransmitStatus(), fixtureErr
		}
		manifest, fixtureErr = saveSignalFixture(r.dataDirectory, generated, manifest)
		if fixtureErr != nil {
			return r.TransmitStatus(), fixtureErr
		}
		iq, fixture = generated, &manifest
	} else {
		pcm, audioRate, readErr := readPCM16WAV(request.AudioPath)
		if readErr != nil {
			return r.TransmitStatus(), fmt.Errorf("read audio: %w", readErr)
		}
		if len(pcm) == 0 || audioRate <= 0 {
			return r.TransmitStatus(), errors.New("the WAV contains no PCM audio")
		}
		var modulationErr error
		iq, modulationErr = modulateTransmitAudio(pcm, audioRate, transmitRate, samples, request.Mode)
		if modulationErr != nil {
			return r.TransmitStatus(), modulationErr
		}
	}
	iqPath := ""
	if fixture != nil {
		iqPath = fixture.IQPath
	} else {
		directory := filepath.Join(r.dataDirectory, "Transmit", "iq")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return r.TransmitStatus(), err
		}
		iqPath = filepath.Join(directory, fmt.Sprintf("%s-%d.cs8", time.Now().UTC().Format("20060102T150405.000Z"), int64(request.FrequencyHz)))
		if err := os.WriteFile(iqPath, iq, 0o600); err != nil {
			return r.TransmitStatus(), err
		}
	}
	now := time.Now()
	status := TransmitStatus{State: "running", Note: "IQ generated; waiting for SDR transmitter.", DeviceID: device.ID, FrequencyHz: request.FrequencyHz, Mode: request.Mode, StartedAt: &now, IQPath: iqPath, Samples: samples, DryRun: request.DryRun, Fixture: fixture}
	r.transmit.mu.Lock()
	if r.transmit.cancel != nil {
		r.transmit.mu.Unlock()
		return r.TransmitStatus(), errors.New("a transmit job is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.transmit.cancel = cancel
	r.transmit.status = status
	r.transmit.mu.Unlock()
	go r.runTransmit(ctx, request, device, iqPath, status)
	return status, nil
}

func (r *Runtime) runTransmit(ctx context.Context, request TransmitRequest, device SDRDevice, iqPath string, started TransmitStatus) {
	var runErr error
	if !request.DryRun {
		if strings.EqualFold(device.Kind, "HackRF") {
			tool, err := findTool("hackrf_transfer")
			if err != nil {
				runErr = errors.New("hackrf_transfer is not installed")
			} else {
				args := []string{"-t", iqPath, "-f", strconv.FormatInt(int64(request.FrequencyHz), 10), "-s", "2000000", "-x", strconv.Itoa(request.TXGainDB)}
				if device.Serial != nil && *device.Serial != "" {
					args = append(args, "-d", *device.Serial)
				}
				command := exec.CommandContext(ctx, tool, args...)
				_, runErr = command.Output()
			}
		} else {
			tool, err := findTool("gpsdr-soapy")
			if err != nil {
				runErr = errors.New("GP-SDR's SoapySDR transmit helper is not installed")
			} else {
				args := []string{"--device", soapyDeviceArguments(device), "--frequency", strconv.FormatInt(int64(request.FrequencyHz), 10), "--rate", "2000000", "--gain", strconv.Itoa(request.TXGainDB), "--bandwidth", "2000000", "--tx-file", iqPath}
				command := exec.CommandContext(ctx, tool, args...)
				_, runErr = command.Output()
			}
		}
	}
	finished := time.Now()
	r.transmit.mu.Lock()
	defer r.transmit.mu.Unlock()
	r.transmit.cancel = nil
	started.FinishedAt = &finished
	if ctx.Err() != nil {
		started.State, started.Note = "stopped", "Transmit stopped by the user."
	} else if runErr != nil {
		started.State, started.Note = "error", runErr.Error()
	} else if request.DryRun {
		started.State, started.Note = "complete", "Dry run complete; no RF was transmitted."
	} else {
		started.State, started.Note = "complete", "Transmit completed. Use a dummy load and verify local regulations before on-air use."
	}
	r.transmit.status = started
}

func (r *Runtime) StopTransmit() TransmitStatus {
	if r.transmit == nil {
		return r.TransmitStatus()
	}
	r.transmit.mu.Lock()
	cancel := r.transmit.cancel
	r.transmit.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return r.TransmitStatus()
}

// modulateTransmitAudio converts mono PCM into HackRF's signed interleaved
// IQ format. It is deterministic and bounded so unit tests can validate the
// waveform without claiming that an antenna test has occurred.
func modulateTransmitAudio(audio []int16, sourceRate, sampleRate, sampleCount int, mode string) ([]byte, error) {
	if len(audio) == 0 || sourceRate <= 0 || sampleRate <= 0 || sampleCount <= 0 {
		return nil, errors.New("invalid transmit audio")
	}
	output := make([]byte, sampleCount*2)
	phase := 0.0
	deviation := 2_500.0
	if mode == "wfm" {
		deviation = 75_000
	}
	for i := 0; i < sampleCount; i++ {
		// Consume the whole source rather than repeating its first second.
		// Linear interpolation avoids nearest-neighbor resampling steps.
		position := float64(i) * float64(sourceRate) / float64(sampleRate)
		index := int(position)
		level := 0.0
		if index < len(audio) {
			next := min(index+1, len(audio)-1)
			fraction := position - float64(index)
			level = (float64(audio[index])*(1-fraction) + float64(audio[next])*fraction) / 32768
		}
		if mode == "am" {
			amplitude := 0.45 + 0.35*level
			output[2*i] = signedIQByte(amplitude * 127)
			output[2*i+1] = 0
			continue
		}
		phase += 2 * math.Pi * deviation * level / float64(sampleRate)
		phase = math.Remainder(phase, 2*math.Pi)
		output[2*i] = signedIQByte(110 * math.Cos(phase))
		output[2*i+1] = signedIQByte(110 * math.Sin(phase))
	}
	return output, nil
}

func signedIQByte(value float64) byte {
	return byte(int8(max(-127, min(127, int(math.Round(value))))))
}

func clampByte(value float64) int {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return int(math.Round(value))
}
