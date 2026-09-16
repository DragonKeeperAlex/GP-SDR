package app

import (
	"encoding/binary"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"
)

func TestAPIToken(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}
	var filesystem fs.FS = web
	server := NewServer(runtime, filesystem, "127.0.0.1", 8073, "secret")

	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.Header.Set("X-GP-SDR-Token", "secret")
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestLiveAudioEndpointStreamsFramedPCM(t *testing.T) {
	runtimeState, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runtimeState.Stop)
	server := NewServer(runtimeState, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, "127.0.0.1", 8073, "secret")
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	request, err := http.NewRequest(http.MethodGet, httpServer.URL+"/api/live-audio", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-GP-SDR-Token", "secret")
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	runtimeState.audioHub.Publish(AudioFrame{ChannelID: "test", SampleRate: 48_000, Samples: []int16{1, -2, 3}})
	var idLength uint16
	var sampleRate, count uint32
	if err := binary.Read(response.Body, binary.LittleEndian, &idLength); err != nil {
		t.Fatal(err)
	}
	if err := binary.Read(response.Body, binary.LittleEndian, &sampleRate); err != nil {
		t.Fatal(err)
	}
	if err := binary.Read(response.Body, binary.LittleEndian, &count); err != nil {
		t.Fatal(err)
	}
	identifier := make([]byte, idLength)
	if _, err := io.ReadFull(response.Body, identifier); err != nil {
		t.Fatal(err)
	}
	samples := make([]int16, count)
	if err := binary.Read(response.Body, binary.LittleEndian, samples); err != nil {
		t.Fatal(err)
	}
	if string(identifier) != "test" || sampleRate != 48_000 || len(samples) != 3 || samples[1] != -2 {
		t.Fatalf("unexpected live audio frame: %q %d %#v", identifier, sampleRate, samples)
	}
}

func TestLegacyAPITokenHeaderRemainsCompatible(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtime, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, "127.0.0.1", 8073, "secret")
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.Header.Set("X-GP-SDR-Token", "secret")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected legacy header to remain compatible, got %d", response.Code)
	}
}
