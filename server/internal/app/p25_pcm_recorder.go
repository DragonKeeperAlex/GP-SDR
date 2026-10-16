package app

import (
	"strconv"
	"strings"
	"time"
)

type p25PCMCall struct {
	metadataKey string
	call        P25ActiveCall
	archive     *p25CallArchive
}

// One recorder belongs to one monitor session. OP25's ordered stream markers
// separate PCM segments; polling still supplies talkgroup metadata, not native
// per-packet call IDs. Do not describe this adapter as sample-level attribution.
type p25PCMRecorder struct {
	runtime      *Runtime
	profile      ScanProfile
	stop         <-chan struct{}
	calls        map[int]P25ActiveCall
	recordings   map[int]*p25PCMCall
	lastSequence map[int]uint64
	nextIssues   map[int][]string
}

func newP25PCMRecorder(r *Runtime, profile ScanProfile, stop <-chan struct{}) *p25PCMRecorder {
	return &p25PCMRecorder{runtime: r, profile: profile, stop: stop, calls: make(map[int]P25ActiveCall), recordings: make(map[int]*p25PCMCall), lastSequence: make(map[int]uint64), nextIssues: make(map[int][]string)}
}

func (p *p25PCMRecorder) setCalls(calls []P25ActiveCall) {
	current := make(map[int]P25ActiveCall)
	for _, call := range calls {
		if call.Grant.Encrypted || !call.Following || call.Grant.GroupID == 0 {
			continue
		}
		current[call.StreamIndex] = call
	}
	for index, recording := range p.recordings {
		call, ok := current[index]
		if !ok || recording.metadataKey != p25CallKey(call) {
			p.finish(index)
		} else {
			recording.call.LastHeardAt = call.LastHeardAt
		}
	}
	p.calls = current
}

func (p *p25PCMRecorder) accept(frame AudioFrame) {
	if frame.SampleRate != 8000 || !strings.HasPrefix(frame.ChannelID, "p25-stream-") || (frame.Control != "" && frame.Control != "drain" && frame.Control != "drop") {
		return
	}
	index, err := strconv.Atoi(strings.TrimPrefix(frame.ChannelID, "p25-stream-"))
	if err != nil || index < 0 {
		return
	}
	if frame.Sequence != 0 {
		last := p.lastSequence[index]
		if last != 0 && frame.Sequence != last+1 && frame.Control != "drop" {
			issue := "Audio interrupted by a recording queue gap"
			if recording := p.recordings[index]; recording != nil {
				recording.archive.issues = append(recording.archive.issues, issue)
				p.finish(index)
			}
			p.nextIssues[index] = []string{issue}
		}
		p.lastSequence[index] = frame.Sequence
	}
	switch frame.Control {
	case "drain":
		p.finish(index)
		return
	case "drop":
		// DROP is a decoder instruction to discard queued PCM, not an end
		// marker for an archive. Never publish or transcribe that segment.
		if recording := p.recordings[index]; recording != nil {
			if recording.archive.spool != nil {
				if err := recording.archive.spool.discard(); err != nil {
					p.runtime.setRuntimeError("Discard P25 checkpoint: " + err.Error())
				}
			}
			delete(p.recordings, index)
		}
		delete(p.nextIssues, index)
		return
	case "":
	default:
		return
	}
	if len(frame.Samples) == 0 {
		return
	}
	call, ok := p.calls[index]
	at := frame.ReceivedAt
	if at.IsZero() {
		at = time.Now()
	}
	if !ok || at.Sub(call.LastHeardAt) > 2*time.Second || call.LastHeardAt.After(at.Add(time.Second)) {
		return
	}
	recording := p.recordings[index]
	if recording == nil {
		key := p25CallKey(call)
		// Start at received PCM, rather than the first observed assignment.
		call.StartedAt = at.Add(-time.Duration(len(frame.Samples)) * time.Second / 8000)
		archive := newP25CallArchive(index)
		archive.issues = append(archive.issues, p.nextIssues[index]...)
		delete(p.nextIssues, index)
		event := p.runtime.p25CallEvent(call)
		event.MediaIssues = append(event.MediaIssues, archive.issues...)
		event.P25AudioIncomplete = len(archive.issues) != 0
		if err := archive.beginSpool(p.runtime.dataDirectory, event, p.profile.Settings.TranscribeVoice); err != nil {
			p.runtime.setRuntimeError("P25 audio checkpoint unavailable: " + err.Error())
		}
		recording = &p25PCMCall{metadataKey: key, call: call, archive: archive}
		p.recordings[index] = recording
	}
	recording.archive.Append(frame)
}

func (p *p25PCMRecorder) checkpoint() {
	for _, recording := range p.recordings {
		if err := recording.archive.checkpoint(); err != nil {
			p.runtime.setRuntimeError("P25 audio checkpoint: " + err.Error())
		}
	}
}

func (p *p25PCMRecorder) finish(index int) {
	if recording := p.recordings[index]; recording != nil {
		p.runtime.finishP25Call(p.stop, p.profile, recording.call, recording.archive)
		delete(p.recordings, index)
	}
}

func (p *p25PCMRecorder) finishAll() {
	for index := range p.recordings {
		p.finish(index)
	}
}
