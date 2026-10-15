package app

import (
	"strings"
	"testing"
)

func TestHackRFOperatingFloorCoversNativeAndSoapy(t *testing.T) {
	for _, driver := range []string{"hackrf_transfer", "SoapySDR:hackrf"} {
		device := SDRDevice{Kind: "HackRF", Driver: driver}
		for _, requested := range []int{0, 1_000_000, 2_400_000, 5_000_000} {
			if got := compatibleUserSampleRate(device, requested, 1_000_000); got < 8_000_000 {
				t.Fatalf("%s request %d selected %d", driver, requested, got)
			}
			profile := ScanProfile{Settings: SurveySettings{P25SampleRateHz: requested}}
			if got := effectiveOP25DeviceRate(profile, device); got != 8_000_000 {
				t.Fatalf("%s OP25 request %d selected %d", driver, requested, got)
			}
		}
		for _, rate := range []int{2_000_000, 5_000_000, 7_999_999} {
			_, err := BuildCaptureCommand(device, CaptureSpec{CenterFrequencyHz: 770_000_000, SampleRateHz: rate})
			if err == nil || !strings.Contains(err.Error(), "between 8 and 20") {
				t.Fatalf("%s rate %d bypassed floor: %v", driver, rate, err)
			}
		}
	}
	// A mixed profile's low shared rate must still be applied independently.
	profile := ScanProfile{Settings: SurveySettings{P25SampleRateHz: 2_400_000}}
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "RTL-SDR"}); got != 2_400_000 {
		t.Fatalf("RTL rate changed: %d", got)
	}
	if got := effectiveOP25DeviceRate(profile, SDRDevice{Kind: "PlutoSDR"}); got != 2_400_000 {
		t.Fatalf("Pluto rate changed: %d", got)
	}
}
