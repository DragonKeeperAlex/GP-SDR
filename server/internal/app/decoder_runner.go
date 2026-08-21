package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DecoderMessage struct {
	DecoderID  string   `json:"decoderID"`
	Protocol   string   `json:"protocol"`
	Summary    string   `json:"summary"`
	Callsigns  []string `json:"callsigns,omitempty"`
	RawText    string   `json:"rawText,omitempty"`
	Confidence float64  `json:"confidence"`
}

func runCandidateDecoder(parent context.Context, decoderID string, audio []int16, audioRate int, iqPath string, frequencyHz float64, spec CaptureSpec) ([]DecoderMessage, error) {
	decoderID = canonicalDecoderID(decoderID)
	ctx, cancel := context.WithTimeout(parent, 35*time.Second)
	defer cancel()
	switch decoderID {
	case "multimon-ng":
		executable, err := findTool("multimon-ng")
		if err != nil {
			return nil, err
		}
		pcm := resamplePCM(audio, audioRate, 22_050)
		var input bytes.Buffer
		if err := binary.Write(&input, binary.LittleEndian, pcm); err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, executable, "-q", "-t", "raw", "-a", "POCSAG512", "-a", "POCSAG1200", "-a", "POCSAG2400", "-a", "FLEX", "-a", "AFSK1200", "-")
		command.Stdin = &input
		output, err := command.CombinedOutput()
		return parseTextDecoderOutput(decoderID, string(output)), decoderCommandError(err, output)
	case "dsd-fme":
		executable, err := findTool("dsd-fme")
		if err != nil {
			return nil, err
		}
		directory, err := os.MkdirTemp("", "gpsdr-dsd-fme-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(directory)
		path := filepath.Join(directory, "discriminator.wav")
		if err := WriteMonoWAV(path, resamplePCM(audio, audioRate, 48_000), 48_000); err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, executable, "-i", path, "-N")
		output, err := command.CombinedOutput()
		return parseTextDecoderOutput(decoderID, string(output)), decoderCommandError(err, output)
	case "rtl-433":
		executable, err := findTool("rtl_433")
		if err != nil {
			return nil, err
		}
		if iqPath == "" {
			return nil, errors.New("rtl_433 requires saved IQ evidence")
		}
		prepared, cleanup, err := prepareUC8DecoderIQ(iqPath, frequencyHz, spec, spec.SampleRateHz)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		command := exec.CommandContext(ctx, executable, "-r", "cu8:"+prepared, "-s", strconv.Itoa(spec.SampleRateHz), "-f", strconv.FormatInt(int64(math.Round(frequencyHz)), 10), "-F", "json")
		output, err := command.CombinedOutput()
		return parseRTL433Output(output), decoderCommandError(err, output)
	case "dump1090":
		executable, err := findAnyTool("dump1090", "dump1090-fa")
		if err != nil {
			return nil, err
		}
		if iqPath == "" {
			return nil, errors.New("dump1090 requires saved IQ evidence")
		}
		prepared, cleanup, err := prepareUC8DecoderIQ(iqPath, frequencyHz, spec, 2_400_000)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		command := exec.CommandContext(ctx, executable, "--ifile", prepared, "--iformat", "UC8", "--raw", "--no-fix")
		output, err := command.CombinedOutput()
		return parseDump1090Output(output), decoderCommandError(err, output)
	case "acarsdec":
		executable, err := findTool("acarsdec")
		if err != nil {
			return nil, err
		}
		directory, err := os.MkdirTemp("", "gpsdr-acars-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(directory)
		path := filepath.Join(directory, "acars.wav")
		if err := WriteMonoWAV(path, resamplePCM(audio, audioRate, 48_000), 48_000); err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, executable, "--output", "oneline:file:path=-", "--sndfile", path)
		output, err := command.CombinedOutput()
		return parseACARSOutput(output), decoderCommandError(err, output)
	case "ais":
		executable, err := findAnyTool("AIS-catcher", "ais-catcher")
		if err != nil {
			return nil, err
		}
		if iqPath == "" {
			return nil, errors.New("AIS-catcher requires saved IQ evidence")
		}
		prepared, cleanup, err := prepareUC8DecoderIQ(iqPath, frequencyHz, spec, spec.SampleRateHz)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		command := exec.CommandContext(ctx, executable, "-r", "CU8", prepared, "-s", strconv.Itoa(spec.SampleRateHz), "-o", "5")
		output, err := command.CombinedOutput()
		return parseAISOutput(output), decoderCommandError(err, output)
	default:
		return nil, fmt.Errorf("live file bridge is not implemented for %s", decoderID)
	}
}

