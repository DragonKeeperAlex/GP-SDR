package app

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// AnalyzeSignalIQ performs bounded, offline waveform classification on the
// selected channel. It is intentionally evidence-based and conservative: the
// result may select a demodulator, while protocol names remain candidates until
// a real decoder confirms frames.
func AnalyzeSignalIQ(data []byte, format SampleFormat, sampleRate int, offsetHz, bandwidthHz float64) SignalIntelligence {
	result := SignalIntelligence{Engine: "GP-SDR local IQ classifier", Modulation: "UNKNOWN", SignalFamily: "Unknown", Confidence: .25,
		Summary: "Insufficient IQ evidence for automatic classification."}
	if len(data) < 4096 || sampleRate <= 0 {
		return result
	}
	channelRate := int(math.Max(48_000, math.Min(240_000, bandwidthHz*3)))
	if channelRate > sampleRate {
		channelRate = sampleRate
	}
	decimation := sampleRate / channelRate
	if decimation < 1 {
		decimation = 1
	}
	actualRate := sampleRate / decimation
	phaseStep := -2 * math.Pi * offsetHz / float64(sampleRate)
	oscI, oscQ := 1.0, 0.0
	stepI, stepQ := math.Cos(phaseStep), math.Sin(phaseStep)
	amps := make([]float64, 0, minInt(len(data)/(2*decimation), 200_000))
	phaseDeltas := make([]float64, 0, cap(amps))
	accI, accQ, count := 0.0, 0.0, 0
	previousI, previousQ, havePrevious := 0.0, 0.0, false
	for index := 0; index+1 < len(data) && len(amps) < 200_000; index += 2 {
		i, q := iqSample(data[index], data[index+1], format)
		mixedI := i*oscI - q*oscQ
		mixedQ := i*oscQ + q*oscI
		nextI := oscI*stepI - oscQ*stepQ
		oscQ = oscI*stepQ + oscQ*stepI
		oscI = nextI
		accI, accQ, count = accI+mixedI, accQ+mixedQ, count+1
		if count < decimation {
			continue
		}
		filteredI, filteredQ := accI/float64(count), accQ/float64(count)
		accI, accQ, count = 0, 0, 0
		amps = append(amps, math.Hypot(filteredI, filteredQ))
		if havePrevious {
			realPart := filteredI*previousI + filteredQ*previousQ
			imagPart := filteredQ*previousI - filteredI*previousQ
			phaseDeltas = append(phaseDeltas, math.Atan2(imagPart, realPart))
		}
		previousI, previousQ, havePrevious = filteredI, filteredQ, true
	}
	if len(phaseDeltas) < 256 {
		return result
	}
	ampMean, ampStd := meanStd(amps)
	phaseMean, phaseStd := meanStd(phaseDeltas)
	ampVariation := ampStd / math.Max(ampMean, 1e-9)
	deviationHz := phaseStd * float64(actualRate) / (2 * math.Pi)
	cluster := phaseClusterScore(phaseDeltas, phaseMean, phaseStd)
	result.AmplitudeVariation = ampVariation
	result.EstimatedDeviationHz = deviationHz
	result.PhaseClusterScore = cluster

	switch {
	case ampVariation > .22 && deviationHz < 8_000:
		result.Modulation, result.SignalFamily, result.Confidence = "AM", "Analog amplitude", clamp(.58+(ampVariation-.22)*.7, .58, .91)
		result.Evidence = []string{fmt.Sprintf("%.0f%% envelope variation", ampVariation*100), fmt.Sprintf("%.1f kHz phase spread", deviationHz/1000)}
	case ampVariation < .20 && cluster >= .46 && deviationHz >= 700:
		result.Modulation, result.SignalFamily, result.Confidence = "DIGITAL", "FSK / constant-envelope digital candidate", clamp(.58+(cluster-.46)*1.1, .58, .91)
		result.Evidence = []string{fmt.Sprintf("%.0f%% phase clustering", cluster*100), fmt.Sprintf("%.1f kHz deviation estimate", deviationHz/1000)}
	case bandwidthHz >= 100_000 || deviationHz >= 18_000:
		result.Modulation, result.SignalFamily, result.Confidence = "WFM", "Analog frequency", clamp(.60+math.Min(deviationHz/100_000, .25), .60, .88)
		result.Evidence = []string{fmt.Sprintf("%.1f kHz deviation estimate", deviationHz/1000), fmt.Sprintf("%.0f kHz channel width", bandwidthHz/1000)}
	case deviationHz >= 400:
		result.Modulation, result.SignalFamily, result.Confidence = "NFM", "Analog frequency", clamp(.57+math.Min(deviationHz/40_000, .24), .57, .84)
		result.Evidence = []string{fmt.Sprintf("%.1f kHz deviation estimate", deviationHz/1000), fmt.Sprintf("%.0f%% envelope variation", ampVariation*100)}
	default:
		result.Modulation, result.SignalFamily, result.Confidence = "CARRIER", "Unmodulated or weak carrier", .48
		result.Evidence = []string{"Low envelope and phase variation"}
	}
	result.Summary = fmt.Sprintf("Local IQ analysis suggests %s (%d%% confidence).", result.SignalFamily, int(math.Round(result.Confidence*100)))
	return result
}

func meanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		delta := value - mean
		variance += delta * delta
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}

func phaseClusterScore(values []float64, mean, standardDeviation float64) float64 {
	if len(values) == 0 || standardDeviation < 1e-9 {
		return 0
	}
	const bins = 48
	histogram := make([]int, bins)
	used := 0
	span := standardDeviation * 3
	for _, value := range values {
		if value < mean-span || value > mean+span {
			continue
		}
		index := int((value - (mean - span)) / (2 * span) * bins)
		if index >= bins {
			index = bins - 1
		}
		if index >= 0 {
			histogram[index]++
			used++
		}
	}
	if used == 0 {
		return 0
	}
	sort.Sort(sort.Reverse(sort.IntSlice(histogram)))
	return float64(histogram[0]+histogram[1]+histogram[2]+histogram[3]) / float64(used)
}

var strictCallsignPattern = regexp.MustCompile(`\b[AKNW][A-Z0-9]?\d[A-Z]{1,3}\b`)

var spokenCallsignTokens = map[string]string{
	"ALFA": "A", "ALPHA": "A", "BRAVO": "B", "CHARLIE": "C", "DELTA": "D", "ECHO": "E", "FOXTROT": "F",
	"GOLF": "G", "HOTEL": "H", "INDIA": "I", "JULIET": "J", "JULIETT": "J", "KILO": "K", "LIMA": "L",
	"MIKE": "M", "NOVEMBER": "N", "OSCAR": "O", "PAPA": "P", "QUEBEC": "Q", "ROMEO": "R", "SIERRA": "S",
	"TANGO": "T", "UNIFORM": "U", "VICTOR": "V", "WHISKEY": "W", "XRAY": "X", "X-RAY": "X", "YANKEE": "Y", "ZULU": "Z",
	"ZERO": "0", "ONE": "1", "TWO": "2", "THREE": "3", "FOUR": "4", "FIVE": "5", "SIX": "6", "SEVEN": "7", "EIGHT": "8", "NINE": "9",
}

// ExtractCallsigns recognizes strict US-format callsigns in normal text and in
// contiguous NATO-phonetic spelling such as "kilo six alpha bravo charlie".
func ExtractCallsigns(text string) []string {
	upper := strings.ToUpper(text)
	found := make(map[string]bool)
	for _, call := range strictCallsignPattern.FindAllString(upper, -1) {
		found[call] = true
	}
	words := strings.FieldsFunc(upper, func(r rune) bool { return !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' })
	buffer := strings.Builder{}
	flush := func() {
		candidate := buffer.String()
		if strictCallsignPattern.MatchString(candidate) && strictCallsignPattern.FindString(candidate) == candidate {
			found[candidate] = true
		}
		buffer.Reset()
	}
	for _, word := range words {
		value, ok := spokenCallsignTokens[word]
		if !ok && len(word) == 1 && ((word[0] >= 'A' && word[0] <= 'Z') || (word[0] >= '0' && word[0] <= '9')) {
			value, ok = word, true
		}
		if !ok {
			flush()
			continue
		}
		buffer.WriteString(value)
		if buffer.Len() > 7 {
			flush()
		}
	}
	flush()
	result := make([]string, 0, len(found))
	for call := range found {
		result = append(result, call)
	}
	sort.Strings(result)
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
