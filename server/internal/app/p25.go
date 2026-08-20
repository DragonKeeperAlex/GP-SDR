package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type P25Status struct {
	State      string  `json:"state"`
	Engine     string  `json:"engine"`
	Executable *string `json:"executable"`
	ProfileID  *string `json:"profileID"`
	ConfigPath *string `json:"configPath"`
	APIURL     *string `json:"apiURL,omitempty"`
	Note       string  `json:"note"`
	Reception  string  `json:"reception,omitempty"`
}

type OP25Manager struct {
	mu           sync.Mutex
	restartMu    sync.Mutex
	command      *exec.Cmd
	done         chan struct{}
	waitError    error
	log          io.Closer
	profileID    *string
	configPath   *string
	lastError    *string
	engine       string
	apiURL       *string
	streamStop   chan struct{}
	audioFeeds   map[uint32]struct{}
	profile      *ScanProfile
	plan         []ReceiverPlanItem
	devices      []SDRDevice
	dataRoot     string
	muted        map[uint32]bool
	restartTimer *time.Timer
}

type op25Configuration struct {
	Channels []op25Channel  `json:"channels"`
	Devices  []op25Device   `json:"devices"`
	Trunking op25Trunking   `json:"trunking"`
	Audio    op25Audio      `json:"audio"`
	Terminal map[string]any `json:"terminal"`
	Metadata map[string]any `json:"metadata"`
}

type op25Channel struct {
	Name            string  `json:"name"`
	Device          string  `json:"device"`
	TrunkingSystem  string  `json:"trunking_sysname"`
	DemodType       string  `json:"demod_type"`
	Destination     string  `json:"destination"`
	ExcessBandwidth float64 `json:"excess_bw"`
	FilterType      string  `json:"filter_type"`
	IFRate          int     `json:"if_rate"`
	SymbolRate      int     `json:"symbol_rate"`
	EnableAnalog    string  `json:"enable_analog"`
	Whitelist       string  `json:"whitelist"`
	Blacklist       string  `json:"blacklist"`
	MetaStreamName  string  `json:"meta_stream_name"`
}

type op25Device struct {
	Arguments     string  `json:"args"`
	Gains         string  `json:"gains"`
	Name          string  `json:"name"`
	Offset        int     `json:"offset"`
	PPM           float64 `json:"ppm"`
	Rate          int     `json:"rate"`
	UsablePercent float64 `json:"usable_bw_pct"`
	Tunable       bool    `json:"tunable"`
}

type op25Trunking struct {
	Module   string            `json:"module"`
	Channels []op25TrunkSystem `json:"chans"`
}

type op25TrunkSystem struct {
	NAC             string `json:"nac"`
	WACN            string `json:"wacn"`
	SystemID        string `json:"sysid"`
	Name            string `json:"sysname"`
	ControlChannels string `json:"control_channel_list"`
	Whitelist       string `json:"whitelist"`
	Blacklist       string `json:"blacklist"`
	TalkgroupTags   string `json:"tgid_tags_file"`
	TDMAControl     bool   `json:"tdma_cc"`
	CryptBehavior   int    `json:"crypt_behavior"`
}

type op25Audio struct {
	Module    string              `json:"module"`
	Instances []op25AudioInstance `json:"instances"`
}

type op25AudioInstance struct {
	Name       string  `json:"instance_name"`
	DeviceName string  `json:"device_name"`
	UDPPort    int     `json:"udp_port"`
	Gain       float64 `json:"audio_gain"`
	Channels   int     `json:"number_channels"`
}

func findOP25() (string, error) {
	return findTool("multi_rx.py")
}

func enabledP25Systems(profile ScanProfile) []P25SystemConfig {
	systems := make([]P25SystemConfig, 0, len(profile.P25Systems))
	for _, system := range profile.P25Systems {
		if system.Enabled {
			systems = append(systems, system)
		}
	}
	return systems
}

