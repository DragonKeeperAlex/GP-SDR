package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type recordingRelayPTT struct {
	states []bool
	err    error
}

func (p *recordingRelayPTT) SetPTT(asserted bool) error {
	p.states = append(p.states, asserted)
	return p.err
}

func TestRadioRelayKeysAndUnkeysAroundAudio(t *testing.T) {
	ptt := &recordingRelayPTT{}
	relay, err := NewRadioRelay(ptt, RelayConfig{KeyDelay: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	if err := relay.Run(context.Background(), func(context.Context) error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("audio callback was not called")
	}
	if !reflect.DeepEqual(ptt.states, []bool{true, false}) {
		t.Fatalf("PTT sequence = %v, want [true false]", ptt.states)
	}
}

func TestRadioRelayAlwaysReleasesPTTOnAudioError(t *testing.T) {
	ptt := &recordingRelayPTT{}
	relay, _ := NewRadioRelay(ptt, RelayConfig{})
	want := errors.New("audio unavailable")
	if err := relay.Run(context.Background(), func(context.Context) error { return want }); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if !reflect.DeepEqual(ptt.states, []bool{true, false}) {
		t.Fatalf("PTT sequence = %v, want [true false]", ptt.states)
	}
}

func TestRadioRelayEnforcesMaximumTransmitTime(t *testing.T) {
	ptt := &recordingRelayPTT{}
	relay, _ := NewRadioRelay(ptt, RelayConfig{MaxTransmit: 5 * time.Millisecond})
	err := relay.Run(context.Background(), func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	if err == nil {
		t.Fatal("expected maximum transmit error")
	}
	if !reflect.DeepEqual(ptt.states, []bool{true, false}) {
		t.Fatalf("PTT sequence = %v, want [true false]", ptt.states)
	}
}

func TestRadioRelayRejectsConcurrentRun(t *testing.T) {
	ptt := &recordingRelayPTT{}
	relay, _ := NewRadioRelay(ptt, RelayConfig{})
	started := make(chan struct{})
	finish := make(chan struct{})
	go func() {
		_ = relay.Run(context.Background(), func(context.Context) error {
			close(started)
			<-finish
			return nil
		})
	}()
	<-started
	if err := relay.Run(context.Background(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("expected concurrent relay run to be rejected")
	}
	close(finish)
}
