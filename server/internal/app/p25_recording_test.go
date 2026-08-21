package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindAndArchiveP25Recording(t *testing.T) {
	root := t.TempDir()
	started := time.Now().Add(-time.Second)
	source := filepath.Join(root, "20260820_System_TO_1201_FROM_55.wav")
	if err := os.WriteFile(source, []byte("radio audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if found := findP25Recording(root, 1201, started); found != source {
		t.Fatalf("unexpected recording match %q", found)
	}
	destination, err := archiveP25Recording(t.TempDir(), source, 1201, started)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "radio audio" {
		t.Fatalf("unexpected archived recording: %q %v", data, err)
	}
}
