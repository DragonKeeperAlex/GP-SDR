package app

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestOP25AudioControlsReachOnlyRecorderInOrder(t *testing.T) {
	hub := NewAudioHub()
	playback, cancelPlayback := hub.Subscribe()
	defer cancelPlayback()
	recorder, cancelRecorder := hub.subscribeP25()
	defer cancelRecorder()
	m := &OP25Manager{audioHub: hub}
	if err := m.startOP25Audio(3); err != nil {
		t.Fatal(err)
	}
	defer m.stopProcess()
	for i := 0; i < 3; i++ {
		socket, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", 23456+i*10))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = socket.Write([]byte{byte(i + 1), 0, byte(i + 1), 0}); err != nil {
			t.Fatal(err)
		}
		flag := byte(0)
		if i == 1 {
			flag = 1
		}
		if _, err = socket.Write([]byte{flag, 0}); err != nil {
			t.Fatal(err)
		}
		socket.Close()
	}
	sequences := map[string]uint64{}
	controls := map[string]string{}
	for i := 0; i < 6; i++ {
		select {
		case frame := <-recorder:
			if frame.Sequence != sequences[frame.ChannelID]+1 || frame.ReceivedAt.IsZero() {
				t.Fatal("recorder lost packet order/time")
			}
			sequences[frame.ChannelID] = frame.Sequence
			if frame.Control != "" {
				controls[frame.ChannelID] = frame.Control
			}
		case <-time.After(time.Second):
			t.Fatal("missing recorder packet")
		}
	}
	if controls["p25-stream-0"] != "drain" || controls["p25-stream-1"] != "drop" || controls["p25-stream-2"] != "drain" {
		t.Fatal("wrong boundary flags", controls)
	}
	for i := 0; i < 3; i++ {
		select {
		case frame := <-playback:
			if frame.Control != "" || len(frame.Samples) != 2 {
				t.Fatal("control packet leaked to playback")
			}
		case <-time.After(time.Second):
			t.Fatal("missing playback PCM")
		}
	}
	select {
	case extra := <-playback:
		t.Fatal("extra playback packet", extra)
	default:
	}
}

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
