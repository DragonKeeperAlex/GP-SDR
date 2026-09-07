package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalAIConfig controls the optional evidence-synthesis layer. Raw IQ never
// leaves this computer: waveform classification and protocol decoders run
// first, and the model receives only their bounded text metadata.
type LocalAIConfig struct {
	Enabled           bool   `json:"enabled"`
	Endpoint          string `json:"endpoint"`
	Model             string `json:"model"`
	Profile           string `json:"profile"`
	MinimumConfidence int    `json:"minimumConfidence"`
}

type LocalAIStatus struct {
	State     string                `json:"state"`
	Note      string                `json:"note"`
	Config    LocalAIConfig         `json:"config"`
	CanManage bool                  `json:"canManage"`
	Models    []LocalAIModelOption  `json:"models,omitempty"`
	Learning  LearningLibraryStatus `json:"learning"`
}

type LocalAIModelOption struct {
	Name          string `json:"name"`
	SizeBytes     int64  `json:"sizeBytes,omitempty"`
	ParameterSize string `json:"parameterSize,omitempty"`
	Quantization  string `json:"quantization,omitempty"`
}

type LocalAIAnalyzer struct {
	mu        sync.RWMutex
	path      string
	config    LocalAIConfig
	client    *http.Client
	semaphore chan struct{}
	learning  *SignalLearningLibrary
}

func NewLocalAIAnalyzer(dataDirectory string, learning ...*SignalLearningLibrary) *LocalAIAnalyzer {
	a := &LocalAIAnalyzer{path: filepath.Join(dataDirectory, "Data", "local-ai.json"), client: &http.Client{Timeout: 90 * time.Second}, semaphore: make(chan struct{}, 1)}
	if len(learning) > 0 {
		a.learning = learning[0]
	}
	a.config = defaultLocalAIConfig()
	if data, err := os.ReadFile(a.path); err == nil {
		_ = json.Unmarshal(data, &a.config)
	}
	a.config = normalizeLocalAIConfig(a.config)
	return a
}

func defaultLocalAIConfig() LocalAIConfig {
	return LocalAIConfig{Endpoint: "http://127.0.0.1:11434", Model: "qwen2.5:1.5b", Profile: "lightweight", MinimumConfidence: 55}
}

func normalizeLocalAIConfig(config LocalAIConfig) LocalAIConfig {
	defaults := defaultLocalAIConfig()
	config.Endpoint = strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")
	if config.Endpoint == "" {
		config.Endpoint = defaults.Endpoint
	}
	config.Model = strings.TrimSpace(config.Model)
	if config.Model == "" {
		config.Model = defaults.Model
	}
	switch config.Profile {
	case "lightweight", "balanced", "deep":
	default:
		config.Profile = defaults.Profile
	}
	if config.MinimumConfidence < 10 || config.MinimumConfidence > 100 {
		config.MinimumConfidence = defaults.MinimumConfidence
	}
	return config
}

func validateLocalAIEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("enter an Ollama HTTP address without a path, query, or credentials")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || (!ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()) {
		return errors.New("the Ollama server must use localhost or a private-network IP address")
	}
	return nil
}

