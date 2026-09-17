package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
		device := SDRDevice{ID: fmt.Sprintf("rtlsdr-%d", index), Name: fmt.Sprintf("RTL-SDR %d", index+1), Kind: "RTL-SDR", Driver: driver, Connected: true, Available: true, SampleRateLimit: &limit, ReceiveChannels: 1, HelperArchitecture: ptr(runtime.GOARCH)}
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
		data, err = []byte("{}"), nil
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
		return fmt.Errorf("cannot establish USB identity for the assigned P25 receiver; refresh Hardware before starting")
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
	output, err := readUSBInventory()
	if err == nil {
		applyUSBInventory(devices, output)
	}
}

func readUSBInventory() (string, error) {
	if runtime.GOOS == "linux" {
		return readLinuxUSBInventory("/sys/bus/usb/devices")
	}
	helper, err := findTool("gpsdr-usb")
	if err != nil {
		return "", err
	}
	return runTool(helper, nil, 3*time.Second)
}

// Sysfs exposes physical USB identities without opening or resetting a radio.
func readLinuxUSBInventory(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		read := func(name string) string {
			data, _ := os.ReadFile(filepath.Join(path, name))
			return strings.TrimSpace(string(data))
		}
		vendor, product := read("idVendor"), read("idProduct")
		kind := ""
		if vendor == "1d50" && product == "6089" {
			kind = "HackRF"
		}
		if vendor == "0bda" && (product == "2838" || product == "2832") {
			kind = "RTL-SDR"
		}
		if kind == "" {
			continue
		}
		bus, err := strconv.Atoi(read("busnum"))
		port := read("devpath")
		if err != nil || port == "" {
			continue
		}
		serial := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(read("serial"))
		fmt.Fprintf(&output, "%s\t%s\t%d\t%s\n", kind, serial, bus, port)
	}
	return output.String(), nil
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
