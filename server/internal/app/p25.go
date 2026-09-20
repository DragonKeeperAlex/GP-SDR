package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type P25Status struct {
	State             string     `json:"state"`
	Engine            string     `json:"engine"`
	Executable        *string    `json:"executable"`
	ProfileID         *string    `json:"profileID"`
	ConfigPath        *string    `json:"configPath"`
	APIURL            *string    `json:"apiURL,omitempty"`
	Note              string     `json:"note"`
	Reception         string     `json:"reception,omitempty"`
	ControlChannelHz  float64    `json:"controlChannelHz,omitempty"`
	ControlSource     string     `json:"controlSource,omitempty"`
	CaptureRateHz     int        `json:"captureRateHz,omitempty"`
	AudioState        string     `json:"audioState,omitempty"`
	AudioFrames       uint64     `json:"audioFrames,omitempty"`
	AudioSampleRateHz int        `json:"audioSampleRateHz,omitempty"`
	AudioLastFrameAt  *time.Time `json:"audioLastFrameAt,omitempty"`
	FrontendState     string     `json:"frontendState,omitempty"`
	// ReceiverDeviceIDs is the live P25 assignment, not merely a profile
	// preference. The web client uses it to avoid rendering controls for a
	// previously selected receiver after a P25 session has started elsewhere.
	ReceiverDeviceIDs []string `json:"receiverDeviceIDs,omitempty"`
}

