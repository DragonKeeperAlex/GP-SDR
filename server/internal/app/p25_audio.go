package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

// OP25 emits little-endian signed 16-bit mono audio at 8 kHz. Receive
// directly so a headless server needs neither a sound card nor ALSA routing.
func (m *OP25Manager) startOP25Audio(count int) error {
	session := NewID()
	m.mu.Lock()
	m.audioSession = session
	m.mu.Unlock()
	for index := 0; index < count; index++ {
		socket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 23456 + index*10})
		if err != nil {
			m.stopProcess()
			return fmt.Errorf("P25 audio port: %w", err)
		}
		m.mu.Lock()
		m.audioSockets = append(m.audioSockets, socket)
		m.mu.Unlock()
		go func(index int) {
			buffer := make([]byte, 65536)
			var sequence uint64
			nativeSession := NewID()
			requireNative := os.Getenv("GPSDR_P25_TAGGED_AUDIO") == "1"
			for {
				n, _, err := socket.ReadFromUDP(buffer)
				if err != nil {
					return
				}
				if bytes.HasPrefix(buffer[:n], []byte("GPSDP25")) {
					frame, err := decodeP25NativeAudio(buffer[:n], index, nativeSession)
					if err != nil {
						m.rejectOP25Audio(session, true)
						continue
					}
					frame.ReceivedAt = time.Now()
					m.publishOP25Audio(session, frame, true, buffer[9] == 0 && frame.Control == "drain")
					continue
				}
				if requireNative {
					m.rejectOP25Audio(session, false)
					continue
				}
				// OP25 reserves two-byte UDP packets for DRAIN/DROP flags.
				// They are not one-sample PCM frames (sockaudio.py agrees).
				if n == 2 {
					flag := binary.LittleEndian.Uint16(buffer[:2])
					if flag <= 1 {
						sequence++
						control := "drain"
						if flag == 1 {
							control = "drop"
						}
						m.publishOP25Audio(session, AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", index), SampleRate: 8000, Control: control, Sequence: sequence, ReceivedAt: time.Now()}, false, false)
					}
					continue
				}
				if n < 2 || n%2 != 0 {
					continue
				}
				samples := make([]int16, n/2)
				for i := range samples {
					samples[i] = int16(binary.LittleEndian.Uint16(buffer[i*2:]))
				}
				sequence++
				m.publishOP25Audio(session, AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", index), SampleRate: 8000, Samples: samples, Sequence: sequence, ReceivedAt: time.Now()}, false, false)
			}
		}(index)
	}
	return nil
}

func (m *OP25Manager) closeOP25Audio() {
	m.mu.Lock()
	m.audioSession = ""
	m.taggedTalkgroupAudio = false
	sockets := m.audioSockets
	m.audioSockets = nil
	m.mu.Unlock()
	for _, socket := range sockets {
		_ = socket.Close()
	}
}

// Serialize publication with Stop/replacement. A reader that already received a
// datagram must not publish it into a later session after its socket is closed.
func (m *OP25Manager) publishOP25Audio(session string, frame AudioFrame, native, suppressed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.audioSession != session || session == "" {
		return
	}
	if native {
		m.nativeAudioPackets++
	}
	if suppressed {
		m.nativeAudioRejected++
	}
	if len(frame.Samples) > 0 {
		m.audioFrames++
		m.audioSamples += uint64(len(frame.Samples))
		m.audioLastAt = frame.ReceivedAt
	}
	frame.TransportSession = session
	m.audioHub.Publish(frame)
}
func (m *OP25Manager) rejectOP25Audio(session string, native bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.audioSession != session || session == "" {
		return
	}
	if native {
		m.nativeAudioRejected++
	} else {
		m.untaggedAudioPackets++
	}
}
