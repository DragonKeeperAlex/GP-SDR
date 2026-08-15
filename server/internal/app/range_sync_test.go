package app

import (
	"strings"
	"testing"
)

func TestGoogleSheetCSVURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://docs.google.com/spreadsheets/d/abc123/edit#gid=42", "https://docs.google.com/spreadsheets/d/abc123/export?format=csv&gid=42"},
		{"https://docs.google.com/spreadsheets/d/abc123/edit?usp=sharing", "https://docs.google.com/spreadsheets/d/abc123/export?format=csv&gid=0"},
		{"https://docs.google.com/spreadsheets/d/e/pub123/pubhtml?gid=9", "https://docs.google.com/spreadsheets/d/e/pub123/pub?output=csv&gid=9"},
	}
	for _, test := range tests {
		got, err := googleSheetCSVURL(test.input)
		if err != nil || got != test.want {
			t.Fatalf("googleSheetCSVURL(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
	if _, err := googleSheetCSVURL("http://127.0.0.1/private"); err == nil {
		t.Fatal("non-Google URL was accepted")
	}
	if _, err := googleSheetCSVURL("https://docs.google.com:8443/spreadsheets/d/example/edit"); err == nil {
		t.Fatal("non-standard Google port was accepted")
	}
}

func TestParseRangeSheetCSV(t *testing.T) {
	data := []byte("Profile,Name,Start MHz,End MHz,Step kHz,Mode,Dwell ms,Enabled,Summary\n" +
		"Local,Airband,118,136.975,25,AM,160,yes,Portable receive ranges\n" +
		"Local,Public safety,769 MHz,775 MHz,12.5,digital,180,true,\n" +
		"Travel,Two meter,144000 kHz,148000 kHz,12500 Hz,NFM,200,no,\n")
	profiles, err := parseRangeSheetCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || profiles[0].Name != "Local" || len(profiles[0].Ranges) != 2 {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}
	if profiles[0].Ranges[0].StartHz != 118e6 || profiles[0].Ranges[0].StepHz != 25e3 || profiles[0].Ranges[0].PreferredMode != "am" {
		t.Fatalf("unexpected airband range: %#v", profiles[0].Ranges[0])
	}
	if profiles[1].Ranges[0].Enabled {
		t.Fatal("disabled sheet range was enabled")
	}
	if !profiles[0].BuiltIn || !strings.HasPrefix(profiles[0].ID, "sheet-") {
		t.Fatalf("synced profile is not a built-in sheet profile: %#v", profiles[0])
	}
}

func TestParseRangeSheetUsesDefaults(t *testing.T) {
	profiles, err := parseRangeSheetCSV([]byte("Start,End\n88,108\n"))
	if err != nil {
		t.Fatal(err)
	}
	rangeItem := profiles[0].Ranges[0]
	if rangeItem.Name != "Range 1" || rangeItem.StartHz != 88e6 || rangeItem.StepHz != 12_500 || rangeItem.DwellMilliseconds != 180 {
		t.Fatalf("unexpected defaults: %#v", rangeItem)
	}
}
