package app

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMapperTracksVisitsHitsOccupancyAndLocation(t *testing.T) {
	latitude, longitude := 37.77491, -122.41941
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), config: MapperConfig{
		IncludeLocation: true, LocationPrecision: "approximate", Latitude: &latitude, Longitude: &longitude, LocationLabel: "Field site",
	}}
	manager.Observe(99_700_000, true, -22, -76, "WFM", "Analog FM", "FM broadcast", "")
	manager.Observe(99_700_000, false, -80, -82, "", "", "", "")
	status := manager.Status()
	if len(status.Records) != 1 {
		t.Fatalf("expected one active frequency, got %d", len(status.Records))
	}
	record := status.Records[0]
	if record.Hits != 1 || record.Checks != 2 || math.Abs(record.Occupancy-.5) > .001 {
		t.Fatalf("unexpected occupancy record: %+v", record)
	}
	if record.Location == nil || record.Location.Latitude != 37.77 || record.Location.Longitude != -122.42 {
		t.Fatalf("expected approximate location, got %+v", record.Location)
	}
	if record.HourlyHits[record.LastSeen.Hour()] != 1 || record.ActivityTimeZone == "" {
		t.Fatalf("expected the active hour and time zone to be recorded: %+v", record)
	}
}

func TestMapperProgressTracksFrequencyPassesAndLongListenWindow(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord)}
	sessionID := manager.BeginSession("decipher", 3)
	manager.BeginTarget(sessionID, 1, 3, 162_550_000, "NOAA Weather", 48*time.Hour)
	manager.Observe(162_550_000, true, -24, -76, "NFM", "Analog FM", "NOAA Weather", "")
	manager.SetIdentification(162_550_000, "RadioReference import · Test", .98)
	manager.CompletePass(sessionID)
	progress := manager.Progress()
	if !progress.Running || progress.CurrentFrequencyHz != 162_550_000 || progress.CurrentIndex != 1 || progress.TotalTargets != 3 || progress.ChecksCompleted != 1 || progress.PassesCompleted != 1 {
		t.Fatalf("unexpected mapper progress: %+v", progress)
	}
	if progress.TargetEndsAt == nil || progress.TargetStartedAt == nil || progress.TargetEndsAt.Sub(*progress.TargetStartedAt) != 48*time.Hour {
		t.Fatalf("expected a 48 hour channel listen window: %+v", progress)
	}
	if progress.EstimatedPassEndAt == nil || progress.EstimatedPassEndAt.Sub(*progress.TargetStartedAt) != 96*time.Hour {
		t.Fatalf("expected ETA to include the current and remaining channel: %+v", progress)
	}
	record := manager.Status().Records[0]
	if record.IdentificationSource != "RadioReference import · Test" || record.Confidence != .98 {
		t.Fatalf("unexpected identification evidence: %+v", record)
	}
	manager.EndSession(sessionID)
	if manager.Progress().Running {
		t.Fatal("mapper should report stopped after ending its session")
	}
}

func TestMapperRejectsDecipherListenPeriodsOutsideFiveSecondsToSevenDays(t *testing.T) {
	manager := &MapperManager{path: filepath.Join(t.TempDir(), "mapper.json"), records: make(map[string]MapperFrequencyRecord)}
	for _, seconds := range []int64{4, 7*24*60*60 + 1} {
		_, err := manager.Update(MapperConfig{Mode: "decipher", DecipherListenSeconds: seconds})
		if err == nil {
			t.Fatalf("expected %d seconds to be rejected", seconds)
		}
	}
	if _, err := manager.Update(MapperConfig{Mode: "decipher", DecipherListenSeconds: 5}); err != nil {
		t.Fatalf("five seconds should be accepted: %v", err)
	}
}

func TestMapperAppliesSafeWorkflowConcurrencyDefaultsAndLimits(t *testing.T) {
	manager := &MapperManager{path: filepath.Join(t.TempDir(), "mapper.json"), records: make(map[string]MapperFrequencyRecord)}
	discovery, err := manager.Update(MapperConfig{Mode: "discovery", DecipherListenSeconds: 5})
	if err != nil || discovery.Config.ConcurrentChannels != 16 {
		t.Fatalf("expected Discovery to default to 16 simultaneous channels: status=%+v error=%v", discovery, err)
	}
	identify, err := manager.Update(MapperConfig{Mode: "decipher", DecipherListenSeconds: 5})
	if err != nil || identify.Config.ConcurrentChannels != 4 {
		t.Fatalf("expected Identify to default to 4 simultaneous channels: status=%+v error=%v", identify, err)
	}
	if _, err := manager.Update(MapperConfig{Mode: "discovery", DecipherListenSeconds: 5, ConcurrentChannels: 33}); err == nil {
		t.Fatal("expected more than 32 simultaneous channels to be rejected")
	}
}

func TestMapperIgnoresQuietFrequenciesUntilActivityIsSeen(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord)}
	manager.Observe(123_450_000, false, -82, -84, "", "", "", "")
	if len(manager.Status().Records) != 0 {
		t.Fatal("quiet-only frequencies should not fill the activity map")
	}
}

