package app

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestP25LegacyMP3ConversionPreservesOriginal(t *testing.T) {
	ffmpeg, err := findTool("ffmpeg")
	if err != nil {
		t.Skip("optional ffmpeg not installed")
	}
	path := filepath.Join(t.TempDir(), "silence.mp3")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, ffmpeg, "-nostdin", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono", "-t", "0.2", "-c:a", "libmp3lame", path).CombinedOutput(); err != nil {
		t.Fatalf("create silence fixture: %s %v", output, err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{transcriber: &Transcriber{}}
	text, err := r.transcribeP25File(ctx, path)
	if err != nil || text != "" {
		t.Fatalf("converted silence not gated: %q %v", text, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(original) != sha256.Sum256(after) {
		t.Fatal("original MP3 changed")
	}
}
