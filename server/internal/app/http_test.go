package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestPasswordSignInAuthorizesBrowserSession(t *testing.T) {
	runtimeState, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := NewPasswordHash("field-console-password")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtimeState, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, "127.0.0.1", 8073, "token")
	if err := server.SetPasswordAuthentication(hash, true); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]string{"password": "field-console-password"})
	login := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	login.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, login)
	if response.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("expected a secure HttpOnly session cookie, got %#v", cookies)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("password session status = %d: %s", response.Code, response.Body.String())
	}
}

func TestPasswordSignInRequiresHTTPS(t *testing.T) {
	runtimeState, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := NewPasswordHash("field-console-password")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtimeState, fstest.MapFS{}, "127.0.0.1", 8073, "token")
	if err := server.SetPasswordAuthentication(hash, false); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"password":"field-console-password"}`))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("HTTP password login = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestP25MP3RecordingPlayback(t *testing.T) {
	root := t.TempDir()
	runtimeState, err := NewRuntime(root, "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "Recordings", "call.mp3")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mp3-fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runtimeState.Events.Append(TransmissionEvent{ID: "p25-mp3", StartedAt: time.Now(), AudioPath: &path}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtimeState, fstest.MapFS{}, "127.0.0.1", 8073, "secret")
	request := httptest.NewRequest(http.MethodGet, "/api/audio?id=p25-mp3", nil)
	response := httptest.NewRecorder()
	server.serveEventAudio(response, request)
	if response.Code != 200 || response.Header().Get("Content-Type") != "audio/mpeg" || response.Body.String() != "mp3-fixture" {
		t.Fatalf("MP3 playback rejected: %d %s", response.Code, response.Body.String())
	}
}

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

func TestLoopbackKioskDoesNotNeedLANToken(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtime, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, "127.0.0.1", 8073, "secret")
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.RemoteAddr = "127.0.0.1:43210"
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("loopback kiosk request = %d, want %d", response.Code, http.StatusOK)
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

func TestServerServesStaticUIWhileRuntimeStarts(t *testing.T) {
	server := NewServer(nil, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ready")}}, "127.0.0.1", 8073, "")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || response.Body.String() != "ready" {
		t.Fatalf("static UI response = %d %q, want immediate UI", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("startup API response = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	server.SetRuntime(runtime)
	response = httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready API response = %d, want %d", response.Code, http.StatusOK)
	}
}