func TestMapperPrunesOnlyKnownLegacyFalsePositiveSignature(t *testing.T) {
	records := map[string]MapperFrequencyRecord{
		"bad":       {FrequencyHz: 10_000_000, Checks: 1, Hits: 1, NoiseDBFS: 0},
		"measured":  {FrequencyHz: 155_250_000, Checks: 1, Hits: 1, NoiseDBFS: -78},
		"revisited": {FrequencyHz: 162_550_000, Checks: 2, Hits: 2, NoiseDBFS: 0},
	}
	if removed := pruneLegacyMapperFalsePositives(records); removed != 1 {
		t.Fatalf("expected one legacy row removed, got %d", removed)
	}
	if _, exists := records["bad"]; exists || len(records) != 2 {
		t.Fatalf("unexpected records after migration: %#v", records)
	}
}

func TestMapperBandIdentification(t *testing.T) {
	tests := []struct {
		frequency float64
		protocol  string
	}{
		{99_700_000, "Analog FM"},
		{125_000_000, "Aviation AM"},
		{1090_000_000, "ADS-B / Mode S candidate"},
		{774_181_250, "Likely P25"},
	}
	for _, test := range tests {
		_, _, protocol, _ := identifyMappedFrequency(test.frequency)
		if protocol != test.protocol {
			t.Fatalf("%.0f Hz: expected %q, got %q", test.frequency, test.protocol, protocol)
		}
	}
}

func TestMapperIdentificationPrefersRadioReferenceProfileEvidence(t *testing.T) {
	store := &ProfileStore{profiles: map[string]ScanProfile{
		"rr-test": {ID: "rr-test", Name: "County public safety", Summary: "RadioReference location import",
			ReferenceArea: &ProfileReferenceArea{Provider: "RadioReference", Latitude: 37.77, Longitude: -122.42, RadiusMiles: 25, Label: "San Francisco"},
			Channels:      []ChannelDefinition{{Name: "County dispatch", FrequencyHz: 155_250_000, BandwidthHz: 12_500, Mode: "nfm", Enabled: true}}},
	}}
	runtimeState := &Runtime{Profiles: store}
	identity := runtimeState.identifyMapperFrequencyAt(155_250_000, &ObservationLocation{Latitude: 37.80, Longitude: -122.40, Precision: "exact"})
	if identity.Name != "County dispatch" || identity.Mode != "NFM" || identity.Protocol != "Analog FM" || identity.Confidence != .98 || !identity.Verified || !strings.HasPrefix(identity.Source, "RadioReference import") {
		t.Fatalf("unexpected RadioReference identification: %+v", identity)
	}
	far := runtimeState.identifyMapperFrequencyAt(155_250_000, &ObservationLocation{Latitude: 34.05, Longitude: -118.24, Precision: "exact"})
	if far.Verified || strings.HasPrefix(far.Source, "RadioReference import") {
		t.Fatalf("distant same-frequency RadioReference entry must be rejected: %+v", far)
	}
}

func TestMapperVerifiedStatusRequiresAuthoritativeEvidence(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time)}
	manager.Observe(155_250_000, true, -31, -76, "NFM", "Analog FM", "County dispatch", "")
	manager.SetIdentification(155_250_000, "Built-in US band plan", .99)
	if manager.Status().VerifiedRecords != 0 {
		t.Fatal("a high-confidence band guess must not count as fully identified")
	}
	distance := 3.2
	manager.SetIdentificationEvidence(155_250_000, "RadioReference import · Local", .98, true, "nearby reference match", &distance)
	status := manager.Status()
	if status.VerifiedRecords != 1 || !mapperRecordFullyIdentified(status.Records[0]) {
		t.Fatalf("authoritative match should count as fully identified: %+v", status)
	}
}

func TestMapperValidDecoderFramesAreFullyIdentified(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time)}
	manager.Observe(1090_000_000, true, -25, -70, "DIGITAL", "", "", "")
	manager.SetDecodedMessages(1090_000_000, "dump1090", []DecoderMessage{{Protocol: "ADS-B", Summary: "valid Mode S frame", Confidence: .99}})
	record := manager.Status().Records[0]
	if !mapperRecordFullyIdentified(record) || record.DetectionStatus != "confirmed" {
		t.Fatalf("valid decoder output should verify the identity: %+v", record)
	}
}

func TestMapperCSVIncludesCompleteRecordsAndEscapesFormulas(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord)}
	manager.Observe(155_250_000, true, -31.25, -76.5, "NFM", "Analog FM", "=unsafe", "+transcript")
	data, rows, err := manager.CSV()
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("expected one exported row, got %d", rows)
	}
	parsed, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 || parsed[1][2] != "'=unsafe" || parsed[1][21] != "'+transcript" {
		t.Fatalf("unexpected CSV output: %#v", parsed)
	}
}

