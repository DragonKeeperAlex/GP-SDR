package app

import (
	"encoding/binary"
	"fmt"
	"net"
)

// OP25 emits little-endian signed 16-bit mono audio at 8 kHz. Receive
// directly so a headless server needs neither a sound card nor ALSA routing.
func (m *OP25Manager) startOP25Audio(count int) error {
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
			for {
				n, _, err := socket.ReadFromUDP(buffer)
				if err != nil {
					return
				}
				if n < 2 || n%2 != 0 {
					continue
				}
				samples := make([]int16, n/2)
				for i := range samples {
					samples[i] = int16(binary.LittleEndian.Uint16(buffer[i*2:]))
				}
				m.audioHub.Publish(AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", index), SampleRate: 8000, Samples: samples})
			}
		}(index)
	}
	return nil
}

func (m *OP25Manager) closeOP25Audio() {
	m.mu.Lock()
	sockets := m.audioSockets
	m.audioSockets = nil
	m.mu.Unlock()
	for _, socket := range sockets {
		_ = socket.Close()
	}
}
