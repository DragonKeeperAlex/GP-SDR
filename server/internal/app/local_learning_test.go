package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestConfirmedLearningSaveIsAtomicAndReloadable(t *testing.T) {
	directory := t.TempDir()
	library := NewSignalLearningLibrary(directory)
	base := TransmissionEvent{ID: "atomic-1", FrequencyHz: 145.5e6, BandwidthHz: 12.5e3, Modulation: "NFM"}
	if _, err := library.Confirm(base, "NFM", "Analog voice", "first", false); err != nil {
		t.Fatal(err)
	}
	updated := base
	updated.Transcript = ptr("updated evidence")
	if _, err := library.Confirm(updated, "NFM", "Analog voice", "updated", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "Data", "confirmed-signal-samples.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var samples []ConfirmedSignalSample
	if err := json.Unmarshal(data, &samples); err != nil {
		t.Fatalf("canonical learning snapshot is invalid: %v", err)
	}
	if len(samples) != 1 || samples[0].Notes != "updated" || samples[0].Transcript != "updated evidence" {
		t.Fatalf("unexpected persisted update: %+v", samples)
	}
	if matches, err := filepath.Glob(filepath.Join(directory, "Data", ".gpsdr-write-*.tmp")); err != nil {
		t.Fatal(err)
	} else if len(matches) != 0 {
		t.Fatal("atomic temporary file was left behind")
	}
}

func TestConfirmMapperLearningCreatesGroundedSample(t *testing.T) {
	directory := t.TempDir()
	runtimeState := &Runtime{learning: NewSignalLearningLibrary(directory), mapper: &MapperManager{records: map[string]MapperFrequencyRecord{
		"155250000": {FrequencyHz: 155_250_000, LastSeen: time.Now(), StrongestDBFS: -42, NoiseDBFS: -91, Modulation: "NFM", ProtocolName: "Analog voice", Hits: 4, Checks: 10},
	}}}
	sample, err := runtimeState.ConfirmMapperLearning(155_250_000, "NFM", "Known campus channel", "verified by monitored audio")
	if err != nil {
		t.Fatal(err)
	}
	if sample.FrequencyHz != 155_250_000 || sample.Protocol != "Known campus channel" || runtimeState.learning.Status().Count != 1 {
		t.Fatalf("unexpected confirmed Mapper sample: %#v", sample)
	}
}

func TestLearningLibraryRejectsSimulatedEvidence(t *testing.T) {
	library := NewSignalLearningLibrary(t.TempDir())
	if _, err := library.Confirm(TransmissionEvent{ID: "demo", Simulated: true}, "NFM", "Voice", "", false); err == nil {
		t.Fatal("simulated evidence was accepted")
	}
}

func TestRFTrainingProducesPersistedModelAndHeldOutReport(t *testing.T) {
	directory := t.TempDir()
	library := NewSignalLearningLibrary(directory)
	for index, modulation := range []string{"NFM", "NFM", "WFM", "WFM"} {
		event := TransmissionEvent{ID: fmt.Sprintf("rf-%d", index), FrequencyHz: 100e6 + float64(index)*50e6, BandwidthHz: 12_500, Modulation: modulation,
			SignalDBFS: -40, NoiseDBFS: -80, Analysis: &SignalIntelligence{Modulation: modulation, SignalFamily: "Analog frequency", Confidence: .8, EstimatedDeviationHz: map[string]float64{"NFM": 3_000, "WFM": 60_000}[modulation], AmplitudeVariation: float64(index) * .01, PhaseClusterScore: .2}}
		if _, err := library.Confirm(event, modulation, modulation+" voice", "reviewed", false); err != nil {
			t.Fatal(err)
		}
	}
	report, err := library.TrainAndSave(directory)
	if err != nil {
		t.Fatal(err)
	}
	if report.TrainCount != 4 || report.HeldOutCount != 4 || len(report.Classes) != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if _, err := os.Stat(filepath.Join(directory, "Data", "rf-classifier.json")); err != nil {
		t.Fatal(err)
	}
	if loadRFClassifier(directory) == nil {
		t.Fatal("trained classifier did not reload")
	}
}

func TestLearningLabelCorrectionPreservesSampleIdentity(t *testing.T) {
	directory := t.TempDir()
	library := NewSignalLearningLibrary(directory)
	sample, err := library.Confirm(TransmissionEvent{ID: "correct-me", Modulation: "NFM"}, "NFM", "Unknown voice", "initial", false)
	if err != nil {
		t.Fatal(err)
	}
	corrected, err := library.CorrectLabel(sample.ID, "WFM", "Broadcast FM", "review correction")
	if err != nil || corrected.ID != sample.ID || corrected.Modulation != "WFM" || !corrected.LabelCorrected || corrected.ReviewStatus != "corrected" {
		t.Fatalf("unexpected correction: %#v %v", corrected, err)
	}
}
