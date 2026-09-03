package app

import (
	"fmt"
	"testing"
	"time"
)

func TestInvalidAnalysisConcurrencyDoesNotStartRun(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Events: store}
	for _, concurrency := range []int{-1, 17} {
		if _, err := runtime.StartDeferredAnalysis("", concurrency); err == nil {
			t.Fatal("invalid concurrency accepted")
		}
		if runtime.DeferredAnalysisStatus().Running {
			t.Fatal("invalid request left analysis running")
		}
	}
}

func TestGroupDeferredEventsCombinesFrequencyWithinReceiveArea(t *testing.T) {
	now := time.Now()
	events := []TransmissionEvent{
		{ID: "a", FrequencyHz: 462_562_500, StartedAt: now.Add(time.Second), Location: &ObservationLocation{Latitude: 37.70, Longitude: -121.76, Label: "Livermore"}},
		{ID: "b", FrequencyHz: 462_562_500, StartedAt: now, Location: &ObservationLocation{Latitude: 37.72, Longitude: -121.75, Label: "Livermore"}},
		{ID: "c", FrequencyHz: 462_562_500, StartedAt: now, Location: &ObservationLocation{Latitude: 34.05, Longitude: -118.24, Label: "Los Angeles"}},
		{ID: "d", FrequencyHz: 467_562_500, StartedAt: now, Location: &ObservationLocation{Latitude: 37.70, Longitude: -121.76, Label: "Livermore"}},
	}
	groups := groupDeferredEvents(events)
	if len(groups) != 3 {
		t.Fatalf("expected 3 frequency/location groups, got %d: %#v", len(groups), groups)
	}
	foundPair := false
	for _, group := range groups {
		if len(group) == 2 {
			foundPair = true
			if group[0].ID != "b" || group[1].ID != "a" {
				t.Fatalf("group was not processed oldest-first: %#v", group)
			}
		}
	}
	if !foundPair {
		t.Fatal("same-frequency captures from the same receive area were not combined")
	}
}

func TestDeferredAnalysisRunsConfiguredParallelGroups(t *testing.T) {
	store, err := NewEventStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 6; index++ {
		event := TransmissionEvent{ID: fmt.Sprintf("parallel-%d", index), StartedAt: time.Now().Add(time.Duration(index) * time.Millisecond), FrequencyHz: 150e6 + float64(index)*12500, AnalysisPolicy: "manual", AnalysisStatus: "pending"}
		if err := store.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	runtime := &Runtime{Events: store}
	status, err := runtime.StartDeferredAnalysis("", 3)
	if err != nil {
		t.Fatal(err)
	}
	if status.Concurrency != 3 {
		t.Fatalf("concurrency = %d, want 3", status.Concurrency)
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.DeferredAnalysisStatus().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status = runtime.DeferredAnalysisStatus()
	if status.Running || status.Completed != 6 || status.Failed != 0 || status.GroupsDone != 6 {
		t.Fatalf("unexpected completed status: %#v", status)
	}
}
