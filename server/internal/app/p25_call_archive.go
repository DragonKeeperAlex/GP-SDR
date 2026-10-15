package app

import (
	"fmt"
	"path/filepath"
	"time"
)

// p25CallArchive keeps bounded, per-OP25-stream PCM for an unencrypted call.
// Three minutes caps a busy three-receiver Pi at under 9 MiB of call audio.
type p25CallArchive struct {
	streamID string
	samples  []int16
	rate     int
}

func newP25CallArchive(index int) *p25CallArchive {
	return &p25CallArchive{streamID: fmt.Sprintf("p25-stream-%d", index), rate: 8000}
}
func (a *p25CallArchive) Append(frame AudioFrame) {
	if frame.ChannelID != a.streamID || frame.SampleRate != a.rate || len(a.samples) >= a.rate*180 {
		return
	}
	remaining := a.rate*180 - len(a.samples)
	if len(frame.Samples) > remaining {
		frame.Samples = frame.Samples[:remaining]
	}
	a.samples = append(a.samples, frame.Samples...)
}
func (a *p25CallArchive) Save(root string, groupID uint32, started time.Time) (string, error) {
	if len(a.samples) == 0 {
		return "", fmt.Errorf("no P25 PCM received for talkgroup %d", groupID)
	}
	path := filepath.Join(root, "Recordings", started.UTC().Format("2006-01-02"), fmt.Sprintf("%s-p25-tg%d.wav", started.UTC().Format("20060102T150405.000Z"), groupID))
	return path, WriteMonoWAV(path, a.samples, a.rate)
}
