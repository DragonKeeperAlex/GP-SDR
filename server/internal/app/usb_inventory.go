package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func rtlDevicesFromUSBInventory(output, driver string) []SDRDevice {
	limit := 3.2e6
	var devices []SDRDevice
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 || fields[0] != "RTL-SDR" {
			continue
		}
		index := len(devices)
		device := SDRDevice{ID: fmt.Sprintf("rtlsdr-%d", index), Name: fmt.Sprintf("RTL-SDR %d", index+1), Kind: "RTL-SDR", Driver: driver, Connected: true, Available: true, SampleRateLimit: &limit, HelperArchitecture: ptr(runtime.GOARCH)}
		if fields[1] != "" {
			device.Serial = ptr(fields[1])
		}
		device.TunerID = "RTL-2832 USB Bus:" + fields[2] + " Port:" + fields[3]
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return []SDRDevice{{ID: "rtlsdr-driver", Name: "RTL-SDR", Kind: "RTL-SDR", Driver: driver, Available: true, SampleRateLimit: &limit, Note: ptr("Driver ready; no RTL-SDR is currently detected.")}}
	}
	return devices
}

// Prevent a headless P25 process from opening radios reserved for other jobs.
// Only the private GP-SDR configuration is changed, never standalone SDRTrunk.
func restrictP25Tuners(root string, assigned []p25AssignedDevice, devices []SDRDevice) error {
	path := filepath.Join(root, "configuration", "tuner_configuration.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var config map[string]any
	if err = json.Unmarshal(data, &config); err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, item := range assigned {
		if item.Device.TunerID != "" {
			allowed[item.Device.TunerID] = true
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	disabled := []map[string]string{}
	for _, device := range devices {
		if device.TunerID == "" || allowed[device.TunerID] {
			continue
		}
		kind := "HACKRF"
		if device.Kind == "RTL-SDR" {
			kind = "RTL2832"
		}
		disabled = append(disabled, map[string]string{"tunerClass": kind, "id": device.TunerID})
	}
	config["disabledTuners"] = disabled
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Match by the physical serial, never the order of historical tuner settings.
func applyUSBIdentities(devices []SDRDevice) {
	helper, err := findTool("gpsdr-usb")
	if err != nil {
		return
	}
	output, err := runTool(helper, nil, 3*time.Second)
	if err != nil {
		return
	}
	applyUSBInventory(devices, output)
}

func applyUSBInventory(devices []SDRDevice, output string) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 || fields[1] == "" {
			continue
		}
		for i := range devices {
			d := &devices[i]
			if d.Kind != fields[0] || d.Serial == nil || !strings.EqualFold(*d.Serial, fields[1]) {
				continue
			}
			name := "HackRF"
			if d.Kind == "RTL-SDR" {
				name = "RTL-2832"
			}
			d.TunerID = name + " USB Bus:" + fields[2] + " Port:" + fields[3]
		}
	}
}
