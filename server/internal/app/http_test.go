package app

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
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

func TestLegacyAPITokenHeaderRemainsCompatible(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), "http://127.0.0.1:8073/", false)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(runtime, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}}, "127.0.0.1", 8073, "secret")
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	request.Header.Set("X-SignalHarbor-Token", "secret")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected legacy header to remain compatible, got %d", response.Code)
	}
}
