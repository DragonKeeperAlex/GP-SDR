package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSignalFixtureIsDeterministicAndStoresTruth(t *testing.T) {
	request := SignalFixtureRequest{Kind: "qpsk", Payload: "GP-SDR", SymbolRate: 4800, SNRDB: 30, FrequencyOffsetHz: 125, IQGainError: .03}
	first, manifest, err := generateSignalFixture(request, 200_000, 20_000)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := generateSignalFixture(request, 200_000, 20_000)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("fixture output is not deterministic")
	}
	if manifest.Kind != "QPSK" || manifest.PayloadUTF8 != "GP-SDR" || manifest.ExpectedSymbols == 0 || manifest.OccupiedBandwidthHz <= 0 {
		t.Fatalf("incomplete truth manifest: %+v", manifest)
	}
	stored, err := saveSignalFixture(t.TempDir(), first, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SHA256 == "" {
		t.Fatal("missing fixture checksum")
	}
	if _, err := os.Stat(stored.IQPath); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(stored.IQPath), "*.truth.json"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("truth manifest not stored: %v %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var decoded SignalFixtureManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SHA256 != stored.SHA256 || decoded.DecodeStatus == "" {
		t.Fatalf("stored truth changed: %+v", decoded)
	}
}

func TestSignalFixtureSupportsExpectedFamilies(t *testing.T) {
	for _, kind := range []string{"cw", "am", "nfm", "wfm", "ook", "2fsk", "gfsk", "gmsk", "bpsk", "qpsk"} {
		iq, manifest, err := generateSignalFixture(SignalFixtureRequest{Kind: kind, Payload: "T", SNRDB: 40}, 100_000, 1000)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if len(iq) != 2000 || manifest.Kind == "" {
			t.Fatalf("%s fixture malformed", kind)
		}
	}
}
