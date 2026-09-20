package app

import "time"

const DefaultRelaySegment = 30 * time.Second

// RelaySegments returns bounded transmission windows. A caller should unkey
// between every adjacent window and rekey only when the next window contains
// audio, avoiding reliance on the handheld's internal timeout.
func RelaySegments(total, segment time.Duration) []time.Duration {
	if total <= 0 {
		return nil
	}
	if segment <= 0 {
		segment = DefaultRelaySegment
	}
	result := []time.Duration{}
	for total > 0 {
		window := segment
		if total < window {
			window = total
		}
		result = append(result, window)
		total -= window
	}
	return result
}
