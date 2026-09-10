package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	ContextLength     int    `json:"contextLength"`
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

type LocalAIBenchmarkModel struct {
	Model             string  `json:"model"`
	Cases             int     `json:"cases"`
	Grounded          int     `json:"grounded"`
	Errors            int     `json:"errors"`
	AverageMillis     float64 `json:"averageMillis"`
	GroundedPercent   float64 `json:"groundedPercent"`
	StructuredPercent float64 `json:"structuredPercent"`
}

type LocalAIBenchmarkStatus struct {
	Running    bool                    `json:"running"`
	StartedAt  *time.Time              `json:"startedAt,omitempty"`
	FinishedAt *time.Time              `json:"finishedAt,omitempty"`
	Current    string                  `json:"current,omitempty"`
	Source     string                  `json:"source"`
	Results    []LocalAIBenchmarkModel `json:"results"`
	LastError  string                  `json:"lastError,omitempty"`
}

type LocalAIAnalyzer struct {
	mu          sync.RWMutex
	path        string
	config      LocalAIConfig
	client      *http.Client
	semaphore   chan struct{}
	learning    *SignalLearningLibrary
	profiles    *ProfileStore
	benchmarkMu sync.RWMutex
	benchmark   LocalAIBenchmarkStatus
}

func (a *LocalAIAnalyzer) SetReferenceProfiles(profiles *ProfileStore) { a.profiles = profiles }

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
	if config.ContextLength != 0 && (config.ContextLength < 2048 || config.ContextLength > 262144) {
		config.ContextLength = 0
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
	return a.analyzeWithConfig(parent, event, config)
}

func (a *LocalAIAnalyzer) analyzeWithConfig(parent context.Context, event TransmissionEvent, config LocalAIConfig) (SignalIntelligence, error) {
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
		"decoderMessages": event.DecoderMessages, "waveformAnalysis": event.Analysis, "frequencyBand": radioFrequencyBand(event.FrequencyHz), "location": event.Location,
		"localReferenceMatches": a.localReferenceMatches(event)}
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
	requestBody, _ := json.Marshal(map[string]any{"model": config.Model, "prompt": prompt, "stream": false, "think": false, "format": format, "options": map[string]any{"temperature": 0.1, "num_ctx": localAIContext(config)}})
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

func (a *LocalAIAnalyzer) BenchmarkStatus() LocalAIBenchmarkStatus {
	a.benchmarkMu.RLock()
	defer a.benchmarkMu.RUnlock()
	status := a.benchmark
	status.Results = append([]LocalAIBenchmarkModel(nil), status.Results...)
	return status
}

func (a *LocalAIAnalyzer) StartBenchmark(models []string) (LocalAIBenchmarkStatus, error) {
	a.benchmarkMu.Lock()
	if a.benchmark.Running {
		a.benchmarkMu.Unlock()
		return a.BenchmarkStatus(), errors.New("a model benchmark is already running")
	}
	available := make(map[string]bool)
	for _, option := range a.Status().Models {
		available[option.Name] = true
	}
	selected := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model != "" && available[model] && !containsString(selected, model) {
			selected = append(selected, model)
		}
	}
	if len(selected) == 0 {
		a.benchmarkMu.Unlock()
		return a.BenchmarkStatus(), errors.New("choose at least one installed generation model")
	}
	if len(selected) > 6 {
		selected = selected[:6]
	}
	now := time.Now().UTC()
	a.benchmark = LocalAIBenchmarkStatus{Running: true, StartedAt: &now, Source: "Built-in grounded radio evidence cases"}
	a.benchmarkMu.Unlock()
	go a.runBenchmark(selected)
	return a.BenchmarkStatus(), nil
}

