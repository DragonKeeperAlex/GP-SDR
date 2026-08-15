package app

import (
	"context"
	"errors"
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

func NewTranscriber() *Transcriber {
	executable := strings.TrimSpace(os.Getenv("GPSDR_WHISPER_EXECUTABLE"))
	if executable == "" {
		executable, _ = findTool("whisper-cli", "main")
	}
	model := strings.TrimSpace(os.Getenv("GPSDR_WHISPER_MODEL"))
	return &Transcriber{executable: executable, model: model, semaphore: make(chan struct{}, 1)}
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
		"-m", t.model, "-f", wavPath, "-otxt", "-of", outputBase, "-nt", "-np")
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
	return strings.TrimSpace(string(text)), nil
}