func TestDecoderTargetsAddProfilesAndMapperCandidateEvidence(t *testing.T) {
	profiles := decoderScanProfiles()
	if len(profiles) != 7 {
		t.Fatalf("expected seven decoder scan profiles, got %d", len(profiles))
	}
	for _, id := range []string{"dsd-fme", "rtl-433", "dump1090", "multimon-ng", "acarsdec", "ais"} {
		found := false
		for _, profile := range profiles {
			for _, channel := range profile.Channels {
				found = found || channel.Decoder != nil && *channel.Decoder == id
			}
			for _, scanRange := range profile.Ranges {
				found = found || scanRange.Decoder != nil && *scanRange.Decoder == id
			}
		}
		if !found {
			t.Fatalf("decoder %s has no scan target", id)
		}
	}

	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord)}
	manager.Observe(1090e6, true, -30, -75, "DIGITAL", "ADS-B / Mode S candidate", "Aircraft transponder", "")
	manager.SetDecoderEvidence(1090e6, "dump1090", "candidate", "RF activity near 1090 MHz", true)
	record := manager.Status().Records[0]
	if record.CandidateDecoder != "dump1090" || record.DetectionStatus != "candidate" || !record.DecoderReady {
		t.Fatalf("unexpected decoder evidence: %+v", record)
	}
}

func TestMapperAcceptsForcedDMRDecoder(t *testing.T) {
	config, err := validateMapperScanConfig(MapperConfig{Mode: "discovery", DeviceID: "receiver", StartHz: 450e6, EndHz: 451e6,
		StepHz: 12_500, DwellMilliseconds: 2500, PreferredMode: "dmr", PreferredDecoder: "dmr", DecipherListenSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	if config.PreferredDecoder != "dmr" || canonicalDecoderID(config.PreferredDecoder) != "dsd-fme" {
		t.Fatalf("DMR decoder was not preserved: %#v", config)
	}
}

func TestMapperSaveCSVUsesDataExportsFolder(t *testing.T) {
	dataDirectory := t.TempDir()
	manager := &MapperManager{recordsPath: filepath.Join(dataDirectory, "Data", "mapper-records.json"), records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time)}
	manager.Observe(162_550_000, true, -20, -70, "NFM", "Analog FM", "NOAA Weather", "")
	result, err := manager.SaveCSV()
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows != 1 || filepath.Dir(result.Path) != filepath.Join(dataDirectory, "Exports", "Mapper") {
		t.Fatalf("unexpected export result: %+v", result)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatal(err)
	}
}

func TestMapperUploadMatchesMasterAdditionsQueueSchema(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ok":true,"added":1}`))
	}))
	defer server.Close()
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time), client: server.Client(), config: MapperConfig{
		WebhookURL: server.URL, SheetURL: "https://docs.google.com/spreadsheets/d/test/edit", Contributor: "Field receiver", Secret: "test-secret",
	}}
	manager.Observe(774_181_250, true, -34, -79, "DIGITAL", "P25 Phase 1", "EBRCS control channel", "")
	status := manager.UploadFrequency(774_181_250)
	if status.LastError != "" || status.UploadedRows != 1 {
		t.Fatalf("unexpected upload status: %+v", status)
	}
	if payload["sheetName"] != "Additions Queue" || payload["secret"] != "test-secret" {
		t.Fatalf("wrong sheet routing: %#v", payload)
	}
	rows, ok := payload["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("missing additions rows: %#v", payload)
	}
	row, ok := rows[0].(map[string]any)
	if !ok || row["type"] != "P25 System" || row["reviewStatus"] != "New" || row["confidence"] != "Heard once" || row["rxMHz"] != 774.18125 {
		t.Fatalf("row does not match master schema: %#v", row)
	}
}

func TestMapperUploadRequiresConfiguredWebhook(t *testing.T) {
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time)}
	status := manager.UploadNow()
	if !strings.Contains(status.LastError, "webhook URL") {
		t.Fatalf("expected actionable setup error, got %+v", status)
	}
}

func TestMapperIdentifiedOnlyUploadFiltersUnverifiedRows(t *testing.T) {
	var received int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var payload struct {
			Signals []MapperFrequencyRecord `json:"signals"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		received = len(payload.Signals)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	manager := &MapperManager{records: make(map[string]MapperFrequencyRecord), lastSeen: make(map[string]time.Time), client: server.Client(),
		config: MapperConfig{WebhookURL: server.URL, UploadVerifiedOnly: true}}
	manager.Observe(155_250_000, true, -30, -75, "NFM", "Analog FM", "Unverified channel", "")
	manager.Observe(162_550_000, true, -25, -70, "NFM", "Analog FM", "NOAA Weather", "")
	manager.SetIdentificationEvidence(162_550_000, "RadioReference import · Local", .98, true, "nearby reference match", nil)
	status := manager.UploadNow()
	if status.LastError != "" || received != 1 || status.UploadedRows != 1 {
		t.Fatalf("identified-only upload sent the wrong rows: received=%d status=%+v", received, status)
	}
	status = manager.UploadFrequency(155_250_000)
	if !strings.Contains(status.LastError, "not fully identified") {
		t.Fatalf("single-row upload should enforce the same filter: %+v", status)
	}
}
