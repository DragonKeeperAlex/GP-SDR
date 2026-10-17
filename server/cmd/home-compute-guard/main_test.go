package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPausePolicy(t *testing.T) {
	for _, tc := range []struct {
		name           string
		a              activity
		manual, paused bool
	}{
		{"idle", activity{Idle: 11 * time.Minute}, false, false},
		{"input", activity{Idle: time.Second}, false, true},
		{"game", activity{Idle: time.Hour, Fullscreen: true}, false, true},
		{"controller", activity{Idle: time.Hour, Controller: true}, false, true},
		{"manual", activity{Idle: time.Hour}, true, true},
		{"unknown", activity{Err: io.ErrUnexpectedEOF}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pauseReason(tc.a, tc.manual, 10*time.Minute) != ""; got != tc.paused {
				t.Fatalf("paused=%v", got)
			}
		})
	}
}
func fixtureGuard(t *testing.T, upstream string, sample func() activity) *guard {
	t.Helper()
	return &guard{sample: sample, idle: 10 * time.Minute, pauseFile: filepath.Join(t.TempDir(), "paused"), upstream: upstream, client: &http.Client{Timeout: 3 * time.Second}, slot: make(chan struct{}, 1)}
}
func TestPausedDoesNotReachModel(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer upstream.Close()
	g := fixtureGuard(t, upstream.URL, func() activity { return activity{Idle: time.Second} })
	w := httptest.NewRecorder()
	g.ServeHTTP(w, httptest.NewRequest("POST", "/api/generate", strings.NewReader(`{"model":"fixture","prompt":"synthetic"}`)))
	if w.Code != 503 || calls.Load() != 0 {
		t.Fatalf("code=%d calls=%d", w.Code, calls.Load())
	}
	if err := os.WriteFile(g.pauseFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	g.sample = func() activity { return activity{Idle: time.Hour} }
	if !g.refresh().Paused {
		t.Fatal("manual pause ignored")
	}
	if err := os.Remove(g.pauseFile); err != nil {
		t.Fatal(err)
	}
	if g.refresh().Paused {
		t.Fatal("idle auto mode did not resume")
	}
}
func TestForwardingForcesUnloadAndNonStreaming(t *testing.T) {
	var generation, unload atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		json.NewDecoder(r.Body).Decode(&p)
		if _, ok := p["prompt"]; !ok {
			unload.Add(1)
			io.WriteString(w, `{}`)
			return
		}
		generation.Add(1)
		if p["keep_alive"] != float64(0) || p["stream"] != false {
			t.Error("unsafe background residency/stream setting")
		}
		io.WriteString(w, `{"response":"fixture","done":true}`)
	}))
	defer upstream.Close()
	g := fixtureGuard(t, upstream.URL, func() activity { return activity{Idle: time.Hour} })
	w := httptest.NewRecorder()
	g.ServeHTTP(w, httptest.NewRequest("POST", "/api/generate", strings.NewReader(`{"model":"fixture","prompt":"synthetic","keep_alive":-1,"stream":true}`)))
	if w.Code != 200 || generation.Load() != 1 || unload.Load() != 1 {
		t.Fatalf("code=%d generation=%d unload=%d", w.Code, generation.Load(), unload.Load())
	}
}
func TestActivityCancelsRunningRequestAndUnloads(t *testing.T) {
	var busy atomic.Bool
	started, ended := make(chan struct{}), make(chan struct{})
	var unload atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p map[string]any
		json.NewDecoder(r.Body).Decode(&p)
		if _, ok := p["prompt"]; !ok {
			unload.Add(1)
			io.WriteString(w, `{}`)
			return
		}
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
			t.Error("upstream was not cancelled")
		}
		close(ended)
	}))
	defer upstream.Close()
	g := fixtureGuard(t, upstream.URL, func() activity {
		if busy.Load() {
			return activity{Idle: 0}
		}
		return activity{Idle: time.Hour}
	})
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		g.ServeHTTP(w, httptest.NewRequest("POST", "/api/generate", strings.NewReader(`{"model":"fixture","prompt":"synthetic"}`)))
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	busy.Store(true)
	g.refresh()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("activity did not interrupt request")
	}
	<-ended
	if w.Code != 503 || unload.Load() != 1 {
		t.Fatalf("code=%d unload=%d", w.Code, unload.Load())
	}
}
func TestNoModelManagementOrForcedResumeEndpoint(t *testing.T) {
	g := fixtureGuard(t, "http://127.0.0.1:1", func() activity { return activity{Idle: time.Hour} })
	for _, path := range []string{"/api/pull", "/api/delete", "/guard/resume"} {
		w := httptest.NewRecorder()
		g.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		if w.Code != 404 {
			t.Fatal(path, w.Code)
		}
	}
}