func (a *LocalAIAnalyzer) runBenchmark(models []string) {
	cases := []struct {
		event      TransmissionEvent
		modulation string
		family     string
	}{
		{TransmissionEvent{FrequencyHz: 99.7e6, BandwidthHz: 200e3, Modulation: "WFM", Analysis: &SignalIntelligence{SignalFamily: "Analog frequency", Modulation: "WFM", Confidence: .95}}, "WFM", "analog"},
		{TransmissionEvent{FrequencyHz: 162.55e6, BandwidthHz: 12500, Modulation: "NFM", Transcript: ptr("National Weather Service forecast"), Analysis: &SignalIntelligence{SignalFamily: "Analog voice", Modulation: "NFM", Confidence: .91}}, "NFM", "analog"},
		{TransmissionEvent{FrequencyHz: 774.5e6, BandwidthHz: 12500, Modulation: "P25", ProtocolName: ptr("P25 Phase 1"), DecoderMessages: []DecoderMessage{{Protocol: "P25", Summary: "NAC 0x293 trunk control frame"}}, Analysis: &SignalIntelligence{SignalFamily: "Digital voice", Modulation: "P25", Confidence: .96}}, "P25", "digital"},
		{TransmissionEvent{FrequencyHz: 460.025e6, BandwidthHz: 12500, Modulation: "DMR", ProtocolName: ptr("DMR"), DecoderMessages: []DecoderMessage{{Protocol: "DMR", Summary: "Color code 1 slot 2 voice frame"}}, Analysis: &SignalIntelligence{SignalFamily: "Digital voice", Modulation: "DMR", Confidence: .94}}, "DMR", "digital"},
		{TransmissionEvent{FrequencyHz: 315e6, BandwidthHz: 12500, Modulation: "UNKNOWN", Analysis: &SignalIntelligence{SignalFamily: "Unknown", Modulation: "UNKNOWN", Confidence: .18}}, "UNKNOWN", "unknown"},
	}
	a.mu.RLock()
	base := a.config
	a.mu.RUnlock()
	for _, model := range models {
		a.benchmarkMu.Lock()
		a.benchmark.Current = model
		a.benchmarkMu.Unlock()
		result := LocalAIBenchmarkModel{Model: model, Cases: len(cases)}
		elapsed := time.Duration(0)
		for _, item := range cases {
			config := base
			config.Enabled, config.Model = true, model
			started := time.Now()
			analysis, err := a.analyzeWithConfig(context.Background(), item.event, config)
			elapsed += time.Since(started)
			if err != nil {
				result.Errors++
				continue
			}
			modulationOK := strings.EqualFold(analysis.Modulation, item.modulation)
			familyOK := strings.Contains(strings.ToLower(analysis.SignalFamily), item.family)
			if modulationOK && familyOK {
				result.Grounded++
			}
		}
		result.AverageMillis = float64(elapsed.Milliseconds()) / float64(len(cases))
		result.GroundedPercent = float64(result.Grounded) / float64(len(cases)) * 100
		result.StructuredPercent = float64(len(cases)-result.Errors) / float64(len(cases)) * 100
		a.benchmarkMu.Lock()
		a.benchmark.Results = append(a.benchmark.Results, result)
		a.benchmarkMu.Unlock()
	}
	finished := time.Now().UTC()
	a.benchmarkMu.Lock()
	a.benchmark.Running, a.benchmark.Current, a.benchmark.FinishedAt = false, "", &finished
	a.benchmarkMu.Unlock()
}

func containsString(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}

func (a *LocalAIAnalyzer) localReferenceMatches(event TransmissionEvent) []map[string]any {
	if a.profiles == nil {
		return nil
	}
	matches := make([]map[string]any, 0, 8)
	for _, profile := range a.profiles.All() {
		source := "Saved profile · " + profile.Name
		verified, reason, distance, allowed := false, "", (*float64)(nil), true
		isReference := strings.HasPrefix(profile.ID, "localdb-") || strings.Contains(strings.ToLower(profile.Summary), "radioreference") || (profile.ReferenceArea != nil && strings.EqualFold(profile.ReferenceArea.Provider, "RadioReference"))
		if !isReference {
			continue
		}
		if profile.ReferenceArea != nil && strings.EqualFold(profile.ReferenceArea.Provider, "RadioReference") {
			source = "RadioReference · " + profile.Name
			verified, reason, distance, allowed = radioReferenceProfileEligibility(profile, event.Location)
			if !allowed {
				continue
			}
		} else {
			source = "Local catalog · " + profile.Name
		}
		for _, channel := range profile.Channels {
			if math.Abs(channel.FrequencyHz-event.FrequencyHz) > math.Max(1, event.BandwidthHz/2) {
				continue
			}
			match := map[string]any{"name": channel.Name, "frequencyHz": channel.FrequencyHz, "mode": channel.Mode, "source": source, "locationVerified": verified}
			if channel.Decoder != nil {
				match["decoder"] = *channel.Decoder
			}
			if reason != "" {
				match["locationReason"] = reason
			}
			if distance != nil {
				match["distanceMiles"] = *distance
			}
			matches = append(matches, match)
			if len(matches) >= 12 {
				return matches
			}
		}
	}
	return matches
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

func localAIContext(config LocalAIConfig) int {
	if config.ContextLength > 0 {
		return config.ContextLength
	}
	if config.Profile == "deep" {
		return 8192
	}
	if config.Profile == "balanced" {
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
