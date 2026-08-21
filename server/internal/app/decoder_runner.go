package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
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

func runCandidateDecoder(parent context.Context, decoderID string, audio []int16, audioRate int, iqPath string, spec CaptureSpec) ([]DecoderMessage, error) {
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
		command := exec.CommandContext(ctx, executable, "-r", iqPath, "-s", strconv.Itoa(spec.SampleRateHz), "-f", strconv.FormatInt(spec.CenterFrequencyHz, 10), "-F", "json")
		output, err := command.CombinedOutput()
		return parseRTL433Output(output), decoderCommandError(err, output)
	default:
		return nil, fmt.Errorf("live file bridge is not implemented for %s", decoderID)
	}
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
