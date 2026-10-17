package app

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Decoder-native facts only. Frequency/site still require matching recent
// receiver metadata; the archive must never substitute a polled talkgroup.
type P25NativeAudio struct {
	CallID                        string
	GroupID, SourceID             uint32
	NAC                           uint16
	Slot                          int
	Algorithm                     uint8
	Valid, CryptoKnown, Encrypted bool
}

const p25NativeMagic = "GPSDP25\x00"

func decodeP25NativeAudio(packet []byte, stream int, session string) (AudioFrame, error) {
	if len(packet) < 32 || string(packet[:8]) != p25NativeMagic || packet[8] != 1 || packet[9] > 2 || (len(packet)-32)%2 != 0 {
		return AudioFrame{}, errors.New("invalid P25 native audio packet")
	}
	flags := binary.LittleEndian.Uint16(packet[10:12])
	id := binary.LittleEndian.Uint32(packet[16:20])
	native := &P25NativeAudio{CallID: fmt.Sprintf("%s:%d", session, id), GroupID: binary.LittleEndian.Uint32(packet[20:24]), SourceID: binary.LittleEndian.Uint32(packet[24:28]), NAC: binary.LittleEndian.Uint16(packet[28:30]), Slot: int(int8(packet[30])), Algorithm: packet[31], Valid: flags&1 != 0, CryptoKnown: flags&2 != 0, Encrypted: flags&4 != 0}
	if flags & ^uint16(7) != 0 || native.GroupID > 65535 || native.SourceID > 0xffffff || native.Slot < -1 || native.Slot > 1 {
		return AudioFrame{}, errors.New("invalid native call identity")
	}
	frame := AudioFrame{ChannelID: fmt.Sprintf("p25-stream-%d", stream), SampleRate: 8000, Sequence: uint64(binary.LittleEndian.Uint32(packet[12:16])), Native: native}
	if frame.Sequence == 0 {
		return AudioFrame{}, errors.New("missing native sequence")
	}
	switch packet[9] {
	case 1:
		frame.Control = "drain"
	case 2:
		frame.Control = "drop"
	default:
		if len(packet) == 32 {
			return AudioFrame{}, errors.New("empty native PCM")
		}
		// Unknown identity/crypto and decrypted-but-encrypted traffic never enter
		// playback, recording or transcription. End only the preceding clear segment.
		if !native.Valid || native.GroupID == 0 || id == 0 || !native.CryptoKnown || native.Encrypted || native.Algorithm != 0x80 {
			frame.Control = "drain"
			return frame, nil
		}
		frame.Samples = make([]int16, (len(packet)-32)/2)
		for i := range frame.Samples {
			frame.Samples[i] = int16(binary.LittleEndian.Uint16(packet[32+2*i:]))
		}
	}
	if frame.Control != "" && len(packet) != 32 {
		return AudioFrame{}, errors.New("control contains PCM")
	}
	return frame, nil
}

func (m *OP25Manager) taggedTalkgroupPlayback() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.taggedTalkgroupAudio
}

// Single-system profiles have an unambiguous talkgroup mixer. Different native
// call generations keep independent browser clocks even on the same talkgroup.
func (r *Runtime) liveAudioIdentifier(frame AudioFrame) string {
	tagged := false
	if frame.TransportSession != "" || frame.Native != nil {
		if r.op25 == nil {
			return ""
		}
		r.op25.mu.Lock()
		current := r.op25.audioSession
		tagged = r.op25.taggedTalkgroupAudio
		r.op25.mu.Unlock()
		if frame.TransportSession == "" || current != frame.TransportSession {
			return ""
		}
	}
	n := frame.Native
	if n == nil {
		return frame.ChannelID
	}
	if !n.Valid || !n.CryptoKnown || n.Encrypted || n.Algorithm != 0x80 {
		return ""
	}
	if tagged {
		return p25MixerID(n.GroupID) + ":" + n.CallID
	}
	return frame.ChannelID
}
