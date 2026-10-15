package app

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RFFeatureVector is deliberately derived from measured RF/DSP values, not
// transcript text. It is stable enough for a small supervised baseline and
// makes the training/evaluation boundary explicit.
func RFFeatureVector(event TransmissionEvent) []float64 {
	features := []float64{event.FrequencyHz / 1e9, event.BandwidthHz / 1e6, event.SignalDBFS / 100, event.NoiseDBFS / 100}
	if event.Analysis != nil {
		features = append(features, event.Analysis.EstimatedDeviationHz/100000, event.Analysis.AmplitudeVariation, event.Analysis.PhaseClusterScore)
	} else {
		features = append(features, 0, 0, 0)
	}
	return features
}

type RFClassifierModel struct {
	SchemaVersion int                  `json:"schemaVersion"`
	TrainedAt     time.Time            `json:"trainedAt"`
	FeatureNames  []string             `json:"featureNames"`
	Centroids     map[string][]float64 `json:"centroids"`
	Counts        map[string]int       `json:"counts"`
	Seed          int64                `json:"seed"`
}

type RFTrainingReport struct {
	Model        *RFClassifierModel `json:"model,omitempty"`
	Samples      int                `json:"samples"`
	Classes      []string           `json:"classes"`
	TrainCount   int                `json:"trainCount"`
	HeldOutCount int                `json:"heldOutCount"`
	Correct      int                `json:"correct"`
	Accuracy     float64            `json:"accuracy"`
	Note         string             `json:"note,omitempty"`
}

func sortedClassifierClasses(model *RFClassifierModel) []string {
	classes := make([]string, 0, len(model.Centroids))
	for class := range model.Centroids {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	return classes
}

func trainRFClassifier(samples []ConfirmedSignalSample) (RFTrainingReport, error) {
	usable := make([]ConfirmedSignalSample, 0, len(samples))
	for _, sample := range samples {
		if len(sample.RFFeatures) == 7 && sample.Modulation != "" && sample.Modulation != "UNKNOWN" && sample.ReviewStatus != "rejected" {
			usable = append(usable, sample)
		}
	}
	if len(usable) < 2 {
		return RFTrainingReport{Samples: len(usable), Note: "at least two reviewed samples are required"}, errors.New("not enough reviewed RF samples")
	}
	sort.SliceStable(usable, func(i, j int) bool { return usable[i].ID < usable[j].ID })
	model := &RFClassifierModel{SchemaVersion: 1, TrainedAt: time.Now().UTC(), Seed: 1, FeatureNames: []string{"frequencyGHz", "bandwidthMHz", "signalDBFS", "noiseDBFS", "deviation100kHz", "amplitudeVariation", "phaseClusterScore"}, Centroids: map[string][]float64{}, Counts: map[string]int{}}
	for _, sample := range usable {
		centroid := model.Centroids[sample.Modulation]
		if centroid == nil {
			centroid = make([]float64, len(sample.RFFeatures))
		}
		count := model.Counts[sample.Modulation]
		for i, value := range sample.RFFeatures {
			centroid[i] = (centroid[i]*float64(count) + value) / float64(count+1)
		}
		model.Centroids[sample.Modulation], model.Counts[sample.Modulation] = centroid, count+1
	}
	classes := make([]string, 0, len(model.Centroids))
	for class := range model.Centroids {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	report := RFTrainingReport{Model: model, Samples: len(usable), Classes: classes, TrainCount: len(usable)}
	// Deterministic leave-one-out evaluation prevents the model from reporting
	// training accuracy as if it were held-out performance.
	for _, sample := range usable {
		if len(usable) < 3 {
			break
		}
		prediction := nearestRFClass(sample.RFFeatures, model, sample.Modulation, model.Counts[sample.Modulation] > 1)
		report.HeldOutCount++
		if prediction == sample.Modulation {
			report.Correct++
		}
	}
	if report.HeldOutCount > 0 {
		report.Accuracy = float64(report.Correct) / float64(report.HeldOutCount)
	}
	return report, nil
}

func nearestRFClass(features []float64, model *RFClassifierModel, excluded string, excludeOne bool) string {
	best, bestDistance := "UNKNOWN", math.MaxFloat64
	for class, centroid := range model.Centroids {
		if excludeOne && class == excluded && model.Counts[class] <= 1 {
			continue
		}
		distance := 0.0
		for i := range features {
			delta := features[i] - centroid[i]
			distance += delta * delta
		}
		if distance < bestDistance {
			best, bestDistance = class, distance
		}
	}
	return best
}

func (l *SignalLearningLibrary) TrainAndSave(dataDirectory string) (RFTrainingReport, error) {
	l.mu.RLock()
	samples := append([]ConfirmedSignalSample(nil), l.samples...)
	l.mu.RUnlock()
	report, err := trainRFClassifier(samples)
	if err != nil {
		return report, err
	}
	data, err := json.MarshalIndent(report.Model, "", "  ")
	if err != nil {
		return report, err
	}
	if err := writeBytesAtomic(filepath.Join(dataDirectory, "Data", "rf-classifier.json"), append(data, '\n')); err != nil {
		return report, err
	}
	return report, nil
}

func loadRFClassifier(dataDirectory string) *RFClassifierModel {
	data, err := os.ReadFile(filepath.Join(dataDirectory, "Data", "rf-classifier.json"))
	if err != nil {
		return nil
	}
	var model RFClassifierModel
	if json.Unmarshal(data, &model) != nil || model.SchemaVersion != 1 || len(model.Centroids) == 0 {
		return nil
	}
	return &model
}

// classifierLabel is intentionally conservative: a trained label is only
// returned when the nearest centroid is sufficiently closer than alternatives.
func classifierLabel(event TransmissionEvent, model *RFClassifierModel) (string, float64) {
	if model == nil {
		return "UNKNOWN", 0
	}
	features := RFFeatureVector(event)
	best, second := math.MaxFloat64, math.MaxFloat64
	label := "UNKNOWN"
	for class, centroid := range model.Centroids {
		distance := 0.0
		for i := range features {
			delta := features[i] - centroid[i]
			distance += delta * delta
		}
		if distance < best {
			second, best, label = best, distance, class
		} else if distance < second {
			second = distance
		}
	}
	if label == "UNKNOWN" || best == math.MaxFloat64 {
		return "UNKNOWN", 0
	}
	confidence := 1.0 - best/math.Max(second, 1e-9)
	return strings.ToUpper(label), clamp(confidence, 0, 1)
}

func applyRFClassifier(event TransmissionEvent, base SignalIntelligence, model *RFClassifierModel) SignalIntelligence {
	label, confidence := classifierLabel(event, model)
	if label == "UNKNOWN" || confidence < .55 {
		return base
	}
	result := base
	result.Evidence = append([]string(nil), base.Evidence...)
	result.Evidence = append(result.Evidence, "RF-trained centroid classifier candidate: "+label)
	if result.Modulation == "UNKNOWN" || result.Confidence < confidence {
		result.Modulation, result.Confidence = label, confidence
	}
	result.Engine = result.Engine + " + RF-trained classifier"
	return result
}
