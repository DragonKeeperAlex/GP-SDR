package app

import "sync"

type AudioFrame struct {
	ChannelID  string
	SampleRate int
	Samples    []int16
}

type AudioHub struct {
	mu          sync.Mutex
	subscribers map[chan AudioFrame]struct{}
}

func NewAudioHub() *AudioHub {
	return &AudioHub{subscribers: make(map[chan AudioFrame]struct{})}
}

func (hub *AudioHub) Subscribe() (<-chan AudioFrame, func()) {
	// Keep enough real-time audio to absorb normal LAN and browser scheduling
	// jitter without letting a slow client grow memory without bound.
	channel := make(chan AudioFrame, 64)
	hub.mu.Lock()
	hub.subscribers[channel] = struct{}{}
	hub.mu.Unlock()
	return channel, func() {
		hub.mu.Lock()
		if _, ok := hub.subscribers[channel]; ok {
			delete(hub.subscribers, channel)
			close(channel)
		}
		hub.mu.Unlock()
	}
}

func (hub *AudioHub) Publish(frame AudioFrame) {
	if len(frame.Samples) == 0 || frame.SampleRate <= 0 || frame.ChannelID == "" {
		return
	}
	frame.Samples = append([]int16(nil), frame.Samples...)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for subscriber := range hub.subscribers {
		select {
		case subscriber <- frame:
		default:
			// Audio is real-time data. Discard the oldest queued frame, then offer
			// the newest one. The previous implementation dropped new audio here,
			// which made a briefly delayed browser play stale data followed by a
			// gap and sounded like packet loss.
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- frame:
			default:
			}
		}
	}
}
