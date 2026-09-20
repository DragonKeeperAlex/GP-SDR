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

func TestBuiltInChannelBanksIncludeCommonReceivePlans(t *testing.T) {
	wanted := map[string]int{
		"Railroad · AAR Voice":          91,
		"Public Safety · Interop":       19,
		"Marine VHF · Calling & Safety": 10,
		"Emergency & Calling":           5,
	}
	for _, profile := range builtInProfiles() {
		if count, ok := wanted[profile.Name]; ok {
			if len(profile.Channels) != count {
				t.Fatalf("%s channel count: got %d, want %d", profile.Name, len(profile.Channels), count)
			}
			for _, channel := range profile.Channels {
				if !channel.Enabled || channel.FrequencyHz <= 0 || channel.BandwidthHz <= 0 {
					t.Fatalf("%s contains an unusable channel: %#v", profile.Name, channel)
				}
			}
			delete(wanted, profile.Name)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing common receive profiles: %#v", wanted)
	}
}

func TestGMRSWholeBandFitsHackRFAndRejectsNarrowRTL(t *testing.T) {
	var gmrs ScanProfile
	for _, profile := range builtInProfiles() {
		if profile.ID == "be8e8ba2-ef4d-47f4-875f-f489bc8d894b" {
			gmrs = profile
			break
		}
	}
	hackRFLimit, rtlLimit := 20_000_000.0, 3_200_000.0
	if spec, channels, ok := widebandSpec(gmrs, SDRDevice{Kind: "HackRF", SampleRateLimit: &hackRFLimit}); !ok || spec.SampleRateHz < 8_000_000 || len(channels) != 30 {
		t.Fatalf("GMRS HackRF wideband plan unavailable: spec=%#v channels=%d ok=%v", spec, len(channels), ok)
	}
	if _, _, ok := widebandSpec(gmrs, SDRDevice{Kind: "RTL-SDR", SampleRateLimit: &rtlLimit}); ok {
		t.Fatal("GMRS input/output span must not be presented as a simultaneous RTL-SDR capture")
	}
}

func TestWidebandSpecHonorsExplicitSafeHackRFRate(t *testing.T) {
	limit := 20_000_000.0
	profile := ScanProfile{Channels: []ChannelDefinition{
		{ID: "a", FrequencyHz: 98_100_000, BandwidthHz: 180_000, Mode: "wfm", Enabled: true},
		{ID: "b", FrequencyHz: 98_300_000, BandwidthHz: 180_000, Mode: "wfm", Enabled: true},
	}, Settings: SurveySettings{SampleRateHz: 5_000_000}}
	spec, _, ok := widebandSpec(profile, SDRDevice{Kind: "HackRF", SampleRateLimit: &limit})
	if !ok || spec.SampleRateHz != 5_000_000 {
		t.Fatalf("explicit safe HackRF band-monitor rate = %+v, ok=%v; want 5 MS/s", spec, ok)
	}
}
