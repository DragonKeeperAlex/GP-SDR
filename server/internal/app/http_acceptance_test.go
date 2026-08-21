package app

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func acceptanceRequest(t *testing.T, server *Server, method, target string, body any, expected int) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	var err error
	if body != nil {
		switch value := body.(type) {
		case string:
			data = []byte(value)
		default:
			data, err = json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(data))
	request.RemoteAddr = "127.0.0.1:54321"
	request.Header.Set("X-GP-SDR-Token", "acceptance")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != expected {
		t.Fatalf("%s %s: expected %d, got %d: %s", method, target, expected, response.Code, response.Body.String())
	}
	return response
}

func decodeAcceptance[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response: %v: %s", err, response.Body.String())
	}
	return value
}

func TestHTTPFeatureSurfaceAcceptance(t *testing.T) {
	runtimeState, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtimeState.Stop)
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}
	var filesystem fs.FS = web
	server := NewServer(runtimeState, filesystem, "127.0.0.1", 8073, "acceptance")

	for _, endpoint := range []string{
		"/api/status", "/api/devices", "/api/decoders", "/api/integrations", "/api/setup",
		"/api/p25/status", "/api/spectrum?bins=64", "/api/calibrations", "/api/remote-receivers",
		"/api/range-sync", "/api/local-database", "/api/mapper", "/api/mapper/progress", "/api/profiles",
		"/api/events?limit=10", "/api/signals?limit=10", "/api/mixer", "/api/receiver-plan",
	} {
		acceptanceRequest(t, server, http.MethodGet, endpoint, nil, http.StatusOK)
	}

	calibration := DeviceCalibration{DeviceID: "simulator-0", DeviceKind: "Simulator", PPMCorrection: 2,
		IQGain: 1.02, IQPhase: .5, DCRemoval: true, LNAGainDB: 16, VGAGainDB: 20, Source: "acceptance"}
	acceptanceRequest(t, server, http.MethodPut, "/api/calibrations", calibration, http.StatusOK)
	calibrations := decodeAcceptance[[]DeviceCalibration](t, acceptanceRequest(t, server, http.MethodGet, "/api/calibrations", nil, http.StatusOK))
	if len(calibrations) != 1 || calibrations[0].DeviceID != "simulator-0" {
		t.Fatalf("calibration was not persisted: %#v", calibrations)
	}
	acceptanceRequest(t, server, http.MethodDelete, "/api/calibrations?deviceID=simulator-0", nil, http.StatusOK)

	remote := RemoteReceiver{ID: "acceptance-rtl-tcp", Name: "Acceptance RTL-TCP", Host: "127.0.0.1", Port: 59999, Enabled: false}
	acceptanceRequest(t, server, http.MethodPut, "/api/remote-receivers", remote, http.StatusOK)
	remotes := decodeAcceptance[[]RemoteReceiver](t, acceptanceRequest(t, server, http.MethodGet, "/api/remote-receivers", nil, http.StatusOK))
	if len(remotes) != 1 || remotes[0].ID != remote.ID {
		t.Fatalf("remote receiver was not persisted: %#v", remotes)
	}
	acceptanceRequest(t, server, http.MethodDelete, "/api/remote-receivers?id="+remote.ID, nil, http.StatusOK)

	rangeConfig := RangeSyncConfig{Enabled: false, SheetURL: "https://docs.google.com/spreadsheets/d/test/edit", IntervalMinutes: 15}
	acceptanceRequest(t, server, http.MethodPut, "/api/range-sync", rangeConfig, http.StatusOK)
	mapperConfig := MapperConfig{Mode: "discovery", DeviceID: "simulator-0", StartHz: 100e6, EndHz: 101e6,
		StepHz: 200e3, DwellMilliseconds: 200, LocationPrecision: "approximate"}
	acceptanceRequest(t, server, http.MethodPut, "/api/mapper", mapperConfig, http.StatusOK)
	export := acceptanceRequest(t, server, http.MethodGet, "/api/mapper/export.csv", nil, http.StatusOK)
	if !bytes.Contains(export.Body.Bytes(), []byte("frequency_hz")) {
		t.Fatal("Mapper CSV header is missing")
	}
	script := acceptanceRequest(t, server, http.MethodGet, "/api/mapper/apps-script.gs", nil, http.StatusOK)
	if !bytes.Contains(script.Body.Bytes(), []byte("Additions Queue")) {
		t.Fatal("Mapper Apps Script does not target the additions queue")
	}
	acceptanceRequest(t, server, http.MethodPost, "/api/mapper/save", map[string]any{}, http.StatusOK)
	acceptanceRequest(t, server, http.MethodPost, "/api/mapper/clear", map[string]any{}, http.StatusOK)

	profiles := decodeAcceptance[[]ScanProfile](t, acceptanceRequest(t, server, http.MethodGet, "/api/profiles", nil, http.StatusOK))
	var fm ScanProfile
	for _, profile := range profiles {
		if profile.Name == "US FM Broadcast · 100 Channels" {
			fm = profile
			break
		}
	}
	if fm.ID == "" {
		t.Fatal("FM acceptance profile is missing")
	}
	duplicate := decodeAcceptance[ScanProfile](t, acceptanceRequest(t, server, http.MethodPost, "/api/profiles/duplicate?id="+fm.ID, map[string]any{}, http.StatusCreated))
	acceptanceRequest(t, server, http.MethodGet, "/api/profiles/export?id="+duplicate.ID, nil, http.StatusOK)
	acceptanceRequest(t, server, http.MethodDelete, "/api/profiles?id="+duplicate.ID, nil, http.StatusOK)
	channels := "Frequency,Name,Mode\n462.5625,Acceptance GMRS,NFM\n100.1,Acceptance FM,WFM\n"
	imported := decodeAcceptance[ScanProfile](t, acceptanceRequest(t, server, http.MethodPost,
		"/api/profiles/import-channels?filename=acceptance.csv", channels, http.StatusCreated))
	if len(imported.Channels) != 2 {
		t.Fatalf("expected two imported channels, got %d", len(imported.Channels))
	}
	acceptanceRequest(t, server, http.MethodDelete, "/api/profiles?id="+imported.ID, nil, http.StatusOK)

	acceptanceRequest(t, server, http.MethodPost, "/api/control/start", map[string]string{"profileID": fm.ID}, http.StatusOK)
	runtimeState.generateDemo()
	status := decodeAcceptance[RuntimeStatus](t, acceptanceRequest(t, server, http.MethodGet, "/api/status", nil, http.StatusOK))
	if !status.Running || status.EventCount == 0 || !status.SimulatorEnabled {
		t.Fatalf("demo survey did not become operational: %#v", status)
	}
	mixer := decodeAcceptance[[]MixerChannel](t, acceptanceRequest(t, server, http.MethodGet, "/api/mixer", nil, http.StatusOK))
	if len(mixer) == 0 {
		t.Fatal("survey mixer is empty")
	}
	muted := true
	volume := .4
	acceptanceRequest(t, server, http.MethodPost, "/api/mixer", map[string]any{"id": mixer[0].ID, "muted": muted, "volume": volume}, http.StatusOK)
	acceptanceRequest(t, server, http.MethodPost, "/api/control/stop", map[string]any{}, http.StatusOK)

	tuner := TunerRequest{DeviceID: "simulator-0", FrequencyHz: 100.1e6, Mode: "wfm", BandwidthHz: 180e3,
		IQGain: 1, IQDCRemoval: true, SquelchDB: 6, NoiseReduction: "voice"}
	acceptanceRequest(t, server, http.MethodPost, "/api/tuner/start", tuner, http.StatusOK)
	status = decodeAcceptance[RuntimeStatus](t, acceptanceRequest(t, server, http.MethodGet, "/api/status", nil, http.StatusOK))
	if status.Mode != "Tuner · WFM" {
		t.Fatalf("unexpected tuner mode: %s", status.Mode)
	}
	acceptanceRequest(t, server, http.MethodPost, "/api/tuner/stop", map[string]any{}, http.StatusOK)
	tuner.GainDB = 99
	acceptanceRequest(t, server, http.MethodPost, "/api/tuner/start", tuner, http.StatusBadRequest)
	acceptanceRequest(t, server, http.MethodPost, "/api/setup/install", map[string]string{"componentID": "does-not-exist"}, http.StatusBadRequest)
	acceptanceRequest(t, server, http.MethodGet, "/api/audio?id=missing", nil, http.StatusNotFound)
}
