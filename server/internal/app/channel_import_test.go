package app

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestRadioReferenceStateCSVHeaderAndModes(t *testing.T) {
	data := []byte("\"Frequency Output\",\"Frequency Input\",Description,\"Alpha Tag\",Mode\n460.575,0,Dispatch,SRM Tac 22,FMN\n148.6625,0,Encrypted,MOTCO MP,P25E\n")
	channels, err := parseChannelCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 || channels[0].FrequencyHz != 460_575_000 || channels[0].Mode != "nfm" {
		t.Fatalf("unexpected RadioReference analog channel: %#v", channels)
	}
	if channels[1].Mode != "digital" || channels[1].Decoder == nil || *channels[1].Decoder != "P25" {
		t.Fatalf("unexpected RadioReference P25 channel: %#v", channels[1])
	}
}

func TestLocalDatabaseSplitsLargeStateCSVIntoStableBanks(t *testing.T) {
	root := t.TempDir()
	var data strings.Builder
	data.WriteString("Frequency Output,Alpha Tag,Mode\n")
	for index := 0; index < 10_005; index++ {
		fmt.Fprintf(&data, "%.6f,Channel %d,FMN\n", 30+float64(index)*.0125, index)
	}
	file := filepath.Join(root, "California.csv")
	if err := os.WriteFile(file, []byte(data.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewProfileStore(filepath.Join(t.TempDir(), "Profiles"))
	if err != nil {
		t.Fatal(err)
	}
	manager := &LocalDatabaseManager{profiles: store, status: LocalDatabaseStatus{Folder: root}}
	first := manager.Scan()
	if first.Profiles != 3 || first.Channels != 10_005 || first.LastError != "" {
		t.Fatalf("unexpected first scan: %+v", first)
	}
	profileCount := len(store.All())
	second := manager.Scan()
	if second.Profiles != 3 || len(store.All()) != profileCount {
		t.Fatalf("rescan duplicated profiles: before=%d after=%d status=%+v", profileCount, len(store.All()), second)
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
