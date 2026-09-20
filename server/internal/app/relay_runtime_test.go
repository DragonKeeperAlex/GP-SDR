package app

import (
	"context"
	"sync"
	"testing"
	"time"
)

type relayTestPTT struct {
	mu     sync.Mutex
	states []bool
}

func (p *relayTestPTT) SetPTT(on bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.states = append(p.states, on)
	return nil
}

type relayTestOutput struct {
	mu              sync.Mutex
	writes, samples int
}

func (o *relayTestOutput) Start(context.Context) error { return nil }
func (o *relayTestOutput) Write(samples []int16) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.writes++
	o.samples += len(samples)
	return nil
}
func (o *relayTestOutput) Stop() error { return nil }

func TestRelayRuntimeConsumesSelectedAudioAndStopsSafely(t *testing.T) {
	hub := NewAudioHub()
	ptt := &relayTestPTT{}
	output := &relayTestOutput{}
	r := newRelayRuntimeForTest(hub, ptt, output)
	r.Start()
	time.Sleep(10 * time.Millisecond)
	hub.Publish(AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{100, 200, 300}})
	time.Sleep(20 * time.Millisecond)
	r.Stop()
	output.mu.Lock()
	writes, samples := output.writes, output.samples
	output.mu.Unlock()
	if writes == 0 || samples != 3 {
		t.Fatalf("writes=%d samples=%d, want one write of three samples", writes, samples)
	}
	ptt.mu.Lock()
	states := append([]bool(nil), ptt.states...)
	ptt.mu.Unlock()
	if len(states) < 2 || !states[0] || states[len(states)-1] {
		t.Fatalf("PTT states=%v, want assert then final release", states)
	}
}
