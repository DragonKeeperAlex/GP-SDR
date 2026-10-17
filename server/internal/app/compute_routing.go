package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HomeComputeStatus struct {
	Available bool   `json:"available"`
	State     string `json:"state"`
	Note      string `json:"note"`
}

// Cheap, bounded probes; never start or load a model to decide where to run.
func (a *LocalAIAnalyzer) homeCompute(ctx context.Context, config LocalAIConfig) HomeComputeStatus {
	if !config.RemoteEnabled {
		return HomeComputeStatus{State: "off", Note: "Home compute is not selected."}
	}
	a.remoteMu.Lock()
	defer a.remoteMu.Unlock()
	key := config.RemoteEndpoint + "|" + config.RemoteModel
	if key == a.remoteKey && time.Since(a.remoteChecked) < 5*time.Second {
		return a.remoteStatus
	}
	status := HomeComputeStatus{State: "offline", Note: "Home PC unavailable; saved home jobs will retry."}
	defer func() { a.remoteKey = key; a.remoteStatus = status; a.remoteChecked = time.Now() }()
	if validateLocalAIEndpoint(config.RemoteEndpoint) != nil {
		return status
	}
	probe, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	read := func(path string, out any) (int, error) {
		req, _ := http.NewRequestWithContext(probe, "GET", config.RemoteEndpoint+path, nil)
		r, err := a.client.Do(req)
		if err != nil {
			return 0, err
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return r.StatusCode, errors.New("probe unavailable")
		}
		return r.StatusCode, json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(out)
	}
	var guard struct {
		Paused *bool  `json:"paused"`
		Reason string `json:"reason"`
	}
	_, err := read("/guard/status", &guard)
	if err == nil && guard.Paused != nil && *guard.Paused {
		status.State = "paused"
		status.Note = "Home PC paused: " + truncateText(guard.Reason, 160)
		return status
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if _, err = read("/api/tags", &tags); err != nil {
		return status
	}
	for _, model := range tags.Models {
		if model.Name == config.RemoteModel {
			status = HomeComputeStatus{Available: true, State: "ready", Note: "Home PC ready for model analysis."}
			return status
		}
	}
	status.State = "setup"
	status.Note = "Selected home model is not installed."
	return status
}

func (a *LocalAIAnalyzer) computeStatus(status *LocalAIStatus) {
	config := normalizeLocalAIConfig(status.Config)
	status.Config = config
	switch config.ComputeMode {
	case "off":
		status.ComputeTarget = "off"
	case "local":
		status.ComputeTarget = "local"
	default:
		status.Home = a.homeCompute(context.Background(), config)
		if status.Home.Available {
			status.ComputeTarget = "home"
			status.State = "ready"
			status.Note = status.Home.Note + " IQ/audio and RF decoding stay on this receiver."
		} else if config.ComputeMode == "auto" {
			status.ComputeTarget = "local"
			status.Note = status.Home.Note + " Using the local model when ready. " + status.Note
		} else {
			status.ComputeTarget = "waiting"
			status.State = "waiting"
			status.Note = status.Home.Note + " Home-only mode keeps evidence queued and does not run the Pi model."
		}
	}
}

func (a *LocalAIAnalyzer) SetComputeMode(mode string) (LocalAIStatus, error) {
	switch mode {
	case "auto", "local", "deferred", "off":
	default:
		return a.Status(), errors.New("choose Automatic, Local, Home only, or Off")
	}
	a.mu.RLock()
	config := a.config
	a.mu.RUnlock()
	if (mode == "auto" || mode == "deferred") && (config.RemoteEndpoint == "" || config.RemoteModel == "") {
		return a.Status(), errors.New("configure a home model server before selecting home compute")
	}
	config.ComputeMode = mode
	return a.Update(config)
}

func (a *LocalAIAnalyzer) analyzePreferred(ctx context.Context, event TransmissionEvent, config LocalAIConfig) (SignalIntelligence, error) {
	if !config.Enabled {
		return SignalIntelligence{}, errors.New("model analysis is off")
	}
	var homeErr error
	if config.RemoteEnabled {
		availability := a.homeCompute(ctx, config)
		if availability.Available {
			remote := &LocalAIAnalyzer{client: a.client, semaphore: a.remoteSemaphore, learning: a.learning, profiles: a.profiles}
			if remote.semaphore == nil {
				remote.semaphore = make(chan struct{}, 1)
			}
			// A separate slot preserves the Pi's local-model capacity during home work.
			result, err := remote.analyzeWithConfig(ctx, event, LocalAIConfig{Enabled: true, Endpoint: config.RemoteEndpoint, Model: config.RemoteModel, Profile: "deep", MinimumConfidence: config.RemoteMinConfidence, ContextLength: config.ContextLength})
			if err == nil {
				result.Engine = strings.Replace(result.Engine, "GP-SDR local model", "GP-SDR home model", 1)
				return result, nil
			}
			homeErr = err
		} else {
			homeErr = errors.New(availability.Note)
		}
		if ctx.Err() != nil {
			return SignalIntelligence{}, ctx.Err()
		}
		if config.ComputeMode == "deferred" {
			if a.deferEvidence != nil {
				if err := a.deferEvidence(event); err != nil {
					return SignalIntelligence{}, fmt.Errorf("cannot save home analysis: %w", err)
				}
			}
			return SignalIntelligence{}, fmt.Errorf("waiting for home compute: %w", homeErr)
		}
	}
	return a.analyzeWithConfig(ctx, event, config)
}
