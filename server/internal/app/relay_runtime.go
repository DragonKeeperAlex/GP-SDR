package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type RelayStatus struct {
	SystemAudio  bool             `json:"systemAudio"`
	SpeakerMuted bool             `json:"speakerMuted"`
	Keyed        bool             `json:"keyed"`
	KeyCount     uint64           `json:"keyCount"`
	AudioSamples uint64           `json:"audioSamples"`
	State        string           `json:"state"`
	Note         string           `json:"note"`
	Active       bool             `json:"active"`
	PTTDevice    string           `json:"pttDevice"`
	AudioDevice  string           `json:"audioDevice"`
	Segment      time.Duration    `json:"segmentNanoseconds"`
	Streams      []RelayStream    `json:"streams"`
	Audio        RelayAudioConfig `json:"audio"`
	OutputVolume float64          `json:"outputVolume"`
}

type RelayRuntime struct {
	systemAudio  bool
	speakerMuted bool
	keyed        bool
	keyCount     uint64
	audioSamples uint64
	lifecycle    sync.Mutex
	mu           sync.Mutex
	active       bool
	status       string
	lastError    string
	outputVolume float64
	settingsPath string
	ptt          RelayPTT
	audio        RelayAudioConfig
	streams      []RelayStream
	cancel       context.CancelFunc
	done         chan struct{}
	mixer        *RelayMixer
	output       relayPCMOutput
	audioHub     *AudioHub
}

type relayPCMOutput interface {
	Start(context.Context) error
	Write([]int16) error
	Stop() error
}

func NewRelayRuntime(audioHub *AudioHub, dataDirectory ...string) *RelayRuntime {
	streams := []RelayStream{{ID: "p25-stream-0", Name: "P25 stream 1", Enabled: true, Volume: 1}}
	mixer, _ := NewRelayMixer(streams)
	r := &RelayRuntime{status: "stopped", outputVolume: .2, ptt: FT232PTT{Device: "/dev/ttyUSB0"}, audio: RelayAudioConfig{Device: "plughw:CARD=A,DEV=0", SampleRate: 8000, Channels: 1}, streams: streams, mixer: mixer, audioHub: audioHub}
	if len(dataDirectory) > 0 && dataDirectory[0] != "" {
		r.loadOutputVolume(dataDirectory[0])
	}
	return r
}

func newRelayRuntimeForTest(audioHub *AudioHub, ptt RelayPTT, output relayPCMOutput) *RelayRuntime {
	streams := []RelayStream{{ID: "p25-stream-0", Name: "P25 stream 1", Enabled: true, Volume: 1}}
	mixer, _ := NewRelayMixer(streams)
	return &RelayRuntime{status: "stopped", outputVolume: 1, ptt: ptt, audio: RelayAudioConfig{Device: "test", SampleRate: 8000, Channels: 1}, streams: streams, mixer: mixer, audioHub: audioHub, output: output}
}

func (r *RelayRuntime) Status() RelayStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	note := "Relay audio routing is configured for the isolated FT232/USB-audio path."
	if r.lastError != "" {
		note = r.lastError
	}
	return RelayStatus{State: r.status, Note: note, Active: r.active, PTTDevice: "/dev/ttyUSB0", AudioDevice: r.audio.Device, Segment: DefaultRelaySegment, Streams: append([]RelayStream(nil), r.streams...), Audio: r.audio, OutputVolume: r.outputVolume, SystemAudio: r.systemAudio, SpeakerMuted: r.speakerMuted, Keyed: r.keyed, KeyCount: r.keyCount, AudioSamples: r.audioSamples}
}

func (r *RelayRuntime) Configure(streams []RelayStream, audio RelayAudioConfig) (RelayStatus, error) {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	audio = audio.normalized()
	if audio.Channels != 1 || audio.SampleRate < 8000 || audio.SampleRate > 48000 {
		return r.Status(), errors.New("relay requires mono PCM at 8,000–48,000 Hz")
	}
	mixer, err := NewRelayMixer(streams)
	if err != nil {
		return r.Status(), err
	}
	if len(mixer.Streams()) == 0 {
		return r.Status(), errors.New("select at least one relay stream")
	}
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return r.Status(), errors.New("stop the relay before changing routing or output")
	}
	r.streams = mixer.Streams()
	r.mixer = mixer
	r.audio = audio.normalized()
	r.mu.Unlock()
	return r.Status(), nil
}

