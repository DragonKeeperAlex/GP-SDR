package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// p25CallArchive keeps bounded, per-OP25-stream PCM for an unencrypted call.
// Three minutes caps a busy three-receiver Pi at under 9 MiB of call audio.
type p25CallArchive struct {
	streamID  string
	samples   []int16
	rate      int
	truncated bool
	eventID   string
	spool     *p25CallSpool
	issues    []string
}

func newP25CallArchive(index int) *p25CallArchive {
	return &p25CallArchive{streamID: fmt.Sprintf("p25-stream-%d", index), rate: 8000}
}
func (a *p25CallArchive) Append(frame AudioFrame) {
	if frame.ChannelID != a.streamID || frame.SampleRate != a.rate {
		return
	}
	remaining := a.rate*180 - len(a.samples)
	if len(frame.Samples) > remaining {
		a.truncated = true
		frame.Samples = frame.Samples[:remaining]
	}
	a.samples = append(a.samples, frame.Samples...)
}
func (a *p25CallArchive) Save(root string, groupID uint32, started time.Time) (string, error) {
	if len(a.samples) == 0 {
		return "", fmt.Errorf("no P25 PCM received for talkgroup %d", groupID)
	}
	if a.eventID == "" {
		a.eventID = NewID()
	}
	path := filepath.Join(root, "Recordings", started.UTC().Format("2006-01-02"), fmt.Sprintf("%s-p25-tg%d-%s-%s.wav", started.UTC().Format("20060102T150405.000Z"), groupID, a.streamID, a.eventID))
	temporary := path + ".part"
	if err := WriteMonoWAV(temporary, a.samples, a.rate); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	file, err := os.OpenFile(temporary, os.O_RDWR, 0)
	if err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	if syncErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		if syncErr != nil {
			return "", syncErr
		}
		return "", closeErr
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return "", err
	}
	return path, nil
}

func (a *p25CallArchive) DurationSeconds() float64 {
	if a == nil || a.rate <= 0 {
		return 0
	}
	return float64(len(a.samples)) / float64(a.rate)
}
