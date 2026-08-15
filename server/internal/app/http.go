package app

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Server struct {
	runtime *Runtime
	web     fs.FS
	token   string
	http    *http.Server
}

func NewServer(runtime *Runtime, web fs.FS, host string, port uint16, token string) *Server {
	s := &Server{runtime: runtime, web: web, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.handleAPI)
	mux.Handle("/", http.FileServer(http.FS(web)))
	s.http = &http.Server{Addr: fmt.Sprintf("%s:%d", host, port), Handler: s.securityHeaders(mux)}
	return s
}
func (s *Server) Handler() http.Handler { return s.http.Handler }
func (s *Server) ListenAndServe() error { return s.http.ListenAndServe() }
func (s *Server) Close() error          { return s.http.Close() }
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func (s *Server) authorized(r *http.Request) bool {
	if s.token == "" {
		return true
	}
	provided := r.Header.Get("X-GP-SDR-Token")
	if provided == "" {
		provided = r.URL.Query().Get("token")
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) == 1
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "A server access token is required.")
		return
	}
	defer r.Body.Close()
	path := r.URL.Path
	switch {
	case r.Method == "GET" && path == "/api/status":
		writeJSON(w, 200, s.runtime.Status())
	case r.Method == "GET" && path == "/api/devices":
		writeJSON(w, 200, s.runtime.Devices())
	case r.Method == "POST" && path == "/api/devices/refresh":
		s.runtime.Refresh()
		writeJSON(w, 200, s.runtime.Devices())
	case r.Method == "GET" && path == "/api/decoders":
		writeJSON(w, 200, s.runtime.Decoders())
	case r.Method == "GET" && path == "/api/integrations":
		writeJSON(w, 200, s.runtime.Integrations())
	case r.Method == "GET" && path == "/api/setup":
		writeJSON(w, 200, s.runtime.Setup())
	case r.Method == "POST" && path == "/api/setup/install":
		var body struct {
			ComponentID string `json:"componentID"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		job, err := s.runtime.Install(body.ComponentID)
		writeResult(w, job, err, http.StatusAccepted)
	case r.Method == "GET" && path == "/api/p25/status":
		writeJSON(w, 200, s.runtime.P25Status())
	case r.Method == "GET" && path == "/api/spectrum":
		writeJSON(w, 200, s.runtime.Spectrum())
	case r.Method == "POST" && path == "/api/tuner/start":
		var request TunerRequest
		if !decodeBody(w, r, &request) {
			return
		}
		if err := s.runtime.Tune(request); err != nil {
			writeError(w, 400, err.Error())
		} else {
			writeJSON(w, 200, s.runtime.Status())
		}
	case r.Method == "POST" && path == "/api/tuner/stop":
		s.runtime.Stop()
		writeJSON(w, 200, s.runtime.Status())
	case r.Method == "GET" && path == "/api/radioreference/nearby":
		zipCode, zipErr := strconv.Atoi(r.URL.Query().Get("zip"))
		radius, radiusErr := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		if zipErr != nil || radiusErr != nil {
			writeError(w, 400, "ZIP code and range are required.")
			return
		}
		result, err := s.runtime.RadioReferenceNearby(zipCode, radius)
		writeResult(w, result, err, 200)
	case r.Method == "GET" && path == "/api/range-sync":
		writeJSON(w, 200, s.runtime.RangeSyncStatus())
	case r.Method == "PUT" && path == "/api/range-sync":
		var config RangeSyncConfig
		if !decodeBody(w, r, &config) {
			return
		}
		status, err := s.runtime.UpdateRangeSync(config)
		writeResult(w, status, err, 200)
	case r.Method == "POST" && path == "/api/range-sync/now":
		writeJSON(w, 200, s.runtime.SyncRangesNow())
	case r.Method == "GET" && path == "/api/profiles":
		writeJSON(w, 200, s.runtime.Profiles.All())
	case r.Method == "POST" && path == "/api/profiles":
		var profile ScanProfile
		if !decodeBody(w, r, &profile) {
			return
		}
		saved, err := s.runtime.Profiles.Save(profile)
		writeResult(w, saved, err, 201)
	case r.Method == "POST" && path == "/api/profiles/import":
		data, err := io.ReadAll(io.LimitReader(r.Body, 1_000_001))
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		saved, err := s.runtime.Profiles.Import(data)
		writeResult(w, saved, err, 201)
	case r.Method == "POST" && path == "/api/profiles/import-channels":
		data, err := io.ReadAll(io.LimitReader(r.Body, 5_000_001))
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		saved, err := s.runtime.Profiles.ImportChannelCSV(r.URL.Query().Get("filename"), data)
		writeResult(w, saved, err, 201)
	case r.Method == "POST" && path == "/api/profiles/duplicate":
		saved, err := s.runtime.Profiles.Duplicate(r.URL.Query().Get("id"))
		writeResult(w, saved, err, 201)
	case r.Method == "DELETE" && path == "/api/profiles":
		err := s.runtime.Profiles.Delete(r.URL.Query().Get("id"))
		if err != nil {
			writeError(w, 400, err.Error())
		} else {
			writeJSON(w, 200, map[string]any{"ok": true, "message": "Profile deleted."})
		}
	case r.Method == "GET" && path == "/api/profiles/export":
		data, err := s.runtime.Profiles.Export(r.URL.Query().Get("id"))
		if err != nil {
			writeError(w, 404, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/vnd.gp-sdr.profile+json")
		w.Header().Set("Content-Disposition", "attachment; filename=GP-SDR-Profile.json")
		_, _ = w.Write(data)
	case r.Method == "GET" && path == "/api/events":
		writeJSON(w, 200, s.runtime.Events.Recent(intQuery(r, "limit", 200)))
	case r.Method == "GET" && path == "/api/audio":
		s.serveEventAudio(w, r)
	case r.Method == "GET" && path == "/api/live-audio":
		s.serveLiveAudio(w, r)
	case r.Method == "GET" && path == "/api/signals":
		writeJSON(w, 200, s.runtime.Events.Signals(intQuery(r, "limit", 500)))
	case r.Method == "POST" && path == "/api/control/start":
		var body struct {
			ProfileID string `json:"profileID"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if err := s.runtime.Start(body.ProfileID); err != nil {
			writeError(w, 400, err.Error())
		} else {
			writeJSON(w, 200, s.runtime.Status())
		}
	case r.Method == "POST" && path == "/api/control/stop":
		s.runtime.Stop()
		writeJSON(w, 200, s.runtime.Status())
	case r.Method == "GET" && path == "/api/mixer":
		writeJSON(w, 200, s.runtime.Mixer())
	case r.Method == "POST" && path == "/api/mixer":
		var body struct {
			ID          string `json:"id"`
			Muted, Solo *bool
			Volume, Pan *float64
		}
		if !decodeBody(w, r, &body) {
			return
		}
		item, err := s.runtime.UpdateMixer(body.ID, body.Muted, body.Solo, body.Volume, body.Pan)
		writeResult(w, item, err, 200)
	case r.Method == "GET" && path == "/api/receiver-plan":
		writeJSON(w, 200, s.runtime.Plan())
	default:
		writeError(w, 404, "Not found.")
	}
}