func (r *RelayRuntime) Start() RelayStatus {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return r.Status()
	}
	// Reap a prior failed run before creating another output.
	oldCancel, oldDone, oldOutput := r.cancel, r.done, r.output
	r.mu.Unlock()
	if oldDone != nil {
		if oldCancel != nil {
			oldCancel()
		}
		<-oldDone
		if oldOutput != nil {
			_ = oldOutput.Stop()
		}
	}
	r.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	output := r.output
	if oldDone != nil {
		output = nil
	}
	if output == nil {
		output = NewRelayAudioOutput(r.audio)
	}
	if err := output.Start(ctx); err != nil {
		cancel()
		r.status, r.lastError = "error", err.Error()
		r.cancel, r.done, r.output = nil, nil, nil
		r.mu.Unlock()
		return r.Status()
	}
	if r.audioHub == nil {
		cancel()
		_ = output.Stop()
		r.status, r.lastError = "error", "relay audio source is unavailable"
		r.mu.Unlock()
		return r.Status()
	}
	var frames <-chan AudioFrame
	var unsubscribe func()
	rate, mixer := r.audio.SampleRate, r.mixer
	if r.systemAudio {
		var err error
		frames, unsubscribe, err = startSystemAudioCapture(ctx, rate, r.relayFailure)
		if err != nil {
			cancel()
			_ = output.Stop()
			r.status, r.lastError = "error", err.Error()
			r.mu.Unlock()
			return r.Status()
		}
		mixer, _ = NewRelayMixer([]RelayStream{{ID: "system-audio", Name: "System audio", Enabled: true, Volume: 1}})
	} else {
		frames, unsubscribe = r.audioHub.Subscribe()
	}
	done := make(chan struct{})
	r.cancel, r.done, r.output = cancel, done, output
	r.active, r.status, r.lastError = true, "ready", ""
	r.mu.Unlock()
	go func() {
		defer close(done)
		defer unsubscribe()
		defer cancel()
		r.consumeAudio(ctx, frames, mixer, output, rate)
	}()
	return r.Status()
}

func (r *RelayRuntime) Stop() RelayStatus {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	r.mu.Lock()
	cancel, done, output := r.cancel, r.done, r.output
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	if output != nil {
		_ = output.Stop()
	}
	r.mu.Lock()
	r.cancel, r.done, r.output = nil, nil, nil
	r.active = false
	if r.lastError == "" {
		r.status = "stopped"
	}
	r.mu.Unlock()
	return r.Status()
}

func (r *RelayRuntime) relayFailure(err error) {
	r.mu.Lock()
	r.active, r.status, r.lastError = false, "error", err.Error()
	r.mu.Unlock()
}

func (r *RelayRuntime) consumeAudio(ctx context.Context, frames <-chan AudioFrame, mixer *RelayMixer, output relayPCMOutput, rate int) {
	// A short clocked queue consumes each sample once, aligns streams, and
	// prevents a silent or unrelated stream from replaying the last voice frame.
	queues := make(map[string][]int16)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	keyed := false
	var keyedAt, lastAudio, resumeAt time.Time
	release := func() bool {
		if !keyed {
			return true
		}
		if err := r.ptt.SetPTT(false); err != nil {
			r.relayFailure(err)
			return false
		}
		keyed = false
		r.mu.Lock()
		r.keyed = false
		r.mu.Unlock()
		return true
	}
	defer func() {
		defer func() { r.mu.Lock(); r.keyed = false; r.mu.Unlock() }()
		if keyed {
			if err := r.ptt.SetPTT(false); err != nil {
				r.relayFailure(err)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				return
			}
			stream, selected := mixer.streams[frame.ChannelID]
			if !selected || !stream.Enabled || stream.Volume == 0 || frame.SampleRate <= 0 {
				continue
			}
			// Bound every stream to two seconds, including before resampling.
			samples := frame.Samples
			if len(samples) > 2*frame.SampleRate {
				samples = samples[len(samples)-2*frame.SampleRate:]
			}
			q := append(queues[frame.ChannelID], resamplePCM(samples, frame.SampleRate, rate)...)
			if len(q) > rate*2 {
				q = q[len(q)-rate*2:]
			}
			queues[frame.ChannelID] = q
		case now := <-tick.C:
			if keyed && now.Sub(keyedAt) >= DefaultRelaySegment {
				if !release() {
					return
				}
				resumeAt = now.Add(250 * time.Millisecond)
			}
			if now.Before(resumeAt) {
				continue
			}
			chunk := make(map[string][]int16)
			for id, q := range queues {
				n := min(len(q), rate/50)
				if n > 0 {
					chunk[id] = q[:n]
					queues[id] = q[n:]
				}
			}
			mixed := mixer.Mix(chunk)
			r.mu.Lock()
			volume := r.outputVolume
			r.mu.Unlock()
			for i := range mixed {
				mixed[i] = int16(float64(mixed[i]) * volume)
			}
			audible := false
			for _, sample := range mixed {
				if sample > 16 || sample < -16 {
					audible = true
					break
				}
			}
			if !audible && !keyed {
				continue
			}
			if len(mixed) == 0 {
				if keyed && now.Sub(lastAudio) >= 300*time.Millisecond {
					if !release() {
						return
					}
				}
				continue
			}
			if !keyed {
				if ctx.Err() != nil {
					return
				}
				// Mark first so an assertion failure still attempts a final release.
				keyed = true
				if err := r.ptt.SetPTT(true); err != nil {
					r.relayFailure(err)
					return
				}
				keyedAt = time.Now()
				r.mu.Lock()
				r.keyed = true
				r.keyCount++
				r.mu.Unlock()
			}
			if audible {
				lastAudio = now
			}
			// A stalled USB-audio writer must not hold PTT indefinitely.
			written := make(chan error, 1)
			go func() { written <- output.Write(mixed) }()
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case err := <-written:
				timer.Stop()
				if err != nil {
					r.relayFailure(fmt.Errorf("relay audio: %w", err))
					return
				}
				r.mu.Lock()
				r.audioSamples += uint64(len(mixed))
				r.mu.Unlock()
			case <-timer.C:
				r.relayFailure(errors.New("relay audio output stalled"))
				return
			}
			if !audible && now.Sub(lastAudio) >= 300*time.Millisecond {
				if !release() {
					return
				}
			}
		}
	}
}
