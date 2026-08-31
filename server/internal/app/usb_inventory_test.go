package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRTLInventoryDoesNotInventDisconnectedReceiver(t *testing.T) {
	devices := rtlDevicesFromUSBInventory("HackRF\tserial\t2\t1.1\n", "rtl_test")
	if len(devices) != 1 || devices[0].Connected {
		t.Fatalf("absent RTL marked connected: %+v", devices)
	}
	devices = rtlDevicesFromUSBInventory("RTL-SDR\t00000001\t2\t1.3\nRTL-SDR\t00000002\t2\t1.4\n", "rtl_test")
	if len(devices) != 2 || devices[0].ID == devices[1].ID || devices[0].TunerID == devices[1].TunerID || *devices[1].Serial != "00000002" {
		t.Fatalf("distinct RTL identities lost: %+v", devices)
	}
}

func TestUSBIdentityMatchesSerialAndRestrictsP25Ownership(t *testing.T) {
	serialA, serialB := "aaaaaaaa", "bbbbbbbb"
	devices := []SDRDevice{{ID: "a", Kind: "HackRF", Serial: &serialA}, {ID: "b", Kind: "HackRF", Serial: &serialB}}
	applyUSBInventory(devices, "HackRF\tbbbbbbbb\t2\t1.2\nHackRF\taaaaaaaa\t1\t3.1\n")
	if devices[0].TunerID != "HackRF USB Bus:1 Port:3.1" || devices[1].TunerID != "HackRF USB Bus:2 Port:1.2" {
		t.Fatalf("wrong physical matching: %+v", devices)
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "configuration"), 0700)
	path := filepath.Join(root, "configuration", "tuner_configuration.json")
	os.WriteFile(path, []byte(`{"tunerConfigurations":[],"otherSetting":true}`), 0600)
	assigned := []p25AssignedDevice{{Device: devices[1], Role: "control"}}
	if preferredSDRTrunkTuner(assigned, root) != devices[1].TunerID {
		t.Fatal("selected serial did not map to current USB port")
	}
	if err := restrictP25Tuners(root, assigned, devices); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var config struct {
		Disabled []struct {
			ID string `json:"id"`
		} `json:"disabledTuners"`
		Other bool `json:"otherSetting"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Disabled) != 1 || config.Disabled[0].ID != devices[0].TunerID || !config.Other {
		t.Fatalf("wrong ownership/preservation: %s", data)
	}
}
