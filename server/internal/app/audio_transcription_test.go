package app

import (
	"context"
	"math"
	"math/rand"
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
	samples := make([]int16, 16000)
	for i := range samples {
		samples[i] = int16(100 * math.Sin(float64(i)*.1))
	}
	if err := WriteMonoWAV(wav, samples, 16000); err != nil {
		t.Fatal(err)
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

func TestTranscriptionNoSpeechGate(t *testing.T) {
	if !transcriptionNoSpeech(make([]int16, 16000), 16000) {
		t.Fatal("silence passed gate")
	}
	rng := rand.New(rand.NewSource(4))
	noise, weakVoice := make([]int16, 32000), make([]int16, 32000)
	for i := range noise {
		noise[i] = int16(rng.NormFloat64() * 1000)
		weakVoice[i] = int16((4 + 4*math.Sin(float64(i)/2400)) * math.Sin(float64(i)*.1))
	}
	if !transcriptionNoSpeech(noise, 16000) {
		t.Fatal("stationary broadband noise passed gate")
	}
	if transcriptionNoSpeech(weakVoice, 16000) {
		t.Fatal("quiet modulated waveform incorrectly gated")
	}
	if got := cleanRadioTranscript("[sounds of water]\nK6ABC, check the engine.\n(engine noises)"); got != "K6ABC, check the engine." {
		t.Fatalf("unexpected cleaned transcript: %q", got)
	}
}

// Opt-in integration check against an installed model and a speech WAV fixture.
func TestInstalledWhisperSpeech(t *testing.T) {
	wav, model, executable := os.Getenv("GPSDR_TEST_SPEECH_WAV"), os.Getenv("GPSDR_TEST_WHISPER_MODEL"), os.Getenv("GPSDR_TEST_WHISPER_BIN")
	if wav == "" || model == "" || executable == "" {
		t.Skip("installed Whisper speech fixture not configured")
	}
	transcriber := &Transcriber{executable: executable, model: model, semaphore: make(chan struct{}, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	text, err := transcriber.Transcribe(ctx, wav)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) < 10 {
		t.Fatalf("speech not transcribed: %q", text)
	}
	t.Logf("installed model transcript: %s", text)
}
