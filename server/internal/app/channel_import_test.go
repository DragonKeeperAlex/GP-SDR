package app

import (
	"strings"
	"testing"
)

func TestBulkChannelImportAcceptsCHIRPCSV(t *testing.T) {
	store, err := NewProfileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("Location,Name,Frequency,Mode,Duplex,Offset\n1,Local Fire,460.575000,NFM,+,5.000\n2,Air Tower,118.100000,AM,,0\n")
	profile, err := store.ImportChannelCSV("field-bank.csv", data)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "field-bank" || len(profile.Channels) != 2 {
		t.Fatalf("unexpected imported profile: %#v", profile)
	}
	if profile.Channels[0].FrequencyHz != 460_575_000 || profile.Channels[0].Mode != "nfm" {
		t.Fatalf("unexpected first channel: %#v", profile.Channels[0])
	}
	if profile.Channels[1].Mode != "am" {
		t.Fatalf("expected AM channel, got %#v", profile.Channels[1])
	}
}

func TestBulkChannelImportAcceptsTabSeparatedHz(t *testing.T) {
	channels, err := parseChannelCSV([]byte("Alpha Tag\tFrequencyHz\tModulation\nWeather\t162550000\tNFM\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 || channels[0].FrequencyHz != 162_550_000 {
		t.Fatalf("unexpected channels: %#v", channels)
	}
}

func TestBundledLocalChannelCountsAndP25Safety(t *testing.T) {
	profiles := builtInProfiles()
	wantedCounts := map[string]int{
		"San Ramon · Handheld Bank":   137,
		"NorCal · Travel Bank":        199,
		"NorCal · Rubicon Bank":       122,
		"California · GMRS Repeaters": 84,
	}
	for _, profile := range profiles {
		if expected, ok := wantedCounts[profile.Name]; ok {
			if len(profile.Channels) != expected {
				t.Fatalf("%s: got %d channels, want %d", profile.Name, len(profile.Channels), expected)
			}
			delete(wantedCounts, profile.Name)
		}
		if profile.Name == "San Ramon · EBRCS P25" {
			if len(profile.P25Systems) != 2 {
				t.Fatalf("expected two local EBRCS sites, got %d", len(profile.P25Systems))
			}
			for _, system := range profile.P25Systems {
				for _, talkgroup := range system.Talkgroups {
					if strings.Contains(strings.ToLower(talkgroup.Name), "dispatch") && strings.Contains(talkgroup.Name, "SRPD") && !talkgroup.Encrypted {
						t.Fatal("encrypted SRPD dispatch must remain explicitly locked out")
					}
				}
			}
		}
	}
	if len(wantedCounts) != 0 {
		t.Fatalf("missing bundled channel profiles: %#v", wantedCounts)
	}
}
