package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAudioHubPublishesIndependentFrames(t *testing.T) {
	hub := NewAudioHub()
	frames, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	samples := []int16{100, -200, 300}
	hub.Publish(AudioFrame{ChannelID: "test", SampleRate: 48_000, Samples: samples})
	samples[0] = 999
	select {
	case frame := <-frames:
		if frame.ChannelID != "test" || frame.SampleRate != 48_000 || frame.Samples[0] != 100 {
			t.Fatalf("unexpected audio frame: %#v", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("audio frame was not published")
	}
}

func TestTranscriberRunsConfiguredOfflineCommand(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "fake-whisper")
	model := filepath.Join(directory, "model.bin")
	wav := filepath.Join(directory, "audio.wav")
	script := `#!/bin/sh
output=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-of" ]; then
    shift
    output="$1"
  fi
  shift
done
printf 'decoded locally' > "${output}.txt"
`
	if err := os.WriteFile(executable, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{model, wav} {
		if err := os.WriteFile(item, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	transcriber := &Transcriber{executable: executable, model: model, semaphore: make(chan struct{}, 1)}
	if status := transcriber.Status(); status.State != "ready" {
		t.Fatalf("transcriber is not ready: %#v", status)
	}
	text, err := transcriber.Transcribe(context.Background(), wav)
	if err != nil {
		t.Fatal(err)
	}
	if text != "decoded locally" {
		t.Fatalf("unexpected transcript %q", text)
	}
}