type OP25Manager struct {
	mu            sync.Mutex
	restartMu     sync.Mutex
	command       *exec.Cmd
	done          chan struct{}
	waitError     error
	log           io.Closer
	profileID     *string
	configPath    *string
	lastError     *string
	engine        string
	apiURL        *string
	streamStop    chan struct{}
	audioFeeds    map[uint32]struct{}
	profile       *ScanProfile
	plan          []ReceiverPlanItem
	devices       []SDRDevice
	dataRoot      string
	muted         map[uint32]bool
	restartTimer  *time.Timer
	sessionStart  time.Time
	rateFallback  bool
	audioHub      *AudioHub
	audioSockets  []io.Closer
	audioFrames   uint64
	audioSamples  uint64
	audioLastAt   time.Time
	captureRateHz int
	op25Calls     map[int]P25ActiveCall
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
		if device.Connected && device.Available {
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
	return buildOP25ConfigurationWithPlan(profile, devices, nil, directory)
}

func buildOP25ConfigurationWithPlan(profile ScanProfile, devices []SDRDevice, plan []ReceiverPlanItem, directory string) ([]byte, error) {
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
		rate := effectiveOP25DeviceRate(profile, device)
		configuration.Devices = append(configuration.Devices, op25Device{Arguments: op25DeviceArguments(device), Gains: op25Gains(device),
			Name: name, Rate: rate, UsablePercent: .85, Tunable: true})
		if device.Calibration != nil {
			configuration.Devices[len(configuration.Devices)-1].PPM = float64(device.Calibration.PPMCorrection)
		}
		if device.Kind == "HackRF" {
			configuration.Devices[len(configuration.Devices)-1].Offset = 100_000
			// Conservative default for the native HackRF path. The repeatable
			// field lock used low gain (LNA 8/VGA 0); higher gain remains an
			// explicit profile choice for weaker signals or different front ends.
			lna, vga := 8, 0
			if profile.Settings.P25LNAGainDB != nil {
				lna = *profile.Settings.P25LNAGainDB
			}
			if profile.Settings.P25VGAGainDB != nil {
				vga = *profile.Settings.P25VGAGainDB
			}
			configuration.Devices[len(configuration.Devices)-1].Gains = fmt.Sprintf("LNA:%d,VGA:%d,AMP:%d", lna, vga, func() int {
				if profile.Settings.P25AmpMode == "on" {
					return 14
				}
				return 0
			}())
		}
		system := systems[index%len(systems)]
		for _, item := range plan {
			if item.DeviceID == nil || *item.DeviceID != device.ID || item.Target == nil {
				continue
			}
			for _, candidate := range systems {
				if *item.Target == candidate.ID || *item.Target == candidate.Name {
					system = candidate
					break
				}
			}
		}
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

// effectiveOP25DeviceRate keeps OP25's per-device configuration aligned with
// the receiver's supported P25 rate. A profile can be shared between Pluto,
// RTL-SDR, and HackRF; a rate valid for one must not be sent to another.
func effectiveOP25DeviceRate(profile ScanProfile, device SDRDevice) int {
	requested := profile.Settings.P25SampleRateHz
	switch device.Kind {
	case "HackRF":
		if isHackRFSampleRate(requested) {
			return requested
		}
		// 5 MS/s leaves enough bandwidth for a P25 control/voice channel while
		// avoiding the USB and CPU pressure that made the automatic 10 MS/s
		// choice miss a known live control channel on the Pi. Higher rates remain
		// available when the user explicitly selects one.
		return 5_000_000
	case "RTL-SDR":
		if _, ok := rtlSDRSampleRateName(requested); ok {
			return requested
		}
		return 2_400_000
	default:
		if requested > 0 && (device.SampleRateLimit == nil || float64(requested) <= *device.SampleRateLimit) {
			return requested
		}
		return 1_000_000
	}
}

func (m *OP25Manager) startOP25(profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	started := false
	defer func() {
		if !started {
			m.closeOP25Audio()
		}
	}()
	m.stopProcess()
	executable, err := findOP25()
	if err != nil {
		return errors.New("OP25 multi_rx.py is not installed")
	}
	runtimeDirectory := filepath.Join(dataDirectory, "Runtime", "OP25", profile.ID)
	configuration, err := buildOP25ConfigurationWithPlan(profile, assignedDevices(plan, devices), plan, runtimeDirectory)
	if err != nil {
		return err
	}
	var config op25Configuration
	if err := json.Unmarshal(configuration, &config); err != nil {
		return err
	}
	captureRateHz := 0
	if len(config.Devices) > 0 {
		captureRateHz = config.Devices[0].Rate
	}
	if m.audioHub != nil {
		config.Audio.Instances = []op25AudioInstance{}
		configuration, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return err
		}
		if err := m.startOP25Audio(len(config.Channels)); err != nil {
			return err
		}
	}
	configPath := filepath.Join(runtimeDirectory, "gpsdr-op25.json")
	if err := os.WriteFile(configPath, configuration, 0o600); err != nil {
		return err
	}
	logPath := filepath.Join(runtimeDirectory, "op25.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command := exec.Command(executable, "-c", configPath, "-v", "2")
	// OP25 keeps talkgroup whitelist and tag paths relative to its working
	// directory. Run it beside the generated configuration so those files are
	// found without weakening the profile's portable JSON format.
	command.Dir = runtimeDirectory
	command.Stdout, command.Stderr = logFile, logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		m.stopProcess()
		return err
	}
	done := make(chan struct{})
	go func() {
		waitError := command.Wait()
		m.mu.Lock()
		if m.command == command {
			m.waitError = waitError
			if waitError != nil {
				message := op25ExitDiagnostic(logPath, waitError)
				m.lastError = &message
			}
			sockets := m.audioSockets
			m.audioSockets = nil
			for _, socket := range sockets {
				_ = socket.Close()
			}
		}
		m.mu.Unlock()
		close(done)
	}()
	id := profile.ID
	m.mu.Lock()
	m.command, m.done, m.log = command, done, logFile
	m.profileID, m.configPath, m.lastError, m.waitError = &id, &configPath, nil, nil
	m.engine, m.apiURL = "OP25", nil
	m.audioFrames, m.audioSamples, m.audioLastAt, m.captureRateHz = 0, 0, time.Time{}, captureRateHz
	m.op25Calls = nil
	m.profile, m.plan, m.devices, m.dataRoot = &profile, append([]ReceiverPlanItem(nil), plan...), append([]SDRDevice(nil), devices...), dataDirectory
	m.sessionStart = time.Now()
	m.mu.Unlock()
	started = true
	return nil
}

// op25ExitDiagnostic preserves the useful, Pi-local decoder context when a
// child exits.  A bare exit status leaves an operator unable to distinguish a
// USB loss, bad control configuration, or an OP25 startup failure from the
// P25 page.  Keep this deliberately short: status responses must stay useful
// on narrow/mobile clients and must never turn a full runtime log into UI.
func op25ExitDiagnostic(logPath string, waitError error) string {
	message := "OP25 stopped"
	if waitError != nil {
		message += ": " + waitError.Error()
	}
	text := strings.TrimSpace(tailText(logPath, 4_096))
	if text == "" {
		return message
	}
	lines := strings.Split(text, "\n")
	context := make([]string, 0, 3)
	for index := len(lines) - 1; index >= 0 && len(context) < 3; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" || strings.HasPrefix(line, "Starting OP25") {
			continue
		}
		context = append(context, line)
	}
	for left, right := 0, len(context)-1; left < right; left, right = left+1, right-1 {
		context[left], context[right] = context[right], context[left]
	}
	if len(context) == 0 {
		return message
	}
	return truncateText(message+" · "+strings.Join(context, " · "), 1_200)
}

