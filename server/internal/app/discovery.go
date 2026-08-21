package app

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func DiscoverDevices(includeSimulator bool) []SDRDevice {
	devices := make([]SDRDevice, 0)
	if includeSimulator {
		limit := 20e6
		devices = append(devices, SDRDevice{ID: "simulator-0", Name: "Demo Receiver", Kind: "Simulator",
			Driver: "built-in", Connected: true, Available: true, SampleRateLimit: &limit,
			Note: ptr("Generates clearly marked sample activity for interface testing.")})
	}
	devices = append(devices, discoverHackRF()...)
	devices = append(devices, discoverRTLSDR()...)
	devices = append(devices, discoverSoapy()...)
	return uniquePhysicalDevices(devices)
}

func uniquePhysicalDevices(devices []SDRDevice) []SDRDevice {
	nativeConnected := make(map[string]int)
	for _, device := range devices {
		if device.Connected && !strings.HasPrefix(device.Driver, "SoapySDR:") {
			nativeConnected[strings.ToLower(device.Kind)]++
		}
	}
	soapySeen := make(map[string]int)
	seen := make(map[string]bool)
	unique := devices[:0]
	for _, device := range devices {
		kind := strings.ToLower(device.Kind)
		if strings.HasPrefix(device.Driver, "SoapySDR:") {
			ordinal := soapySeen[kind]
			soapySeen[kind]++
			if ordinal < nativeConnected[kind] {
				continue
			}
		}
		key := device.ID
		if device.Serial != nil && strings.TrimSpace(*device.Serial) != "" {
			key = strings.ToLower(device.Kind) + ":" + strings.ToLower(strings.TrimSpace(*device.Serial))
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, device)
		}
	}
	return unique
}

func discoverHackRF() []SDRDevice {
	tool, err := findTool("hackrf_info")
	limit := 20e6
	if err != nil {
		return []SDRDevice{{ID: "hackrf-driver", Name: "HackRF", Kind: "HackRF", Driver: "libhackrf", Available: false, SampleRateLimit: &limit, Note: ptr("Install HackRF host tools to enable this driver.")}}
	}
	output, _ := runTool(tool, nil, 4*time.Second)
	serials := valuesAfter("Serial number:", output)
	if len(serials) == 0 {
		serials = valuesAfter("Serial No:", output)
	}
	if len(serials) == 0 {
		return []SDRDevice{{ID: "hackrf-driver", Name: "HackRF", Kind: "HackRF", Driver: tool, Available: true, Connected: false, SampleRateLimit: &limit, HelperArchitecture: ptr(runtime.GOARCH), Note: ptr("Driver ready; no HackRF is currently detected.")}}
	}
	items := make([]SDRDevice, 0, len(serials))
	for index, serial := range serials {
		serial = validHackRFSerial(serial)
		s := serial
		id := fmt.Sprintf("hackrf-%d", index)
		var serialPointer *string
		if serial != "" {
			id, serialPointer = "hackrf-"+serial, &s
		}
		items = append(items, SDRDevice{ID: id, Name: "HackRF One", Kind: "HackRF", Serial: serialPointer, Driver: tool, Connected: true, Available: true, SampleRateLimit: &limit, HelperArchitecture: ptr(runtime.GOARCH)})
	}
	return items
}

func discoverRTLSDR() []SDRDevice {
	tool, err := findTool("rtl_test")
	limit := 3.2e6
	if err != nil {
		return []SDRDevice{{ID: "rtlsdr-driver", Name: "RTL-SDR", Kind: "RTL-SDR", Driver: "librtlsdr", Available: false, SampleRateLimit: &limit, Note: ptr("Install RTL-SDR host tools to enable this driver.")}}
	}
	output, _ := runTool(tool, []string{"-t"}, 4*time.Second)
	re := regexp.MustCompile(`(?i)found\s+(\d+)\s+device`)
	count := 0
	if match := re.FindStringSubmatch(output); len(match) > 1 {
		count, _ = strconv.Atoi(match[1])
	}
	if count < 1 {
		return []SDRDevice{{ID: "rtlsdr-driver", Name: "RTL-SDR", Kind: "RTL-SDR", Driver: tool, Available: true, Connected: false, SampleRateLimit: &limit, HelperArchitecture: ptr(runtime.GOARCH), Note: ptr("Driver ready; no RTL-SDR is currently detected.")}}
	}
	items := make([]SDRDevice, 0, count)
	for i := 0; i < count; i++ {
		serial := rtlSDRSerial(i)
		var serialPointer *string
		if serial != "" {
			serialPointer = &serial
		}
		items = append(items, SDRDevice{ID: fmt.Sprintf("rtlsdr-%d", i), Name: fmt.Sprintf("RTL-SDR %d", i+1), Kind: "RTL-SDR", Driver: tool, Connected: true, Available: true, SampleRateLimit: &limit, HelperArchitecture: ptr(runtime.GOARCH)})
		items[len(items)-1].Serial = serialPointer
	}
	return items
}

