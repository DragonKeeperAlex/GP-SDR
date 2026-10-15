package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelayVolumePersistsIncludingMute(t *testing.T) {
	dir := t.TempDir()
	for _, volume := range []float64{.17, 0, .65} {
		r := NewRelayRuntime(NewAudioHub(), dir)
		if _, err := r.SetOutputVolume(volume); err != nil {
			t.Fatal(err)
		}
		rebooted := NewRelayRuntime(NewAudioHub(), dir)
		if got := rebooted.Status().OutputVolume; got != volume {
			t.Fatalf("reloaded %v, want %v", got, volume)
		}
		if rebooted.Status().Active {
			t.Fatal("volume restore started transmission")
		}
	}
}

func TestRelayVolumeSaveFailurePreservesPriorValue(t *testing.T) {
	r := NewRelayRuntime(NewAudioHub(), t.TempDir())
	before := r.Status().OutputVolume
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	r.settingsPath = filepath.Join(blocker, "relay-output.json")
	if _, err := r.SetOutputVolume(.9); err == nil {
		t.Fatal("expected persistence error")
	}
	if r.Status().OutputVolume != before {
		t.Fatal("failed save changed active gain")
	}
}

func TestRelayCorruptVolumeLoadsMuted(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Data"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Data", "relay-output.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	r := NewRelayRuntime(NewAudioHub(), dir)
	if r.Status().OutputVolume != 0 {
		t.Fatal("corrupt setting must load muted")
	}
}
