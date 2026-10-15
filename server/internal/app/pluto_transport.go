package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type PlutoTransport struct {
	URI       string `json:"uri"`
	Label     string `json:"label"`
	Interface string `json:"interface,omitempty"`
	LinkMbps  int    `json:"linkMbps,omitempty"`
}

var iioContextLine = regexp.MustCompile(`(?m)^\s*\d+:\s*(.*?)serial=([^\s,]+).*?\[([^\]]+)\]`)
var iioIPv4 = regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)\s`)

func parsePlutoContexts(output string) map[string][]PlutoTransport {
	result := make(map[string][]PlutoTransport)
	for _, m := range iioContextLine.FindAllStringSubmatch(output, -1) {
		uri := m[3]
		if strings.HasPrefix(uri, "ip:") {
			if ip := iioIPv4.FindStringSubmatch(m[1]); len(ip) > 1 {
				uri = "ip:" + ip[1]
			}
		}
		if !strings.HasPrefix(uri, "usb:") && !strings.HasPrefix(uri, "ip:") {
			continue
		}
		duplicate := false
		for _, old := range result[m[2]] {
			if old.URI == uri {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		label := "Network · " + strings.TrimPrefix(uri, "ip:")
		if strings.HasPrefix(uri, "usb:") {
			label = "USB · " + uri
		}
		result[m[2]] = append(result[m[2]], PlutoTransport{URI: uri, Label: label})
	}
	return result
}

func annotatePlutoRoute(transport *PlutoTransport) {
	if !strings.HasPrefix(transport.URI, "ip:") {
		return
	}
	host := strings.TrimPrefix(transport.URI, "ip:")
	if net.ParseIP(host) == nil {
		return
	}
	tool, err := findTool("ip")
	if err != nil {
		return
	}
	output, err := runTool(tool, []string{"-j", "route", "get", host}, time.Second)
	if err != nil {
		return
	}
	var routes []struct {
		Dev string `json:"dev"`
	}
	if json.Unmarshal([]byte(output), &routes) != nil || len(routes) == 0 {
		return
	}
	iface := routes[0].Dev
	if filepath.Base(iface) != iface {
		return
	}
	transport.Interface = iface
	devicePath, _ := filepath.EvalSymlinks(filepath.Join("/sys/class/net", iface, "device"))
	label := "Ethernet"
	if strings.Contains(devicePath, "/usb") {
		label = "USB network"
	}
	speed, _ := os.ReadFile(filepath.Join("/sys/class/net", iface, "speed"))
	var mbps int
	fmt.Sscanf(string(speed), "%d", &mbps)
	if mbps > 0 {
		transport.LinkMbps = mbps
	}
	transport.Label = fmt.Sprintf("%s · %s · %s", label, host, iface)
	if mbps > 0 {
		transport.Label += fmt.Sprintf(" · %d Mb/s link", mbps)
	}
}

func attachPlutoTransportChoices(devices []SDRDevice) {
	hasPluto := false
	for _, d := range devices {
		if d.Kind == "PlutoSDR" && d.Connected {
			hasPluto = true
		}
	}
	if !hasPluto {
		return
	}
	tool, err := findTool("iio_info")
	if err != nil {
		return
	}
	output, _ := runTool(tool, []string{"-s"}, 12*time.Second)
	contexts := parsePlutoContexts(output)
	for i := range devices {
		d := &devices[i]
		if d.Kind != "PlutoSDR" || d.Serial == nil {
			continue
		}
		d.Transports = contexts[*d.Serial]
		for j := range d.Transports {
			annotatePlutoRoute(&d.Transports[j])
		}
	}
}

func loadPlutoPreferences(directory string) (map[string]string, error) {
	preferences := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(directory, "Data", "pluto-transports.json"))
	if errors.Is(err, os.ErrNotExist) {
		return preferences, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &preferences); err != nil {
		return nil, err
	}
	return preferences, nil
}

func applyPlutoPreference(d *SDRDevice, uri string) {
	if uri == "" {
		return
	}
	for _, option := range d.Transports {
		if option.URI == uri || strings.HasPrefix(uri, "usb:") && strings.HasPrefix(option.URI, "usb:") {
			uri = option.URI
			d.Transport = uri
			d.DeviceArguments = "driver=plutosdr,uri=" + uri
			return
		}
	}
	// Retain the requested route and fail visibly; never silently use USB.
	d.Transport = uri
	d.DeviceArguments = "driver=plutosdr,uri=" + uri
	d.Connected = false
	d.Note = ptr("Saved Pluto transport is unavailable. Reconnect it or select another connection in Hardware.")
}

func (r *Runtime) SetPlutoTransport(id, uri string) (SDRDevice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running || len(r.mapperJobs) > 0 || r.characterization != nil && r.characterization.Status().Running {
		return SDRDevice{}, errors.New("stop receiver work before changing Pluto connection")
	}
	index := -1
	for i, d := range r.devices {
		if d.ID == id && d.Kind == "PlutoSDR" {
			index = i
			break
		}
	}
	if index < 0 {
		return SDRDevice{}, errors.New("Pluto receiver not found")
	}
	device := r.devices[index]
	found := false
	for _, option := range device.Transports {
		if option.URI == uri {
			found = true
		}
	}
	if !found {
		return SDRDevice{}, errors.New("choose a discovered connection for this Pluto board")
	}
	tool, err := findTool("SoapySDRUtil")
	if err != nil {
		return SDRDevice{}, err
	}
	probe, err := runTool(tool, []string{"--probe=driver=plutosdr,uri=" + uri}, 15*time.Second)
	if err != nil || !strings.Contains(probe, "Device identification") {
		return SDRDevice{}, errors.New("selected Pluto connection did not respond")
	}
	serial := firstValue("hw_serial=", probe)
	if device.Serial == nil || serial != *device.Serial {
		return SDRDevice{}, errors.New("selected connection did not identify the expected Pluto board")
	}
	preferences, err := loadPlutoPreferences(r.dataDirectory)
	if err != nil {
		return SDRDevice{}, err
	}
	preferences[id] = uri
	data, err := json.Marshal(preferences)
	if err != nil {
		return SDRDevice{}, err
	}
	if err = writeBytesAtomic(filepath.Join(r.dataDirectory, "Data", "pluto-transports.json"), data); err != nil {
		return SDRDevice{}, err
	}
	device.Connected = true
	device.Note = nil
	applyPlutoPreference(&device, uri)
	r.devices[index] = device
	return device, nil
}