func (a *LocalAIAnalyzer) Status() LocalAIStatus {
	a.mu.RLock()
	config := a.config
	a.mu.RUnlock()
	status := LocalAIStatus{Config: config, CanManage: true}
	if a.learning != nil {
		status.Learning = a.learning.Status()
	}
	if !config.Enabled {
		status.State, status.Note = "off", "Local model analysis is off."
		return status
	}
	if err := validateLocalAIEndpoint(config.Endpoint); err != nil {
		status.State, status.Note = "error", err.Error()
		return status
	}
	probeContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(probeContext, http.MethodGet, config.Endpoint+"/api/tags", nil)
	response, err := a.client.Do(request)
	if err != nil {
		status.State, status.Note = "setup", "Start the local model service or use Install model runtime."
		return status
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		status.State, status.Note = "error", "The local model service did not answer correctly."
		return status
	}
	var tags struct {
		Models []struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			Details struct {
				ParameterSize     string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if json.NewDecoder(response.Body).Decode(&tags) == nil {
		configuredModelFound := false
		for _, model := range tags.Models {
			if strings.Contains(strings.ToLower(model.Name), "embed") {
				continue
			}
			status.Models = append(status.Models, LocalAIModelOption{Name: model.Name, SizeBytes: model.Size, ParameterSize: model.Details.ParameterSize, Quantization: model.Details.QuantizationLevel})
			configuredModelFound = configuredModelFound || model.Name == config.Model
		}
		if !configuredModelFound {
			status.State, status.Note = "setup", "The selected model is not installed on this Ollama server. Choose one of the available models."
			return status
		}
	}
	status.State, status.Note = "ready", "Evidence synthesis is ready. Only bounded text metadata is sent to the configured Ollama server; IQ and audio stay in GP-SDR."
	return status
}

func (a *LocalAIAnalyzer) Update(config LocalAIConfig) (LocalAIStatus, error) {
	config = normalizeLocalAIConfig(config)
	if err := validateLocalAIEndpoint(config.Endpoint); err != nil {
		return a.Status(), err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return a.Status(), err
	}
	if err = os.MkdirAll(filepath.Dir(a.path), 0o755); err == nil {
		err = os.WriteFile(a.path, append(data, '\n'), 0o600)
	}
	if err != nil {
		return a.Status(), err
	}
	a.mu.Lock()
	a.config = config
	a.mu.Unlock()
	return a.Status(), nil
}

type localAIResponse struct {
	SignalFamily string   `json:"signalFamily"`
	Modulation   string   `json:"modulation"`
	Summary      string   `json:"summary"`
	Confidence   float64  `json:"confidence"`
	Evidence     []string `json:"evidence"`
	Callsigns    []string `json:"callsigns"`
}

func (a *LocalAIAnalyzer) Analyze(parent context.Context, event TransmissionEvent) (SignalIntelligence, error) {
	a.mu.RLock()
	config := a.config
	a.mu.RUnlock()
	if !config.Enabled {
		return SignalIntelligence{}, errors.New("local model analysis is off")
	}
	select {
	case a.semaphore <- struct{}{}:
		defer func() { <-a.semaphore }()
	case <-parent.Done():
		return SignalIntelligence{}, parent.Err()
	}
	metadata := map[string]any{"frequencyHz": event.FrequencyHz, "bandwidthHz": event.BandwidthHz, "observedModulation": event.Modulation,
		"signalDBFS": event.SignalDBFS, "noiseDBFS": event.NoiseDBFS, "transcript": stringValue(event.Transcript), "callsigns": event.Callsigns,
		"decoderMessages": event.DecoderMessages, "waveformAnalysis": event.Analysis, "frequencyBand": radioFrequencyBand(event.FrequencyHz), "location": event.Location}
	encoded, _ := json.Marshal(metadata)
	examples := []ConfirmedSignalSample{}
	if a.learning != nil {
		examples = a.learning.Similar(event, 5)
	}
	for index := range examples {
		examples[index].IQPath, examples[index].AudioPath = "", ""
	}
	exampleData, _ := json.Marshal(examples)
	prompt := "You are GP-SDR's conservative radio evidence analyst. Correlate only the supplied local DSP, decoder, transcript, frequency and location evidence. Never claim a protocol without decoded frames; label guesses as candidates. Confirmed examples are user-labeled ground truth, but may differ from the current signal. Return JSON with signalFamily, modulation, summary, confidence from 0 to 1, evidence, and callsigns. Use UNKNOWN when evidence is insufficient. Confirmed examples:\n" + string(exampleData) + "\nCurrent evidence:\n" + string(encoded)
	format := map[string]any{"type": "object", "required": []string{"signalFamily", "modulation", "summary", "confidence", "evidence", "callsigns"}, "properties": map[string]any{
		"signalFamily": map[string]any{"type": "string"}, "modulation": map[string]any{"type": "string"}, "summary": map[string]any{"type": "string"},
		"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1}, "evidence": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"callsigns": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}}
	// Radio analysis needs the schema-constrained answer, not a model's hidden
	// reasoning stream. Qwen 3.5 can otherwise return only `thinking` and leave
	// Ollama's `response` empty.
	requestBody, _ := json.Marshal(map[string]any{"model": config.Model, "prompt": prompt, "stream": false, "think": false, "format": format, "options": map[string]any{"temperature": 0.1, "num_ctx": localAIContext(config.Profile)}})
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint+"/api/generate", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return SignalIntelligence{}, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return SignalIntelligence{}, fmt.Errorf("local model returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Response string `json:"response"`
	}
	if err = json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return SignalIntelligence{}, err
	}
	var output localAIResponse
	if err = json.Unmarshal([]byte(envelope.Response), &output); err != nil {
		return SignalIntelligence{}, errors.New("local model returned invalid identification JSON")
	}
	output.Confidence = clamp(output.Confidence, 0, 1)
	expectedBand := radioFrequencyBand(event.FrequencyHz)
	reportedBand := reportedRadioBand(output.SignalFamily)
	if expectedBand != "" && reportedBand != "" && reportedBand != expectedBand {
		output.Evidence = append(output.Evidence, "Model frequency band corrected from "+reportedBand+" to "+expectedBand)
		output.SignalFamily = strings.Replace(output.SignalFamily, reportedBand, expectedBand, 1)
		output.SignalFamily = strings.Replace(output.SignalFamily, strings.ToLower(reportedBand), expectedBand, 1)
		output.Confidence = minFloat(output.Confidence, .69)
	}
	measuredModulation := strings.ToUpper(strings.TrimSpace(event.Modulation))
	if event.Analysis != nil && event.Analysis.Confidence >= .55 && event.Analysis.Modulation != "" && event.Analysis.Modulation != "UNKNOWN" {
		measuredModulation = strings.ToUpper(event.Analysis.Modulation)
	}
	if measuredModulation != "" && measuredModulation != "AUTO" && measuredModulation != "UNKNOWN" && strings.ToUpper(output.Modulation) != measuredModulation {
		output.Evidence = append(output.Evidence, "Model modulation corrected to match measured DSP evidence")
		output.Modulation = measuredModulation
		output.Confidence = minFloat(output.Confidence, .69)
	}
	if output.Confidence*100 < float64(config.MinimumConfidence) {
		output.SignalFamily, output.Modulation = "Unknown", "UNKNOWN"
	}
	family := firstNonEmpty(strings.TrimSpace(output.SignalFamily), "Unknown")
	modulation := firstNonEmpty(strings.ToUpper(strings.TrimSpace(output.Modulation)), "UNKNOWN")
	// Free-form model prose routinely contradicted the structured result (for
	// example a P25 family paired with an "analog" summary). Store a concise
	// structured candidate instead; decoded frames remain the verification gate.
	summary := fmt.Sprintf("Local evidence candidate: %s · %s · %.0f%%", family, modulation, output.Confidence*100)
	return SignalIntelligence{Engine: "GP-SDR local model · " + config.Model, SignalFamily: family,
		Modulation: modulation, Confidence: output.Confidence,
		Summary: summary, Evidence: output.Evidence, Callsigns: mergeUniqueStrings(
			ExtractCallsigns(strings.Join(output.Callsigns, " ")),
			ExtractCallsigns(stringValue(event.Transcript)))}, nil
}

func radioFrequencyBand(frequencyHz float64) string {
	switch {
	case frequencyHz >= 3e6 && frequencyHz < 30e6:
		return "HF"
	case frequencyHz >= 30e6 && frequencyHz < 300e6:
		return "VHF"
	case frequencyHz >= 300e6 && frequencyHz < 1e9:
		return "UHF"
	case frequencyHz >= 1e9 && frequencyHz < 30e9:
		return "SHF"
	default:
		return ""
	}
}

func reportedRadioBand(family string) string {
	upper := strings.ToUpper(family)
	for _, band := range []string{"SHF", "UHF", "VHF", "HF"} {
		if strings.Contains(upper, band) {
			return band
		}
	}
	return ""
}

func localAIContext(profile string) int {
	if profile == "deep" {
		return 8192
	}
	if profile == "balanced" {
		return 4096
	}
	return 2048
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
