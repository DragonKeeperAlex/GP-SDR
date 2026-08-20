package app

import "testing"

func TestUniquePhysicalDevicesPrefersNativeDiscovery(t *testing.T) {
	serial := "abc123"
	devices := []SDRDevice{
		{ID: "hackrf-abc123", Kind: "HackRF", Serial: &serial, Driver: "/opt/homebrew/bin/hackrf_info"},
		{ID: "soapy-hackrf-abc123", Kind: "HackRF", Serial: &serial, Driver: "SoapySDR:hackrf"},
	}
	unique := uniquePhysicalDevices(devices)
	if len(unique) != 1 || unique[0].ID != "hackrf-abc123" {
		t.Fatalf("expected native hardware entry, got %#v", unique)
	}
}
