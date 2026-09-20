package app

import (
	"net"
	"testing"
	"time"
)

func TestOP25AudioStreamsPCMWithoutSoundDevice(t *testing.T) {
	hub := NewAudioHub()
	frames, cancel := hub.Subscribe()
	defer cancel()
	manager := &OP25Manager{audioHub: hub}
	if err := manager.startOP25Audio(1); err != nil {
		t.Fatal(err)
	}
	defer manager.stopProcess()
	socket, err := net.Dial("udp", "127.0.0.1:23456")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	// Drain/drop control flags and malformed packets must not become PCM.
	for _, packet := range [][]byte{{0, 0}, {1, 0}, {1, 2, 3}} {
		if _, err = socket.Write(packet); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = socket.Write([]byte{1, 0, 255, 255}); err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-frames:
		if frame.SampleRate != 8000 || len(frame.Samples) != 2 || frame.Samples[0] != 1 || frame.Samples[1] != -1 {
			t.Fatalf("bad PCM: %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("missing audio frame")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.audioFrames != 1 || manager.audioSamples != 2 || manager.audioLastAt.IsZero() {
		t.Fatalf("P25 audio telemetry was not updated: frames=%d samples=%d at=%v", manager.audioFrames, manager.audioSamples, manager.audioLastAt)
	}
}

func TestOP25AudioPortsCanBeReusedAfterStop(t *testing.T) {
	manager := &OP25Manager{audioHub: NewAudioHub()}
	for attempt := 0; attempt < 3; attempt++ {
		if err := manager.startOP25Audio(1); err != nil {
			t.Fatal(err)
		}
		manager.closeOP25Audio()
		manager.closeOP25Audio()
	}
}
