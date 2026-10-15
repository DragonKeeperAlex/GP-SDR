package app

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestRelayMixerSumsBeforeClipping(t *testing.T) {
	m, _ := NewRelayMixer([]RelayStream{{ID: "a", Enabled: true, Volume: 1}, {ID: "b", Enabled: true, Volume: 1}, {ID: "c", Enabled: true, Volume: 1}})
	for i := 0; i < 100; i++ {
		got := m.Mix(map[string][]int16{"a": {30000}, "b": {30000}, "c": {-30000}})
		if got[0] != 30000 {
			t.Fatalf("order-dependent clipping: %v", got)
		}
	}
}

func TestRelayReleasesPTTAfterAudioEnds(t *testing.T) {
	hub := NewAudioHub()
	ptt := &relayTestPTT{}
	r := newRelayRuntimeForTest(hub, ptt, &relayObservedOutput{frames: make(chan []int16, 10)})
	r.Start()
	defer r.Stop()
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{100}})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		ptt.mu.Lock()
		states := append([]bool(nil), ptt.states...)
		ptt.mu.Unlock()
		if len(states) >= 2 && !states[len(states)-1] {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("PTT remained asserted after end of audio")
}

type relayFailOutput struct{}

func (*relayFailOutput) Start(context.Context) error { return nil }
func (*relayFailOutput) Write([]int16) error         { return errors.New("USB disconnected") }
func (*relayFailOutput) Stop() error                 { return nil }

func TestRelayOutputFailureUnkeysAndReportsError(t *testing.T) {
	hub := NewAudioHub()
	ptt := &relayTestPTT{}
	r := newRelayRuntimeForTest(hub, ptt, &relayFailOutput{})
	r.Start()
	defer r.Stop()
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{100}})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if r.Status().State == "error" {
			r.Stop()
			ptt.mu.Lock()
			defer ptt.mu.Unlock()
			if len(ptt.states) < 2 || ptt.states[len(ptt.states)-1] {
				t.Fatal("PTT not released")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("output failure was hidden")
}

func TestRelayZeroVolumeRemainsMuted(t *testing.T) {
	m, err := NewRelayMixer([]RelayStream{{ID: "muted", Enabled: true, Volume: 0}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range m.Mix(map[string][]int16{"muted": {1234, -1234}}) {
		if s != 0 {
			t.Fatalf("zero-volume stream produced %d", s)
		}
	}
}

func TestRelayRejectsNonfiniteVolume(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := NewRelayMixer([]RelayStream{{ID: "bad", Volume: v}}); err == nil {
			t.Fatalf("accepted volume %v", v)
		}
	}
}

type relayObservedOutput struct{ frames chan []int16 }

func (o *relayObservedOutput) Start(context.Context) error { return nil }
func (o *relayObservedOutput) Write(s []int16) error {
	o.frames <- append([]int16(nil), s...)
	return nil
}
func (o *relayObservedOutput) Stop() error { return nil }

func TestRelayDoesNotReplaySelectedAudioOnUnselectedFrame(t *testing.T) {
	hub := NewAudioHub()
	o := &relayObservedOutput{frames: make(chan []int16, 10)}
	r := newRelayRuntimeForTest(hub, &relayTestPTT{}, o)
	r.Start()
	defer r.Stop()
	time.Sleep(20 * time.Millisecond)
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{100, 200}})
	select {
	case <-o.frames:
	case <-time.After(time.Second):
		t.Fatal("selected audio not delivered")
	}
	hub.Publish(AudioFrame{ChannelID: "unselected", SampleRate: 8000, Samples: []int16{999}})
	select {
	case frame := <-o.frames:
		t.Fatalf("unselected input replayed stale audio: %v", frame)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestRelayResamplesIncomingAudio(t *testing.T) {
	hub := NewAudioHub()
	o := &relayObservedOutput{frames: make(chan []int16, 10)}
	r := newRelayRuntimeForTest(hub, &relayTestPTT{}, o)
	r.Start()
	defer r.Stop()
	time.Sleep(20 * time.Millisecond)
	samples := make([]int16, 1600)
	for i := range samples {
		samples[i] = 100
	}
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 16000, Samples: samples})
	count := 0
	deadline := time.After(time.Second)
	for count < 800 {
		select {
		case frame := <-o.frames:
			count += len(frame)
		case <-deadline:
			t.Fatalf("got %d samples, want 800", count)
		}
	}
	if count != 800 {
		t.Fatalf("got %d samples, want 800", count)
	}
	select {
	case frame := <-o.frames:
		t.Fatalf("extra audio: %d samples", len(frame))
	case <-time.After(80 * time.Millisecond):
	}
}
