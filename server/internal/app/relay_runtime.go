package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

type RelayStatus struct {
	State       string           `json:"state"`
	Note        string           `json:"note"`
	Active      bool             `json:"active"`
	PTTDevice   string           `json:"pttDevice"`
	AudioDevice string           `json:"audioDevice"`
	Segment     time.Duration    `json:"segmentNanoseconds"`
	Streams     []RelayStream    `json:"streams"`
	Audio       RelayAudioConfig `json:"audio"`
}

type RelayRuntime struct {
	mu       sync.Mutex
	active   bool
	status   string
	ptt      RelayPTT
	audio    RelayAudioConfig
	streams  []RelayStream
	cancel   context.CancelFunc
	done     chan struct{}
	mixer    *RelayMixer
	output   relayPCMOutput
	audioHub *AudioHub
}

type relayPCMOutput interface {
	Start(context.Context) error
	Write([]int16) error
	Stop() error
}

func NewRelayRuntime(audioHub *AudioHub) *RelayRuntime {
	streams := []RelayStream{{ID: "p25-stream-0", Name: "P25 stream 1", Enabled: true, Volume: 1}}
	mixer, _ := NewRelayMixer(streams)
	return &RelayRuntime{status: "stopped", ptt: FT232PTT{Device: "/dev/ttyUSB0"}, audio: RelayAudioConfig{Device: "plughw:2,0", SampleRate: 8000, Channels: 1}, streams: streams, mixer: mixer, audioHub: audioHub}
}

func newRelayRuntimeForTest(audioHub *AudioHub, ptt RelayPTT, output relayPCMOutput) *RelayRuntime {
	streams := []RelayStream{{ID: "p25-stream-0", Name: "P25 stream 1", Enabled: true, Volume: 1}}
	mixer, _ := NewRelayMixer(streams)
	return &RelayRuntime{status: "stopped", ptt: ptt, audio: RelayAudioConfig{Device: "test", SampleRate: 8000, Channels: 1}, streams: streams, mixer: mixer, audioHub: audioHub, output: output}
}

func (r *RelayRuntime) Status() RelayStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return RelayStatus{State: r.status, Note: "Relay audio routing is configured for the isolated FT232/USB-audio path.", Active: r.active, PTTDevice: "/dev/ttyUSB0", AudioDevice: r.audio.Device, Segment: DefaultRelaySegment, Streams: append([]RelayStream(nil), r.streams...), Audio: r.audio}
}

func (r *RelayRuntime) Configure(streams []RelayStream, audio RelayAudioConfig) (RelayStatus, error) {
	mixer, err := NewRelayMixer(streams)
	if err != nil {
		return r.Status(), err
	}
	if len(mixer.Streams()) == 0 {
		return r.Status(), errors.New("select at least one relay stream")
	}
	r.mu.Lock()
	r.streams = mixer.Streams()
	r.mixer = mixer
	r.audio = audio.normalized()
	r.mu.Unlock()
	return r.Status(), nil
}

func (r *RelayRuntime) Start() RelayStatus {
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return r.Status()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	if r.output == nil {
		r.output = NewRelayAudioOutput(r.audio)
	}
	output := r.output
	if err := output.Start(ctx); err != nil {
		r.status = "error"
		r.mu.Unlock()
		return r.Status()
	}
	r.output = output
	r.active = true
	r.status = "ready"
	r.mu.Unlock()
	go r.consumeAudio(ctx, r.done)
	return r.Status()
}

func (r *RelayRuntime) Stop() RelayStatus {
	r.mu.Lock()
	cancel, done, output := r.cancel, r.done, r.output
	r.cancel, r.done, r.output = nil, nil, nil
	r.active = false
	r.status = "stopped"
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	_ = r.ptt.SetPTT(false)
	if done != nil {
		<-done
	}
	if output != nil {
		_ = output.Stop()
	}
	return r.Status()
}

func (r *RelayRuntime) consumeAudio(ctx context.Context, done chan struct{}) {
	if r.audioHub == nil {
		close(done)
		return
	}
	frames, unsubscribe := r.audioHub.Subscribe()
	defer unsubscribe()
	defer close(done)
	selected := make(map[string][]int16)
	segment := time.NewTimer(DefaultRelaySegment)
	if !segment.Stop() {
		<-segment.C
	}
	keyed := false
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				return
			}
			r.mu.Lock()
			mixer, output := r.mixer, r.output
			r.mu.Unlock()
			if mixer == nil || output == nil {
				continue
			}
			selected[frame.ChannelID] = frame.Samples
			mixed := mixer.Mix(selected)
			if len(mixed) == 0 {
				continue
			}
			if !keyed {
				if err := r.ptt.SetPTT(true); err != nil {
					continue
				}
				keyed = true
				segment.Reset(DefaultRelaySegment)
			}
			if err := output.Write(mixed); err != nil {
				continue
			}
		case <-segment.C:
			if keyed {
				_ = r.ptt.SetPTT(false)
				keyed = false
				time.Sleep(250 * time.Millisecond)
			}
		}
	}
}