func assignedDevices(plan []ReceiverPlanItem, devices []SDRDevice) []SDRDevice {
	byID := make(map[string]SDRDevice)
	for _, device := range devices {
		if device.Connected {
			byID[device.ID] = device
		}
	}
	items := make([]SDRDevice, 0)
	seen := make(map[string]bool)
	for _, assignment := range plan {
		if assignment.DeviceID == nil || seen[*assignment.DeviceID] {
			continue
		}
		if device, ok := byID[*assignment.DeviceID]; ok {
			items = append(items, device)
			seen[device.ID] = true
		}
	}
	return items
}

func BuildOP25Configuration(profile ScanProfile, devices []SDRDevice, directory string) ([]byte, error) {
	systems := enabledP25Systems(profile)
	if len(systems) == 0 {
		return nil, errors.New("profile has no enabled P25 system")
	}
	if len(devices) == 0 {
		return nil, errors.New("P25 trunk following needs at least one assigned SDR")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	configuration := op25Configuration{
		Trunking: op25Trunking{Module: "tk_p25.py"},
		Audio:    op25Audio{Module: "sockaudio.py"},
		Terminal: map[string]any{"module": "terminal.py", "terminal_type": "http:127.0.0.1:8081", "http_plot_interval": 1.0, "curses_plot_interval": 0.1},
		Metadata: map[string]any{"module": "icemeta.py", "streams": []any{}},
	}
	audioDevice := firstEnvironment("GPSDR_OP25_AUDIO_DEVICE")
	if audioDevice == "" {
		audioDevice = "default"
	}
	for index, device := range devices {
		name := fmt.Sprintf("sdr%d", index)
		configuration.Devices = append(configuration.Devices, op25Device{Arguments: op25DeviceArguments(device), Gains: op25Gains(device),
			Name: name, Rate: 1_000_000, UsablePercent: .85, Tunable: true})
		system := systems[index%len(systems)]
		whitelist, tags, err := writeTalkgroupFiles(directory, system)
		if err != nil {
			return nil, err
		}
		port := 23456 + index*10
		configuration.Channels = append(configuration.Channels, op25Channel{Name: fmt.Sprintf("Voice %d", index+1), Device: name,
			TrunkingSystem: system.Name, DemodType: "cqpsk", Destination: fmt.Sprintf("udp://127.0.0.1:%d", port),
			ExcessBandwidth: .2, FilterType: "rc", IFRate: 24_000, SymbolRate: 4_800, EnableAnalog: "off",
			Whitelist: whitelist, MetaStreamName: fmt.Sprintf("stream_%d", index)})
		configuration.Audio.Instances = append(configuration.Audio.Instances, op25AudioInstance{Name: fmt.Sprintf("audio%d", index),
			DeviceName: audioDevice, UDPPort: port, Gain: 1, Channels: 1})
		if !hasOP25System(configuration.Trunking.Channels, system.Name) {
			configuration.Trunking.Channels = append(configuration.Trunking.Channels, op25TrunkSystem{
				NAC: normalizedHex(system.NAC, "0x0"), WACN: normalizedHex(system.WACN, "0x00000"),
				SystemID: normalizedHex(system.SystemID, "0x000"), Name: system.Name,
				ControlChannels: frequencyListMHz(system.ControlChannelsHz), Whitelist: whitelist, TalkgroupTags: tags,
				TDMAControl: system.TDMAControl, CryptBehavior: 2,
			})
		}
	}
	return json.MarshalIndent(configuration, "", "  ")
}

func (m *OP25Manager) startOP25(profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	m.stopProcess()
	executable, err := findOP25()
	if err != nil {
		return errors.New("OP25 multi_rx.py is not installed")
	}
	runtimeDirectory := filepath.Join(dataDirectory, "Runtime", "OP25", profile.ID)
	configuration, err := BuildOP25Configuration(profile, assignedDevices(plan, devices), runtimeDirectory)
	if err != nil {
		return err
	}
	configPath := filepath.Join(runtimeDirectory, "gpsdr-op25.json")
	if err := os.WriteFile(configPath, configuration, 0o600); err != nil {
		return err
	}
	logPath := filepath.Join(runtimeDirectory, "op25.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command := exec.Command(executable, "-c", configPath, "-v", "2")
	command.Dir = filepath.Dir(executable)
	command.Stdout, command.Stderr = logFile, logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	done := make(chan struct{})
	go func() {
		waitError := command.Wait()
		m.mu.Lock()
		if m.command == command {
			m.waitError = waitError
		}
		m.mu.Unlock()
		close(done)
	}()
	id := profile.ID
	m.mu.Lock()
	m.command, m.done, m.log = command, done, logFile
	m.profileID, m.configPath, m.lastError, m.waitError = &id, &configPath, nil, nil
	m.engine, m.apiURL = "OP25", nil
	m.mu.Unlock()
	return nil
}

func (m *OP25Manager) stopProcess() {
	m.mu.Lock()
	command, done, logFile := m.command, m.done, m.log
	m.command, m.done, m.log, m.profileID, m.configPath = nil, nil, nil, nil, nil
	m.mu.Unlock()
	if command == nil || command.Process == nil {
		return
	}
	_ = command.Process.Signal(interruptSignal())
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = command.Process.Kill()
		<-done
	}
	if logFile != nil {
		_ = logFile.Close()
	}
}

func (m *OP25Manager) op25Status() P25Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.command != nil {
		select {
		case <-m.done:
			note := "OP25 stopped."
			if m.waitError != nil {
				note = m.waitError.Error()
			}
			return P25Status{State: "error", Engine: "OP25", Executable: ptr(m.command.Path), ProfileID: m.profileID, ConfigPath: m.configPath, Note: note}
		default:
		}
		return P25Status{State: "running", Engine: "OP25", Executable: ptr(m.command.Path), ProfileID: m.profileID, ConfigPath: m.configPath,
			Note: "OP25 is following trunk grants; encrypted voice is silenced."}
	}
	if executable, err := findOP25(); err == nil {
		note := "OP25 is ready."
		if m.lastError != nil {
			note = *m.lastError
		}
		return P25Status{State: "ready", Engine: "OP25", Executable: &executable, Note: note}
	}
	return P25Status{State: "setup", Engine: "none", Note: "The bundled P25 receiver is missing from this package."}
}

