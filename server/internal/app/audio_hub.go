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
	channel := make(chan AudioFrame, 16)
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
			// Audio is real-time data. A slow browser drops an old frame rather
			// than blocking RF capture, logging, or other listeners.
		}
	}
}
