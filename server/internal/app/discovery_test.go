package app

import (
	"strings"
	"testing"
)

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

func TestUniquePhysicalDevicesMatchesNativeAndSoapyHackRFByOrdinal(t *testing.T) {
	invalid, zero := "��", "00000000000000000000000000000000"
	devices := []SDRDevice{
		{ID: "hackrf-0", Kind: "HackRF", Serial: &invalid, Driver: "/opt/homebrew/bin/hackrf_info", Connected: true},
		{ID: "soapy-hackrf-0", Kind: "HackRF", Serial: &zero, Driver: "SoapySDR:hackrf", Connected: true},
	}
	unique := uniquePhysicalDevices(devices)
	if len(unique) != 1 || unique[0].ID != "hackrf-0" {
		t.Fatalf("same HackRF was exposed twice: %#v", unique)
	}
}

func TestValidHackRFSerialRejectsTransientProbeGarbage(t *testing.T) {
	if got := validHackRFSerial("��"); got != "" {
		t.Fatalf("accepted invalid serial %q", got)
	}
	if got := validHackRFSerial("00000000000000000000000000000000"); got != "" {
		t.Fatalf("accepted placeholder serial %q", got)
	}
	const serial = "000000000000000024B862DC3140C5CB"
	if got := validHackRFSerial(serial); got != strings.ToLower(serial) {
		t.Fatalf("rejected valid serial %q", got)
	}
}

func TestParseHackRFInfoOutputKeepsSelfTestPerDevice(t *testing.T) {
	output := `hackrf_info version: 2026.01.3
Found HackRF
Index: 0
Serial number: 000000000000000024b862dc3140c5cb
Board ID Number: 4 (HackRF One)
Found HackRF
Index: 1
Serial number: 0000000000000000922c63dc217ea447
Board ID Number: 4 (HackRF One)
Self-test FAIL:
Mixer: RFFC5072, ID: 4544, Rev: 2, Locks: 111 (PASS)
`
	probes := parseHackRFInfoOutput(output)
	if len(probes) != 2 {
		t.Fatalf("expected two HackRF probes, got %#v", probes)
	}
	if probes[0].Serial == probes[1].Serial || probes[0].SelfTestFailed || !probes[1].SelfTestFailed {
		t.Fatalf("self-test result was not kept with the correct radio: %#v", probes)
	}
	device := hackRFDevice(probes[1], 1, "hackrf_info", 20e6)
	if !device.Connected || !device.Available || device.HealthWarning != "" || !device.FirmwareSelfTestWarning {
		t.Fatalf("firmware diagnostic should be hidden from receive status but retained internally: %+v", device)
	}
}
