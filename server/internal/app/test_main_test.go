package app

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Unit/HTTP tests must not reopen attached radios as a side effect of
	// creating runtimes. Physical tests always require explicit opt-in.
	if os.Getenv("GPSDR_HARDWARE_TEST") != "1" && os.Getenv("GP_SDR_HARDWARE_TEST") != "1" && os.Getenv("GPSDR_CAPTURE_TEST") != "1" {
		discoverPhysicalDevices = func() []SDRDevice { return nil }
	}
	os.Exit(m.Run())
}
