package app

import (
	"sync"
	"time"
)

type AudioFrame struct {
	ChannelID  string
	SampleRate int
	Samples    []int16
	// Recorder-only transport evidence; ordinary audio subscribers never see
	// control packets. The browser PCM wire format is unchanged.
	Control    string
	Sequence   uint64
	ReceivedAt time.Time
}

type AudioHub struct {
	mu          sync.Mutex
	subscribers map[chan AudioFrame]bool
}

func NewAudioHub() *AudioHub {
	return &AudioHub{subscribers: make(map[chan AudioFrame]bool)}
}

func (hub *AudioHub) Subscribe() (<-chan AudioFrame, func()) {
	return hub.subscribe(false)
}

func (hub *AudioHub) subscribeP25() (<-chan AudioFrame, func()) {
	return hub.subscribe(true)
}

func (hub *AudioHub) subscribe(controls bool) (<-chan AudioFrame, func()) {
	// Keep enough real-time audio to absorb normal LAN and browser scheduling
	// jitter without letting a slow client grow memory without bound.
	capacity := 64
	if controls {
		capacity = 256
	} // bounded headroom for three streams during metadata polling
	channel := make(chan AudioFrame, capacity)
	hub.mu.Lock()
	hub.subscribers[channel] = controls
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
	if frame.SampleRate <= 0 || frame.ChannelID == "" || (frame.Control != "" && frame.Control != "drain" && frame.Control != "drop") || (frame.Control == "" && len(frame.Samples) == 0) {
		return
	}
	frame.Samples = append([]int16(nil), frame.Samples...)
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for subscriber, controls := range hub.subscribers {
		if frame.Control != "" && !controls {
			continue
		}
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
