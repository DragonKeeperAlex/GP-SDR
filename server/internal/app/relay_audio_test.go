package app

import (
	"reflect"
	"testing"
)

func TestRelayAudioUsesExplicitUSBDeviceAndPCMFormat(t *testing.T) {
	got := (RelayAudioConfig{Device: "plughw:2,0", SampleRate: 8000, Channels: 1}).AplayArgs()
	want := []string{"-D", "plughw:2,0", "-q", "-t", "raw", "-f", "S16_LE", "-r", "8000", "-c", "1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("aplay args = %v, want %v", got, want)
	}
}
