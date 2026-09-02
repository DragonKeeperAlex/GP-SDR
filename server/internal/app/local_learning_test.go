package app

import (
	"strings"
	"testing"
)

func TestConfirmedLearningLibraryPersistsAndExports(t *testing.T) {
	directory := t.TempDir()
	library := NewSignalLearningLibrary(directory)
	event := TransmissionEvent{ID: "event-1", FrequencyHz: 462.55e6, BandwidthHz: 20e3, Modulation: "NFM", SignalDBFS: -42, NoiseDBFS: -71,
		Transcript: ptr("K6ABC radio check"), Callsigns: []string{"K6ABC"}, Analysis: &SignalIntelligence{SignalFamily: "Analog frequency", Confidence: .8}}
	sample, err := library.Confirm(event, "NFM", "Analog voice", "confirmed by clear audio", false)
	if err != nil {
		t.Fatal(err)
	}
	if sample.Protocol != "Analog voice" || library.Status().Count != 1 {
		t.Fatalf("unexpected sample: %+v", sample)
	}
	reloaded := NewSignalLearningLibrary(directory)
	if reloaded.Status().Count != 1 || len(reloaded.Similar(event, 5)) != 1 {
		t.Fatal("confirmed sample did not persist")
	}
	export := string(reloaded.ExportJSONL())
	if !strings.Contains(export, `"protocol":"Analog voice"`) || strings.Contains(export, "iqPath") {
		t.Fatalf("unexpected training export: %s", export)
	}
}

func TestLearningLibraryRejectsSimulatedEvidence(t *testing.T) {
	library := NewSignalLearningLibrary(t.TempDir())
	if _, err := library.Confirm(TransmissionEvent{ID: "demo", Simulated: true}, "NFM", "Voice", "", false); err == nil {
		t.Fatal("simulated evidence was accepted")
	}
}
