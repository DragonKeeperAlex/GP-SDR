// home-compute-guard gives the interactive Windows user priority over GP-SDR AI jobs.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type activity struct {
	Idle                   time.Duration
	Fullscreen, Controller bool
	Err                    error
}
type guardStatus struct {
	Paused             bool      `json:"paused"`
	Reason             string    `json:"reason"`
	IdleSeconds        int64     `json:"idleSeconds"`
	ResumeAfterSeconds int64     `json:"resumeAfterSeconds"`
	Active             bool      `json:"active"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
type guard struct {
	mu                              sync.Mutex
	sample                          func() activity
	idle                            time.Duration
	pauseFile, statusFile, upstream string
	client                          *http.Client
	status                          guardStatus
	cancel                          context.CancelFunc
	slot                            chan struct{}
}

func pauseReason(a activity, manual bool, idle time.Duration) string {
	switch {
	case manual:
		return "Manually paused"
	case a.Err != nil:
		return "Waiting for interactive desktop activity information"
	case a.Controller:
		return "Game controller in use"
	case a.Fullscreen:
		return "Fullscreen application in use"
	case a.Idle < idle:
		return "PC in use; waiting for idle time"
	default:
		return ""
	}
}
func (g *guard) refresh() guardStatus {
	// sample owns controller history; serialize all native reads and state changes.
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.sample()
	_, err := os.Stat(g.pauseFile)
	manual := err == nil || !os.IsNotExist(err)
	reason := pauseReason(a, manual, g.idle)
	g.status = guardStatus{Paused: reason != "", Reason: reason, IdleSeconds: int64(a.Idle / time.Second), ResumeAfterSeconds: int64(g.idle / time.Second), Active: g.cancel != nil, UpdatedAt: time.Now().UTC()}
	if g.status.Paused && g.cancel != nil {
		g.cancel()
	}
	return g.status
}
func (g *guard) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		s := g.refresh()
		// A small local status snapshot contains no window titles, keys or prompts.
		if g.statusFile != "" {
			b, _ := json.MarshalIndent(s, "", "  ")
			tmp := g.statusFile + ".tmp"
			if os.WriteFile(tmp, b, 0600) == nil {
				_ = os.Rename(tmp, g.statusFile)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func reply(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func (g *guard) paused(w http.ResponseWriter, reason string) {
	w.Header().Set("Retry-After", "30")
	reply(w, http.StatusServiceUnavailable, map[string]string{"error": "GP-SDR home compute paused: " + reason + ". Saved jobs will retry automatically."})
}
func (g *guard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.URL.Path == "/guard/status" {
		reply(w, 200, g.refresh())
		return
	}
	readOnly := r.Method == "GET" && (r.URL.Path == "/api/tags" || r.URL.Path == "/api/ps" || r.URL.Path == "/api/version")
	generate := r.Method == "POST" && (r.URL.Path == "/api/generate" || r.URL.Path == "/api/chat")
	if !readOnly && !generate {
		reply(w, 404, map[string]string{"error": "Unsupported background-compute endpoint"})
		return
	}
	var body []byte
	if generate {
		s := g.refresh()
		if s.Paused {
			g.paused(w, s.Reason)
			return
		}
		select {
		case g.slot <- struct{}{}:
			defer func() { <-g.slot }()
		default:
			g.paused(w, "Another background request is running")
			return
		}
		var p map[string]json.RawMessage
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&p) != nil || p == nil {
			reply(w, 400, map[string]string{"error": "Invalid or oversized JSON request"})
			return
		}
		var model string
		if json.Unmarshal(p["model"], &model) != nil || model == "" {
			reply(w, 400, map[string]string{"error": "Model is required"})
			return
		}
		// GP-SDR consumes bounded responses. Never leave background model VRAM resident.
		p["stream"] = json.RawMessage("false")
		p["keep_alive"] = json.RawMessage("0")
		body, _ = json.Marshal(p)
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		g.mu.Lock()
		g.cancel = cancel
		g.mu.Unlock()
		defer func() { cancel(); g.unload(model); g.mu.Lock(); g.cancel = nil; g.mu.Unlock() }()
		if s = g.refresh(); s.Paused {
			g.paused(w, s.Reason)
			return
		}
		r = r.WithContext(ctx)
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, g.upstream+r.URL.Path, bytes.NewReader(body))
	if err != nil {
		reply(w, 500, map[string]string{"error": "Cannot create background request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := g.client.Do(req)
	if err != nil {
		if s := g.refresh(); s.Paused {
			g.paused(w, s.Reason)
		} else {
			reply(w, 502, map[string]string{"error": "Home model unavailable; saved jobs will retry"})
		}
		return
	}
	defer response.Body.Close()
	// Buffer before delivering: a cancelled generation must never be accepted as complete.
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil || r.Context().Err() != nil {
		g.paused(w, "Request interrupted; it will be retried")
		return
	}
	if generate {
		if s := g.refresh(); s.Paused {
			g.paused(w, s.Reason)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(data)
}
func (g *guard) unload(model string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	b, _ := json.Marshal(map[string]any{"model": model, "keep_alive": 0})
	req, _ := http.NewRequestWithContext(ctx, "POST", g.upstream+"/api/generate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if r, e := g.client.Do(req); e == nil {
		io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
		r.Body.Close()
	}
}
func main() {
	listen := flag.String("listen", "0.0.0.0:11435", "Private-LAN guard endpoint (restrict Windows firewall to the Pi)")
	idle := flag.Duration("idle", 10*time.Minute, "Idle delay before automatic resumption")
	directory := flag.String("state-dir", ".", "Directory for manual pause marker and status")
	flag.Parse()
	if *idle < time.Minute {
		log.Fatal("idle delay must be at least one minute")
	}
	if err := os.MkdirAll(*directory, 0700); err != nil {
		log.Fatal(err)
	}
	g := &guard{sample: nativeActivity, idle: *idle, pauseFile: filepath.Join(*directory, "paused"), statusFile: filepath.Join(*directory, "status.json"), upstream: "http://127.0.0.1:11434", client: &http.Client{Timeout: 3 * time.Minute}, slot: make(chan struct{}, 1)}
	g.refresh()
	go g.monitor(context.Background())
	server := &http.Server{Addr: *listen, Handler: g, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 4 * time.Minute, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	fmt.Println("GP-SDR home compute guard: user activity takes priority; automatic idle delay", *idle)
	log.Fatal(server.ListenAndServe())
}
