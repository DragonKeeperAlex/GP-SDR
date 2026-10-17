package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type relayOutputSettings struct {
	Volume      float64 `json:"volume"`
	SystemAudio bool    `json:"systemAudio"`
}

func validRelayOutputVolume(volume float64) bool {
	return !math.IsNaN(volume) && !math.IsInf(volume, 0) && volume >= 0 && volume <= 1
}

func (r *RelayRuntime) loadOutputVolume(directory string) {
	r.settingsPath = filepath.Join(directory, "Data", "relay-output.json")
	data, err := os.ReadFile(r.settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	var settings relayOutputSettings
	if err == nil {
		err = json.Unmarshal(data, &settings)
	}
	if err == nil && !validRelayOutputVolume(settings.Volume) {
		err = errors.New("invalid saved relay output volume")
	}
	if err != nil {
		r.outputVolume = 0
		r.lastError = fmt.Sprintf("Relay output muted: could not read saved volume: %v", err)
		return
	}
	r.outputVolume = settings.Volume
	r.systemAudio = settings.SystemAudio
}

// This gain belongs only to the relay PCM path. No ALSA/system mixer is changed.
func (r *RelayRuntime) SetOutputVolume(volume float64) (RelayStatus, error) {
	if !validRelayOutputVolume(volume) {
		return r.Status(), errors.New("relay output volume must be between 0 and 1")
	}
	r.mu.Lock()
	if r.settingsPath != "" {
		data, err := json.Marshal(relayOutputSettings{Volume: volume, SystemAudio: r.systemAudio})
		if err == nil {
			err = writeBytesAtomic(r.settingsPath, data)
		}
		if err != nil {
			r.mu.Unlock()
			return r.Status(), fmt.Errorf("save relay output volume: %w", err)
		}
	}
	r.outputVolume = volume
	r.mu.Unlock()
	return r.Status(), nil
}

func (r *RelayRuntime) SetSystemAudio(enabled bool) (RelayStatus, error) {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return r.Status(), errors.New("stop the relay before changing audio mode")
	}
	if r.settingsPath != "" {
		data, err := json.Marshal(relayOutputSettings{Volume: r.outputVolume, SystemAudio: enabled})
		if err == nil {
			err = writeBytesAtomic(r.settingsPath, data)
		}
		if err != nil {
			r.mu.Unlock()
			return r.Status(), err
		}
	}
	r.systemAudio = enabled
	r.mu.Unlock()
	return r.Status(), nil
}
