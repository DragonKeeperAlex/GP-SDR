package app

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RemoteReceiverStore struct {
	mu    sync.RWMutex
	path  string
	items []RemoteReceiver
}

func NewRemoteReceiverStore(path string) (*RemoteReceiverStore, error) {
	s := &RemoteReceiverStore{path: path, items: []RemoteReceiver{}}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &s.items); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *RemoteReceiverStore) List() []RemoteReceiver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]RemoteReceiver(nil), s.items...)
}

func (s *RemoteReceiverStore) Save(item RemoteReceiver) (RemoteReceiver, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.Host = strings.TrimSpace(item.Host)
	if item.Name == "" {
		item.Name = "Remote RTL-SDR"
	}
	if item.ID == "" {
		item.ID = "rtl-tcp-" + NewID()
	}
	if item.Host == "" || item.Port < 1 || item.Port > 65535 {
		return item, errors.New("enter a valid rtl_tcp host and port")
	}
	if net.ParseIP(item.Host) == nil && strings.ContainsAny(item.Host, " /\\") {
		return item, errors.New("enter a valid rtl_tcp host name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.items {
		if s.items[i].ID == item.ID {
			s.items[i] = item
			found = true
			break
		}
	}
	if !found {
		s.items = append(s.items, item)
	}
	return item, s.persistLocked()
}

func (s *RemoteReceiverStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.items[:0]
	for _, item := range s.items {
		if item.ID != id {
			next = append(next, item)
		}
	}
	s.items = next
	return s.persistLocked()
}

func (s *RemoteReceiverStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func remoteDevices(items []RemoteReceiver) []SDRDevice {
	result := make([]SDRDevice, 0, len(items))
	limit := 3.2e6
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		connected := false
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(item.Host, itoa(item.Port)), 350*time.Millisecond)
		if err == nil {
			connected = true
			_ = conn.Close()
		}
		note := "Remote rtl_tcp server is unreachable."
		if connected {
			note = "Remote rtl_tcp receiver is online; tuner, audio, spectrum, scanning, and Mapper are available."
		}
		device := SDRDevice{ID: item.ID, Name: item.Name, Kind: "RTL-TCP", Driver: "rtl_tcp", Host: item.Host, Port: item.Port, Available: true, Connected: connected, SampleRateLimit: &limit, Note: &note}
		applyNominalFrequencyRange(&device)
		result = append(result, device)
	}
	return result
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for value > 0 {
		i--
		b[i] = digits[value%10]
		value /= 10
	}
	return string(b[i:])
}