func rtlSDRSerial(index int) string {
	tool, err := findTool("rtl_eeprom")
	if err != nil {
		return ""
	}
	output, _ := runTool(tool, []string{"-d", strconv.Itoa(index)}, 3*time.Second)
	values := valuesAfter("Serial number:", output)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func discoverSoapy() []SDRDevice {
	tool, err := findTool("SoapySDRUtil")
	if err != nil {
		return nil
	}
	output, _ := runTool(tool, []string{"--find"}, 5*time.Second)
	blocks := strings.Split(output, "Found device")
	items := make([]SDRDevice, 0)
	for index, block := range blocks[1:] {
		driver := firstValue("driver =", block)
		if driver == "" {
			driver = "soapy"
		}
		label := firstValue("label =", block)
		if label == "" {
			label = "SoapySDR " + driver
		}
		serial := firstValue("serial =", block)
		if strings.EqualFold(driver, "hackrf") {
			serial = validHackRFSerial(serial)
		}
		id := fmt.Sprintf("soapy-%s-%d", driver, index)
		var serialPtr *string
		if serial != "" {
			id = "soapy-" + driver + "-" + serial
			serialPtr = &serial
		}
		items = append(items, SDRDevice{ID: id, Name: label, Kind: kindForDriver(driver), Serial: serialPtr, Driver: "SoapySDR:" + driver, Connected: true, Available: true, HelperArchitecture: ptr(runtime.GOARCH)})
	}
	return items
}

func validHackRFSerial(value string) string {
	value = strings.TrimSpace(value)
	if matched, _ := regexp.MatchString(`(?i)^[0-9a-f]{32}$`, value); !matched || strings.Trim(value, "0") == "" {
		return ""
	}
	return strings.ToLower(value)
}

func DiscoverDecoders() []DecoderDescriptor {
	items := []DecoderDescriptor{{ID: "analog", Name: "Analog Receiver", Standards: []string{"AM", "NFM", "WFM"}, State: "ready", Note: "Built-in activity classification and demodulator foundation."}}
	definitions := []struct {
		id, name            string
		standards, commands []string
		missing             string
	}{
		{"p25", "SDRTrunk", []string{"P25 Phase 1", "P25 Phase 2", "P25 Trunking"}, []string{"sdr-trunk", "sdr-trunk.bat"}, "The SDRTrunk P25 component is included with complete GP-SDR packages."},
		{"dsd-fme", "DSD-FME", []string{"P25", "DMR", "NXDN", "D-STAR", "YSF", "M17"}, []string{"dsd-fme", "dsd"}, "Optional digital voice decoder is not installed."},
		{"rtl-433", "rtl_433", []string{"ISM Sensors", "Weather Sensors", "TPMS"}, []string{"rtl_433"}, "Install rtl_433 to decode supported sensor protocols."},
		{"dump1090", "dump1090", []string{"ADS-B", "Mode S"}, []string{"dump1090", "dump1090-fa"}, "Install dump1090 to decode ADS-B."},
		{"multimon-ng", "multimon-ng", []string{"POCSAG", "FLEX", "MDC1200", "DTMF"}, []string{"multimon-ng"}, "Install multimon-ng to enable pager and signaling decoders."},
		{"acarsdec", "acarsdec", []string{"ACARS"}, []string{"acarsdec"}, "Install acarsdec to decode ACARS."},
		{"ais", "AIS-catcher", []string{"AIS"}, []string{"AIS-catcher", "ais-catcher"}, "Install AIS-catcher to decode marine AIS."},
	}
	for _, def := range definitions {
		item := DecoderDescriptor{ID: def.id, Name: def.name, Standards: def.standards, State: "optional", Note: def.missing}
		if def.id == "p25" {
			if path, err := findSDRTrunk(); err == nil {
				item.State, item.Executable, item.Note = "ready", &path, "SDRTrunk headless P25 engine is ready."
			}
			items = append(items, item)
			continue
		}
		for _, command := range def.commands {
			if path, err := findTool(command); err == nil {
				item.State = "ready"
				item.Executable = &path
				item.Note = "Available through " + filepath.Base(path) + "."
				break
			}
		}
		items = append(items, item)
	}
	return items
}

func runTool(path string, args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return string(output), ctx.Err()
	}
	return string(output), err
}
func valuesAfter(marker, text string) []string {
	var values []string
	for _, line := range strings.Split(text, "\n") {
		if index := strings.Index(strings.ToLower(line), strings.ToLower(marker)); index >= 0 {
			value := strings.TrimSpace(line[index+len(marker):])
			if value != "" {
				values = append(values, value)
			}
		}
	}
	return values
}
func firstValue(marker, text string) string {
	values := valuesAfter(marker, text)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
func kindForDriver(driver string) string {
	value := strings.ToLower(driver)
	switch {
	case strings.Contains(value, "rtl"):
		return "RTL-SDR"
	case strings.Contains(value, "hackrf"):
		return "HackRF"
	case strings.Contains(value, "airspy"):
		return "Airspy"
	case strings.Contains(value, "sdrplay"):
		return "SDRplay"
	case strings.Contains(value, "lime"):
		return "LimeSDR"
	case strings.Contains(value, "blade"):
		return "bladeRF"
	case strings.Contains(value, "pluto"):
		return "PlutoSDR"
	case strings.Contains(value, "uhd") || strings.Contains(value, "usrp"):
		return "USRP"
	case strings.Contains(value, "remote"):
		return "Remote"
	default:
		return "Other"
	}
}