func (s *Server) serveLiveAudio(w http.ResponseWriter, r *http.Request) {
	if s.runtime.audioHub == nil {
		writeError(w, 503, "Live audio is unavailable.")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "Streaming is unavailable.")
		return
	}
	frames, unsubscribe := s.runtime.audioHub.Subscribe()
	defer unsubscribe()
	w.Header().Set("Content-Type", "application/vnd.gp-sdr.pcm")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case frame, open := <-frames:
			if !open {
				return
			}
			identifier := []byte(frame.ChannelID)
			if len(identifier) > 65535 || len(frame.Samples) > 10_000_000 {
				continue
			}
			var packet bytes.Buffer
			packet.Grow(10 + len(identifier) + len(frame.Samples)*2)
			_ = binary.Write(&packet, binary.LittleEndian, uint16(len(identifier)))
			_ = binary.Write(&packet, binary.LittleEndian, uint32(frame.SampleRate))
			_ = binary.Write(&packet, binary.LittleEndian, uint32(len(frame.Samples)))
			_, _ = packet.Write(identifier)
			_ = binary.Write(&packet, binary.LittleEndian, frame.Samples)
			if _, err := w.Write(packet.Bytes()); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) serveEventAudio(w http.ResponseWriter, r *http.Request) {
	event, ok := s.runtime.Events.Get(r.URL.Query().Get("id"))
	if !ok || event.AudioPath == nil || filepath.Ext(*event.AudioPath) != ".wav" {
		writeError(w, 404, "Recording not found.")
		return
	}
	recordingRoot := filepath.Join(s.runtime.dataDirectory, "Recordings")
	relative, err := filepath.Rel(recordingRoot, *event.AudioPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		writeError(w, 403, "Recording path is outside GP-SDR storage.")
		return
	}
	file, err := os.Open(*event.AudioPath)
	if err != nil {
		writeError(w, 404, "Recording not found.")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		writeError(w, 404, "Recording not found.")
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	http.ServeContent(w, r, filepath.Base(*event.AudioPath), info.ModTime(), file)
}

func decodeBody(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1_000_000))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, 400, "Invalid request: "+err.Error())
		return false
	}
	return true
}
func writeResult(w http.ResponseWriter, value any, err error, status int) {
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, status, value)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "message": message})
}
func intQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
