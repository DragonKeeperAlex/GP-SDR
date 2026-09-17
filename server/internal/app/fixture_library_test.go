package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureLibraryPreservesTruthAndSkipsInvalid(t *testing.T) {
	r := &Runtime{dataDirectory: t.TempDir()}
	list, err := r.FixtureLibrary()
	if err != nil || len(list.Fixtures) != 0 {
		t.Fatalf("empty library: %+v %v", list, err)
	}
	iq, truth, err := generateSignalFixture(SignalFixtureRequest{Kind: "nfm"}, 200000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	truth, err = saveSignalFixture(r.dataDirectory, iq, truth)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(truth.IQPath), "bad.truth.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	list, err = r.FixtureLibrary()
	if err != nil || len(list.Fixtures) != 1 || list.Invalid != 1 || list.Fixtures[0].SHA256 != truth.SHA256 {
		t.Fatalf("library: %+v %v", list, err)
	}
	if list.Fixtures[0].TrainingEligibility == "" {
		t.Fatal("truth provenance lost")
	}
}

func TestTransmitRejectsBusyBeforeGeneration(t *testing.T) {
	r := &Runtime{transmit: newTransmitState(), dataDirectory: t.TempDir()}
	r.transmit.generation.Lock()
	if _, err := r.Transmit(TransmitRequest{}); err == nil {
		t.Fatal("parallel preparation accepted")
	}
	r.transmit.generation.Unlock()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.transmit.cancel = cancel
	if _, err := r.Transmit(TransmitRequest{}); err == nil {
		t.Fatal("active transmission accepted")
	}
	entries, _ := os.ReadDir(r.dataDirectory)
	if len(entries) != 0 {
		t.Fatal("busy request generated files")
	}
}