func (m *OP25Manager) stopProcess() {
	m.mu.Lock()
	command, done, logFile := m.command, m.done, m.log
	sockets := m.audioSockets
	m.audioSockets = nil
	m.command, m.done, m.log, m.profileID, m.configPath = nil, nil, nil, nil, nil
	m.mu.Unlock()
	for _, socket := range sockets {
		_ = socket.Close()
	}
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
			if m.lastError != nil {
				note = *m.lastError
			} else if m.waitError != nil {
				note = m.waitError.Error()
			}
			return P25Status{State: "error", Engine: "OP25", Executable: ptr(m.command.Path), ProfileID: m.profileID, ConfigPath: m.configPath, Note: note}
		default:
		}
		audioState := "waiting"
		var audioLastAt *time.Time
		if !m.audioLastAt.IsZero() {
			at := m.audioLastAt
			audioLastAt = &at
			if time.Since(at) < 5*time.Second {
				audioState = "receiving"
			} else {
				audioState = "idle"
			}
		}
		status := P25Status{State: "running", Engine: "OP25", Executable: ptr(m.command.Path), ProfileID: m.profileID, ConfigPath: m.configPath, CaptureRateHz: m.captureRateHz,
			Reception: "searching", Note: "OP25 is checking the configured P25 control channels.", ReceiverDeviceIDs: p25ReceiverDeviceIDs(m.plan, m.devices)}
		status.AudioState, status.AudioFrames, status.AudioSampleRateHz, status.AudioLastFrameAt = audioState, m.audioFrames, 8000, audioLastAt
		status.FrontendState = p25FrontendState(m.profile, m.plan, m.devices)
		if frequency, ok := readOP25ControlStatus(); ok {
			status.Reception = "locked"
			status.ControlChannelHz = frequency
			status.ControlSource = "OP25 decoded control messages"
			status.Note = "P25 control channel locked; waiting for an enabled, unencrypted call."
		}
		return status
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

// p25FrontendState reports the values written to the live OP25 receiver
// config, rather than a profile form's shared/default values.
func p25FrontendState(profile *ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice) string {
	if profile == nil {
		return ""
	}
	assigned := assignedDevices(plan, devices)
	if len(assigned) == 0 {
		return ""
	}
	device := assigned[0]
	switch device.Kind {
	case "HackRF":
		lna, vga := 24, 24
		if profile.Settings.P25LNAGainDB != nil {
			lna = *profile.Settings.P25LNAGainDB
		}
		if profile.Settings.P25VGAGainDB != nil {
			vga = *profile.Settings.P25VGAGainDB
		}
		amp := "off"
		if profile.Settings.P25AmpMode == "on" {
			amp = "on"
		}
		return fmt.Sprintf("LNA %d dB · VGA %d dB · RF amp %s", lna, vga, amp)
	case "RTL-SDR":
		return "RTL LNA 36 dB"
	case "PlutoSDR":
		return "Pluto PGA 45 dB"
	default:
		return op25Gains(device)
	}
}

func p25ReceiverDeviceIDs(plan []ReceiverPlanItem, devices []SDRDevice) []string {
	assigned := p25DeviceAssignments(plan, devices)
	ids := make([]string, 0, len(assigned))
	for _, assignment := range assigned {
		ids = append(ids, assignment.Device.ID)
	}
	return ids
}

// OP25 can decode control traffic without an enabled talkgroup receiving a
// voice grant. Use its local decoded state rather than a voice-only log marker.
func readOP25ControlStatus() (float64, bool) {
	updates := readOP25Updates()
	for _, update := range updates {
		var kind string
		_ = json.Unmarshal(update["json_type"], &kind)
		if kind != "trunk_update" {
			continue
		}
		for key, raw := range update {
			if _, err := strconv.Atoi(key); err != nil {
				continue
			}
			var state struct {
				Frequency float64 `json:"rxchan"`
				Active    int     `json:"network_active"`
				TopLine   string  `json:"top_line"`
				LastTSBK  float64 `json:"last_tsbk"`
			}
			if json.Unmarshal(raw, &state) == nil && state.Active == 1 && state.Frequency > 0 && state.LastTSBK > float64(time.Now().Unix()-10) && strings.Contains(state.TopLine, "tsbks ") {
				return state.Frequency, true
			}
		}
	}
	return 0, false
}

func readOP25Updates() []map[string]json.RawMessage {
	client := &http.Client{Timeout: 300 * time.Millisecond}
	response, err := client.Post("http://127.0.0.1:8081/", "application/json", strings.NewReader(`[{"command":"update","arg1":0,"arg2":0}]`))
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	var updates []map[string]json.RawMessage
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&updates); err != nil {
		return nil
	}
	return updates
}

func readFileTail(path string, limit int64) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return ""
	}
	if info.Size() > limit {
		if _, err := file.Seek(info.Size()-limit, io.SeekStart); err != nil {
			return ""
		}
	}
	data, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return ""
	}
	return string(data)
}

func op25DeviceArguments(device SDRDevice) string {
	if strings.HasPrefix(device.Driver, "SoapySDR:") {
		return "soapy=0," + soapyDeviceArguments(device)
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
	if device.Kind == "PlutoSDR" {
		return "PGA:45"
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
