package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (r *Runtime) transcribeP25File(ctx context.Context, path string) (string, error) {
	if strings.EqualFold(filepath.Ext(path), ".wav") {
		return r.transcriber.Transcribe(ctx, path)
	}
	if !strings.EqualFold(filepath.Ext(path), ".mp3") {
		return "", fmt.Errorf("unsupported P25 transcription audio format")
	}
	ffmpeg, err := findTool("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("install ffmpeg on the receiver to transcribe legacy MP3 calls")
	}
	directory, err := os.MkdirTemp("", "gpsdr-p25-convert-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(directory)
	wav := filepath.Join(directory, "call.wav")
	convert, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	command := exec.CommandContext(convert, ffmpeg, "-nostdin", "-hide_banner", "-loglevel", "error", "-threads", "1", "-i", path, "-t", "180", "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-threads", "1", wav)
	if output, err := command.CombinedOutput(); err != nil {
		return "", fmt.Errorf("MP3 conversion failed: %s: %w", truncateText(strings.TrimSpace(string(output)), 500), err)
	}
	return r.transcriber.Transcribe(ctx, wav)
}