func findAnyTool(names ...string) (string, error) {
	for _, name := range names {
		if executable, err := findTool(name); err == nil {
			return executable, nil
		}
	}
	return "", fmt.Errorf("%s is not installed", strings.Join(names, " or "))
}

// prepareUC8DecoderIQ shifts the observed channel from GP-SDR's intentional
// hardware-center offset to baseband, resamples when a decoder requires a
// fixed rate, and converts HackRF signed bytes to decoder-friendly UC8.
func prepareUC8DecoderIQ(path string, frequencyHz float64, spec CaptureSpec, targetRate int) (string, func(), error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", func() {}, err
	}
	if len(data) < 2 || spec.SampleRateHz <= 0 || targetRate <= 0 {
		return "", func() {}, errors.New("IQ evidence metadata is invalid")
	}
	format := ComplexSigned8
	if strings.EqualFold(filepath.Ext(path), ".cu8") {
		format = ComplexUnsigned8
	}
	outputCount := int(math.Floor(float64(len(data)/2) * float64(targetRate) / float64(spec.SampleRateHz)))
	if outputCount < 1 {
		return "", func() {}, errors.New("IQ evidence is too short")
	}
	output := make([]byte, outputCount*2)
	offsetHz := frequencyHz - float64(spec.CenterFrequencyHz)
	for index := 0; index < outputCount; index++ {
		position := float64(index) * float64(spec.SampleRateHz) / float64(targetRate)
		left := int(position)
		if left >= len(data)/2 {
			left = len(data)/2 - 1
		}
		next := left + 1
		if next >= len(data)/2 {
			next = left
		}
		fraction := position - float64(left)
		i0, q0 := decoderIQPair(data, left, format)
		i1, q1 := decoderIQPair(data, next, format)
		iValue, qValue := i0*(1-fraction)+i1*fraction, q0*(1-fraction)+q1*fraction
		phase := -2 * math.Pi * offsetHz * position / float64(spec.SampleRateHz)
		cosine, sine := math.Cos(phase), math.Sin(phase)
		shiftedI := iValue*cosine - qValue*sine
		shiftedQ := iValue*sine + qValue*cosine
		output[index*2] = byte(clampInt(int(math.Round(shiftedI+128)), 0, 255))
		output[index*2+1] = byte(clampInt(int(math.Round(shiftedQ+128)), 0, 255))
	}
	directory, err := os.MkdirTemp("", "gpsdr-decoder-iq-*")
	if err != nil {
		return "", func() {}, err
	}
	prepared := filepath.Join(directory, fmt.Sprintf("%.0fHz-%dsps.cu8", frequencyHz, targetRate))
	if err := os.WriteFile(prepared, output, 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return "", func() {}, err
	}
	return prepared, func() { _ = os.RemoveAll(directory) }, nil
}

func decoderIQPair(data []byte, index int, format SampleFormat) (float64, float64) {
	iValue, qValue := float64(data[index*2]), float64(data[index*2+1])
	if format == ComplexSigned8 {
		return float64(int8(data[index*2])), float64(int8(data[index*2+1]))
	}
	return iValue - 127.5, qValue - 127.5
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func decoderCommandError(err error, output []byte) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if len(message) > 1200 {
		message = message[len(message)-1200:]
	}
	if message == "" {
		message = err.Error()
	}
	return errors.New(message)
}

func resamplePCM(samples []int16, sourceRate, targetRate int) []int16 {
	if len(samples) == 0 || sourceRate <= 0 || targetRate <= 0 || sourceRate == targetRate {
		return append([]int16(nil), samples...)
	}
	count := int(float64(len(samples)) * float64(targetRate) / float64(sourceRate))
	if count < 1 {
		return nil
	}
	result := make([]int16, count)
	for index := range result {
		position := float64(index) * float64(sourceRate) / float64(targetRate)
		left := int(position)
		if left >= len(samples)-1 {
			result[index] = samples[len(samples)-1]
			continue
		}
		fraction := position - float64(left)
		result[index] = int16(float64(samples[left])*(1-fraction) + float64(samples[left+1])*fraction)
	}
	return result
}

