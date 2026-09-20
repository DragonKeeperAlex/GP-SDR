package app

import (
	"errors"
	"math"
	"sort"
)

type RelayStream struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Enabled bool    `json:"enabled"`
	Volume  float64 `json:"volume"`
}

type RelayMixer struct {
	streams map[string]RelayStream
}

func NewRelayMixer(streams []RelayStream) (*RelayMixer, error) {
	m := &RelayMixer{streams: make(map[string]RelayStream, len(streams))}
	for _, stream := range streams {
		if stream.ID == "" {
			return nil, errors.New("relay stream ID is required")
		}
		if stream.Volume == 0 {
			stream.Volume = 1
		}
		if stream.Volume < 0 || stream.Volume > 2 {
			return nil, errors.New("relay stream volume must be between 0 and 2")
		}
		m.streams[stream.ID] = stream
	}
	return m, nil
}

func (m *RelayMixer) Streams() []RelayStream {
	result := make([]RelayStream, 0, len(m.streams))
	for _, stream := range m.streams {
		result = append(result, stream)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (m *RelayMixer) SetStream(stream RelayStream) error {
	if stream.ID == "" {
		return errors.New("relay stream ID is required")
	}
	if stream.Volume < 0 || stream.Volume > 2 {
		return errors.New("relay stream volume must be between 0 and 2")
	}
	m.streams[stream.ID] = stream
	return nil
}

// Mix combines frames by stream ID. Missing streams contribute silence and
// output is hard-limited to int16 to prevent summed-stream clipping wraparound.
func (m *RelayMixer) Mix(frames map[string][]int16) []int16 {
	length := 0
	for id, samples := range frames {
		if stream, ok := m.streams[id]; ok && stream.Enabled && len(samples) > length {
			length = len(samples)
		}
	}
	output := make([]int16, length)
	for id, samples := range frames {
		stream, ok := m.streams[id]
		if !ok || !stream.Enabled {
			continue
		}
		for i, sample := range samples {
			value := float64(output[i]) + float64(sample)*stream.Volume
			output[i] = int16(math.Max(-32768, math.Min(32767, value)))
		}
	}
	return output
}
