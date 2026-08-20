package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MapperConfig struct {
	WebhookURL string `json:"webhookURL"`
	Secret     string `json:"secret,omitempty"`
	AutoUpload bool   `json:"autoUpload"`
}

type MapperStatus struct {
	Config       MapperConfig `json:"config"`
	LastUpload   *time.Time   `json:"lastUpload,omitempty"`
	LastError    string       `json:"lastError,omitempty"`
	UploadedRows int          `json:"uploadedRows"`
}

type MapperManager struct {
	mu           sync.RWMutex
	path         string
	config       MapperConfig
	lastUpload   *time.Time
	lastError    string
	uploadedRows int
	lastSeen     map[string]time.Time
	events       *EventStore
	client       *http.Client
}

func NewMapperManager(dataDirectory string, events *EventStore) *MapperManager {
	m := &MapperManager{path: filepath.Join(dataDirectory, "Data", "mapper.json"), events: events, client: &http.Client{Timeout: 12 * time.Second}, lastSeen: make(map[string]time.Time)}
	if data, err := os.ReadFile(m.path); err == nil {
		_ = json.Unmarshal(data, &m.config)
	}
	go m.loop()
	return m
}

func (m *MapperManager) Status() MapperStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MapperStatus{Config: m.config, LastUpload: m.lastUpload, LastError: m.lastError, UploadedRows: m.uploadedRows}
}

func (m *MapperManager) Update(config MapperConfig) (MapperStatus, error) {
	config.WebhookURL = strings.TrimSpace(config.WebhookURL)
	if config.WebhookURL != "" {
		parsed, err := url.Parse(config.WebhookURL)
		if err != nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost"))) {
			return MapperStatus{}, errors.New("enter a valid HTTPS Google Apps Script webhook URL")
		}
	}
	if config.AutoUpload && config.WebhookURL == "" {
		return MapperStatus{}, errors.New("add a webhook URL before enabling automatic upload")
	}
	m.mu.Lock()
	m.config = config
	data, _ := json.MarshalIndent(config, "", "  ")
	_ = os.MkdirAll(filepath.Dir(m.path), 0o700)
	err := os.WriteFile(m.path, data, 0o600)
	m.mu.Unlock()
	return m.Status(), err
}

func (m *MapperManager) UploadNow() MapperStatus { m.upload(); return m.Status() }
func (m *MapperManager) loop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.RLock()
		enabled := m.config.AutoUpload
		m.mu.RUnlock()
		if enabled {
			m.upload()
		}
	}
}

func (m *MapperManager) upload() {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()
	if config.WebhookURL == "" {
		return
	}
	signals := m.events.Signals(5000)
	rows := make([]SignalSummary, 0)
	m.mu.RLock()
	for _, signal := range signals {
		if !signal.LastSeen.After(m.lastSeen[signal.ID]) {
			continue
		}
		rows = append(rows, signal)
	}
	m.mu.RUnlock()
	if len(rows) == 0 {
		return
	}
	payload, _ := json.Marshal(map[string]any{"source": "GP-SDR Mapper", "secret": config.Secret, "signals": rows})
	request, _ := http.NewRequest(http.MethodPost, config.WebhookURL, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err == nil && response != nil {
		response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			err = errors.New("sheet webhook rejected the upload")
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastError = err.Error()
		return
	}
	now := time.Now()
	m.lastUpload = &now
	m.lastError = ""
	m.uploadedRows += len(rows)
	for _, row := range rows {
		m.lastSeen[row.ID] = row.LastSeen
	}
}
