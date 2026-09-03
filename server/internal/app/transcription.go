package app

import (
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TranscriptionStatus struct {
	State      string  `json:"state"`
	Executable *string `json:"executable"`
	Model      *string `json:"model"`
	Note       string  `json:"note"`
}

type Transcriber struct {
	executable string
	model      string
	semaphore  chan struct{}
}

func NewTranscriber(dataDirectories ...string) *Transcriber {
	executable := strings.TrimSpace(os.Getenv("GPSDR_WHISPER_EXECUTABLE"))
	if executable == "" {
		executable, _ = findTool("whisper-cli", "main")
	}
	model := strings.TrimSpace(os.Getenv("GPSDR_WHISPER_MODEL"))
	if model == "" {
		model = findWhisperModel(dataDirectories...)
	}
	return &Transcriber{executable: executable, model: model, semaphore: make(chan struct{}, 1)}
}

func findWhisperModel(dataDirectories ...string) string {
	candidates := make([]string, 0, 16)
	for _, directory := range dataDirectories {
		if strings.TrimSpace(directory) != "" {
			for _, name := range []string{"ggml-base.en.bin", "ggml-small.en.bin", "ggml-tiny.en.bin", "ggml-base.bin", "ggml-small.bin", "ggml-tiny.bin"} {
				candidates = append(candidates, filepath.Join(directory, "Components", "Whisper", name))
			}
		}
	}
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		for _, name := range []string{"ggml-base.en.bin", "ggml-small.en.bin", "ggml-tiny.en.bin"} {
			candidates = append(candidates, filepath.Join(base, "..", "Models", name), filepath.Join(base, "..", "Resources", "Models", name))
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && !info.IsDir() && info.Size() > 10*1024*1024 {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func (t *Transcriber) Status() TranscriptionStatus {
	if t.executable == "" {
		return TranscriptionStatus{State: "setup", Note: "Install whisper.cpp or bundle whisper-cli."}
	}
	if t.model == "" {
		return TranscriptionStatus{State: "setup", Executable: ptr(t.executable), Note: "Choose a local model or set GPSDR_WHISPER_MODEL."}
	}
	if _, err := os.Stat(t.model); err != nil {
		return TranscriptionStatus{State: "error", Executable: ptr(t.executable), Model: ptr(t.model), Note: "The configured Whisper model was not found."}
	}
	return TranscriptionStatus{State: "ready", Executable: ptr(t.executable), Model: ptr(t.model), Note: "Offline transcription is ready."}
}

func (t *Transcriber) Transcribe(parent context.Context, wavPath string) (string, error) {
	samples, rate, err := readPCM16WAV(wavPath)
	if err != nil {
		return "", err
	}
	if transcriptionNoSpeech(samples, rate) {
		return "", nil
	}
	if t.Status().State != "ready" {
		return "", errors.New("offline transcription is not configured")
	}
	select {
	case t.semaphore <- struct{}{}:
		defer func() { <-t.semaphore }()
	case <-parent.Done():
		return "", parent.Err()
	}
	contextWithTimeout, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	outputDirectory, err := os.MkdirTemp("", "gpsdr-transcript-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(outputDirectory)
	outputBase := filepath.Join(outputDirectory, "transcript")
	command := exec.CommandContext(contextWithTimeout, t.executable,
		"-m", t.model, "-f", wavPath, "-l", "en", "--prompt", "Two-way radio traffic. Callsigns may be spoken with NATO phonetics.", "-otxt", "-of", outputBase, "-nt", "-np")
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	text, err := os.ReadFile(outputBase + ".txt")
	if err != nil {
		return "", err
	}
	return cleanRadioTranscript(string(text)), nil
}

// Conservative cheap gate, not a speech classifier. Reject only digital silence
// and long, stationary broadband noise; never reject merely quiet speech.
func transcriptionNoSpeech(samples []int16, rate int) bool {
	if len(samples) == 0 || rate <= 0 {
		return true
	}
	var mean, energy float64
	for _, sample := range samples {
		mean += float64(sample)
	}
	mean /= float64(len(samples))
	crossings := 0
	frames := []float64{}
	frameSize := maxInt(1, rate/50)
	var frameEnergy float64
	for index, sample := range samples {
		x := float64(sample) - mean
		energy += x * x
		frameEnergy += x * x
		if index > 0 && (x >= 0) != (float64(samples[index-1])-mean >= 0) {
			crossings++
		}
		if (index+1)%frameSize == 0 {
			frames = append(frames, math.Sqrt(frameEnergy/float64(frameSize)))
			frameEnergy = 0
		}
	}
	if math.Sqrt(energy/float64(len(samples))) < 1 {
		return true
	}
	if len(frames) < 25 {
		return false
	}
	var rmsMean, variance float64
	for _, rms := range frames {
		rmsMean += rms
	}
	rmsMean /= float64(len(frames))
	for _, rms := range frames {
		variance += (rms - rmsMean) * (rms - rmsMean)
	}
	cv := math.Sqrt(variance/float64(len(frames))) / rmsMean
	return cv < .07 && float64(crossings)/float64(len(samples)-1) > .40
}

// Remove standalone sound-effect captions, not spoken sentences about those
// sounds. Unknown annotations are retained rather than guessing they are junk.
func cleanRadioTranscript(raw string) string {
	lines := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		key := strings.ToLower(strings.Trim(line, "[]() .*\t"))
		switch key {
		case "", "blank_audio", "no speech", "silence", "static", "noise", "music", "water sounds", "sounds of water", "water flowing", "engine noises", "engine noise", "engine sounds", "wind blowing", "wind noise", "applause", "inaudible":
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
