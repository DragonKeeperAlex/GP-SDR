package app

import "testing"

func TestMapperExcludesNoisyHackRFButLeavesOtherDevicesEligible(t *testing.T) {
	noisySerial := "000000000000000024b862dc3140c5cb"
	if reason := mapperExclusionReason(SDRDevice{Kind: "HackRF", Serial: &noisySerial}); reason == "" {
		t.Fatal("expected c5cb to be excluded from Mapper")
	}
	goodSerial := "0000000000000000922c63dc217ea447"
	if reason := mapperExclusionReason(SDRDevice{Kind: "HackRF", Serial: &goodSerial}); reason != "" {
		t.Fatalf("unexpected Mapper exclusion: %s", reason)
	}
}
