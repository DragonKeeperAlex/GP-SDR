package app

import "testing"

func TestRelayMixerCombinesEnabledStreamsAndClips(t *testing.T) {
	mixer, err := NewRelayMixer([]RelayStream{
		{ID: "p25", Enabled: true, Volume: 1},
		{ID: "dmr", Enabled: true, Volume: .5},
		{ID: "muted", Enabled: false, Volume: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := mixer.Mix(map[string][]int16{"p25": {30000, 1000}, "dmr": {10000, 1000}, "muted": {32767, 32767}})
	if got[0] != 32767 || got[1] != 1500 {
		t.Fatalf("mixed samples = %v, want [32767 1500]", got)
	}
}

func TestRelayMixerRejectsInvalidVolume(t *testing.T) {
	if _, err := NewRelayMixer([]RelayStream{{ID: "bad", Volume: 3}}); err == nil {
		t.Fatal("expected invalid volume error")
	}
}
