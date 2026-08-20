package app

import (
	"math"
	"testing"
)

func TestUSFMBroadcastPresetHasEveryChannel(t *testing.T) {
	for _, profile := range builtInProfiles() {
		if profile.ID != "0836ac3e-d346-4d63-8fd2-17dddf3b5b68" {
			continue
		}
		if len(profile.Channels) != 100 {
			t.Fatalf("FM channel count: %d", len(profile.Channels))
		}
		if math.Abs(profile.Channels[0].FrequencyHz-88_100_000) > 1 || math.Abs(profile.Channels[99].FrequencyHz-107_900_000) > 1 {
			t.Fatalf("FM endpoints: %.0f %.0f", profile.Channels[0].FrequencyHz, profile.Channels[99].FrequencyHz)
		}
		for _, channel := range profile.Channels {
			if channel.Mode != "wfm" || channel.BandwidthHz != 180_000 {
				t.Fatalf("invalid FM channel: %#v", channel)
			}
		}
		return
	}
	t.Fatal("US FM broadcast profile missing")
}

func TestBuiltInScanRangesFitHackRFBandwidth(t *testing.T) {
	for _, profile := range builtInProfiles() {
		for _, scanRange := range profile.Ranges {
			if width := scanRange.EndHz - scanRange.StartHz; width > 20_000_000+1 {
				t.Fatalf("%s / %s spans %.3f MHz", profile.Name, scanRange.Name, width/1e6)
			}
		}
	}
}
