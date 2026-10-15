package app

import (
	"reflect"
	"testing"
	"time"
)

func TestRelaySegmentsUseTwentySecondWindows(t *testing.T) {
	got := RelaySegments(65*time.Second, 0)
	want := []time.Duration{20 * time.Second, 20 * time.Second, 20 * time.Second, 5 * time.Second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("segments = %v, want %v", got, want)
	}
}

func TestRelaySegmentsEmptyForNoAudio(t *testing.T) {
	if got := RelaySegments(0, time.Second); got != nil {
		t.Fatalf("segments = %v, want nil", got)
	}
}
