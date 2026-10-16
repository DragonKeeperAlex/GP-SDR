package app

import (
	"fmt"
	"strings"
)

func (r *Runtime) p25CallEvent(call P25ActiveCall) TransmissionEvent {
	label := fmt.Sprintf("Talkgroup %d", call.Grant.GroupID)
	if call.Talkgroup != nil && call.Talkgroup.AlphaTag != "" {
		label = call.Talkgroup.AlphaTag
	}
	protocol := "P25"
	if strings.Contains(strings.ReplaceAll(strings.ToLower(call.Grant.Protocol), " ", ""), "phase2") {
		protocol = "P25 Phase 2"
	}
	duration := call.LastHeardAt.Sub(call.StartedAt).Seconds()
	if duration < 0 {
		duration = 0
	}
	systemName, talkgroupID := call.Grant.System, call.Grant.GroupID
	var sourceRadioID *uint32
	if call.Grant.SourceID != 0 {
		source := call.Grant.SourceID
		sourceRadioID = &source
	}
	decoderEngine := "OP25"
	if r.op25 != nil {
		r.op25.mu.Lock()
		decoderEngine = r.op25.engine
		r.op25.mu.Unlock()
	}
	event := TransmissionEvent{ID: NewID(), P25StreamIndex: ptr(call.StreamIndex), P25TDMASlot: call.TDMASlot, StartedAt: call.StartedAt, DurationSeconds: duration,
		FrequencyHz: float64(call.Grant.FrequencyHz), BandwidthHz: 12_500, SignalDBFS: -30, NoiseDBFS: -80,
		Modulation: "Digital", ProtocolName: &protocol, Label: &label, DeviceID: call.DeviceSerial, Confidence: .99,
		SystemName: &systemName, TalkgroupID: &talkgroupID, SourceRadioID: sourceRadioID, Encrypted: call.Grant.Encrypted,
		Analysis: &SignalIntelligence{Engine: decoderEngine + " frame decoder", Modulation: "DIGITAL", SignalFamily: protocol, Confidence: .99,
			Evidence: []string{fmt.Sprintf("Talkgroup %d", call.Grant.GroupID), fmt.Sprintf("Source radio %d", call.Grant.SourceID)}, Summary: decoderEngine + " reported a " + protocol + " voice call."}}
	return event
}

// Persist a call once, with the saved PCM duration rather than polling duration.
func (r *Runtime) finishP25Call(stop <-chan struct{}, profile ScanProfile, call P25ActiveCall, archive *p25CallArchive) {
	event := r.p25CallEvent(call)
	if archive != nil && !call.Grant.Encrypted {
		if archive.eventID == "" {
			archive.eventID = event.ID
		}
		event.ID = archive.eventID
		if err := archive.checkpoint(); err != nil {
			r.setRuntimeError("P25 final checkpoint: " + err.Error())
		}
		if path, err := archive.Save(r.dataDirectory, call.Grant.GroupID, call.StartedAt); err == nil {
			event.AudioPath = &path
			event.DurationSeconds = archive.DurationSeconds()
			if archive.truncated {
				event.MediaIssues = append(event.MediaIssues, "Recording limited to 180 seconds")
			}
		} else {
			event.MediaIssues = append(event.MediaIssues, "P25 recording unavailable: "+err.Error())
			r.setRuntimeError("P25 recording: " + err.Error())
		}
	}
	if err := r.Events.Append(event); err != nil {
		r.setRuntimeError("Save P25 call metadata: " + err.Error())
		return
	}
	// Keep checkpoints until the audio, journal and optional transcription job
	// are durable. Startup can safely retry with this same event identity.
	queued := true
	if event.AudioPath != nil && profile.Settings.TranscribeVoice {
		queued = r.p25Transcription != nil
		if r.p25Transcription != nil {
			if err := r.p25Transcription.enqueue(event.ID, false); err != nil {
				queued = false
				r.setRuntimeError("Queue P25 transcription: " + err.Error())
			}
		}
	} else if archive == nil && !call.Grant.Encrypted && profile.Settings.P25ArchiveCalls && event.Analysis.Engine != "OP25 frame decoder" {
		go r.attachP25Recording(stop, profile, call, event.ID)
	}
	if archive != nil && archive.spool != nil && (event.AudioPath != nil || len(archive.samples) == 0) && queued {
		if err := archive.spool.remove(); err != nil {
			r.setRuntimeError("P25 checkpoint cleanup: " + err.Error())
		}
	}
}