var decoderProtocolPattern = regexp.MustCompile(`(?i)\b(P25|DMR|NXDN|D-?STAR|YSF|POCSAG(?:512|1200|2400)?|FLEX|AFSK1200|MDC1200|DTMF)\b`)

func parseTextDecoderOutput(decoderID, output string) []DecoderMessage {
	lines := strings.Split(strings.ReplaceAll(output, "\r", "\n"), "\n")
	result := make([]DecoderMessage, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		match := decoderProtocolPattern.FindString(line)
		if line == "" || match == "" {
			continue
		}
		if len(line) > 1000 {
			line = line[:1000]
		}
		result = append(result, DecoderMessage{DecoderID: decoderID, Protocol: strings.ToUpper(match), Summary: line,
			Callsigns: ExtractCallsigns(line), RawText: line, Confidence: .96})
		if len(result) >= 25 {
			break
		}
	}
	return result
}

func parseRTL433Output(output []byte) []DecoderMessage {
	result := make([]DecoderMessage, 0)
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil || len(item) == 0 {
			continue
		}
		model := strings.TrimSpace(fmt.Sprint(item["model"]))
		identifier := strings.TrimSpace(fmt.Sprint(item["id"]))
		summary := model
		if identifier != "" && identifier != "<nil>" {
			summary += " · ID " + identifier
		}
		if summary == "" {
			summary = "Decoded ISM sensor frame"
		}
		raw, _ := json.Marshal(item)
		result = append(result, DecoderMessage{DecoderID: "rtl-433", Protocol: "ISM sensor", Summary: summary, RawText: string(raw), Confidence: .99})
		if len(result) >= 25 {
			break
		}
	}
	return result
}

var modeSFramePattern = regexp.MustCompile(`(?m)[*@]([0-9A-Fa-f]{14}|[0-9A-Fa-f]{28});`)

func parseDump1090Output(output []byte) []DecoderMessage {
	frames := modeSFramePattern.FindAllSubmatch(output, 25)
	result := make([]DecoderMessage, 0, len(frames))
	for _, match := range frames {
		frame := strings.ToUpper(string(match[1]))
		summary := "Mode S frame"
		if len(frame) >= 8 {
			summary += " · ICAO " + frame[2:8]
		}
		result = append(result, DecoderMessage{DecoderID: "dump1090", Protocol: "ADS-B / Mode S", Summary: summary, RawText: frame, Confidence: .99})
	}
	return result
}

func parseACARSOutput(output []byte) []DecoderMessage {
	result := make([]DecoderMessage, 0)
	for _, raw := range bytes.Split(output, []byte{'\n'}) {
		line := strings.TrimSpace(string(raw))
		if !strings.HasPrefix(line, "#") || len(line) < 8 {
			continue
		}
		if len(line) > 1000 {
			line = line[:1000]
		}
		result = append(result, DecoderMessage{DecoderID: "acarsdec", Protocol: "ACARS", Summary: line, Callsigns: ExtractCallsigns(line), RawText: line, Confidence: .98})
		if len(result) >= 25 {
			break
		}
	}
	return result
}

func parseAISOutput(output []byte) []DecoderMessage {
	result := make([]DecoderMessage, 0)
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil || len(item) == 0 {
			continue
		}
		mmsi := decoderMapValue(item, "mmsi", "MMSI")
		name := decoderMapValue(item, "shipname", "ship_name", "name", "NAME")
		callsign := decoderMapValue(item, "callsign", "call_sign", "CALLSIGN")
		parts := []string{"AIS vessel frame"}
		if mmsi != "" {
			parts = append(parts, "MMSI "+mmsi)
		}
		if name != "" {
			parts = append(parts, name)
		}
		callsigns := []string{}
		if callsign != "" {
			callsigns = append(callsigns, callsign)
		}
		raw, _ := json.Marshal(item)
		result = append(result, DecoderMessage{DecoderID: "ais", Protocol: "AIS", Summary: strings.Join(parts, " · "), Callsigns: callsigns, RawText: string(raw), Confidence: .99})
		if len(result) >= 25 {
			break
		}
	}
	return result
}

func decoderMapValue(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}