func op25DeviceArguments(device SDRDevice) string {
	if strings.HasPrefix(device.Driver, "SoapySDR:") {
		return "soapy=" + soapyDeviceArguments(device)
	}
	switch device.Kind {
	case "RTL-SDR":
		pieces := strings.Split(device.ID, "-")
		index := pieces[len(pieces)-1]
		if _, err := strconv.Atoi(index); err != nil {
			index = "0"
		}
		return "rtl=" + index
	case "HackRF":
		if device.Serial != nil && *device.Serial != "" {
			return "hackrf=" + *device.Serial
		}
		return "hackrf"
	default:
		return strings.ToLower(device.Kind)
	}
}

func op25Gains(device SDRDevice) string {
	if device.Kind == "HackRF" {
		return "LNA:24,VGA:24"
	}
	return "LNA:36"
}

func normalizedHex(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}
	if !strings.HasPrefix(value, "0x") {
		value = "0x" + value
	}
	return value
}

func frequencyListMHz(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatFloat(value/1e6, 'f', 6, 64))
	}
	return strings.Join(parts, ",")
}

func hasOP25System(systems []op25TrunkSystem, name string) bool {
	for _, system := range systems {
		if system.Name == name {
			return true
		}
	}
	return false
}

func writeTalkgroupFiles(directory string, system P25SystemConfig) (string, string, error) {
	safeName := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, system.Name)
	whitelistName := safeName + ".wlist"
	tagsName := safeName + "-tags.tsv"
	var whitelist, tags strings.Builder
	for _, talkgroup := range system.Talkgroups {
		if talkgroup.Encrypted || !talkgroup.Enabled {
			continue
		}
		fmt.Fprintf(&whitelist, "%d\n", talkgroup.ID)
		fmt.Fprintf(&tags, "%d\t%s\n", talkgroup.ID, strings.ReplaceAll(talkgroup.Name, "\t", " "))
	}
	if err := os.WriteFile(filepath.Join(directory, whitelistName), []byte(whitelist.String()), 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(directory, tagsName), []byte(tags.String()), 0o600); err != nil {
		return "", "", err
	}
	return whitelistName, tagsName, nil
}
