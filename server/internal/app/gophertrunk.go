package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const bundledP25Version = "0.9.9-gpsdr1"

type P25TalkgroupState struct {
	ID          uint32 `json:"id"`
	AlphaTag    string `json:"alpha_tag"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Group       string `json:"group,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Lockout     bool   `json:"lockout,omitempty"`
	Scan        bool   `json:"scan"`
	Record      bool   `json:"record"`
	Mute        bool   `json:"mute"`
	Discovered  bool   `json:"discovered"`
}

type p25GrantState struct {
	System      string `json:"system"`
	Protocol    string `json:"protocol"`
	GroupID     uint32 `json:"group_id"`
	SourceID    uint32 `json:"source_id"`
	FrequencyHz uint32 `json:"frequency_hz"`
	Encrypted   bool   `json:"encrypted"`
	Emergency   bool   `json:"emergency"`
}

type P25ActiveCall struct {
	Grant        p25GrantState      `json:"grant"`
	Talkgroup    *P25TalkgroupState `json:"talkgroup,omitempty"`
	DeviceSerial string             `json:"device_serial"`
	StartedAt    time.Time          `json:"started_at"`
	LastHeardAt  time.Time          `json:"last_heard_at"`
	Following    bool               `json:"following"`
}

type p25AssignedDevice struct {
	Device SDRDevice
	Role   string
}

func findGopherTrunk() (string, error) {
	return findTool("gophertrunk")
}

// Start selects the bundled, dependency-free P25 stack first. OP25 remains a
// compatible fallback for source checkouts made before GP-SDR began bundling
// GopherTrunk, but packaged releases never require the user to install it.
func (m *OP25Manager) Start(profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	m.Stop()
	if executable, err := findGopherTrunk(); err == nil {
		return m.startGopherTrunk(executable, profile, plan, devices, dataDirectory)
	}
	if _, err := findOP25(); err == nil {
		return m.startOP25(profile, plan, devices, dataDirectory)
	}
	return errors.New("the GP-SDR P25 engine is missing; reinstall the complete GP-SDR package")
}

func (m *OP25Manager) Stop() {
	m.mu.Lock()
	streamStop := m.streamStop
	m.streamStop = nil
	m.audioFeeds = nil
	m.mu.Unlock()
	if streamStop != nil {
		close(streamStop)
	}
	m.stopProcess()
	m.mu.Lock()
	m.engine, m.apiURL = "", nil
	m.mu.Unlock()
}

func (m *OP25Manager) Status() P25Status {
	m.mu.Lock()
	engine := m.engine
	command, done := m.command, m.done
	waitError := m.waitError
	profileID, configPath, apiURL := m.profileID, m.configPath, m.apiURL
	m.mu.Unlock()
	if engine == "GopherTrunk" && command != nil {
		select {
		case <-done:
			note := "The bundled P25 engine stopped."
			if waitError != nil {
				note = waitError.Error()
			}
			return P25Status{State: "error", Engine: engine, Executable: ptr(command.Path), ProfileID: profileID, ConfigPath: configPath, APIURL: apiURL, Note: note}
		default:
			status := P25Status{State: "running", Engine: engine, Executable: ptr(command.Path), ProfileID: profileID, ConfigPath: configPath, APIURL: apiURL,
				Reception: "searching", Note: "P25 engine is running and searching for a control channel."}
			if configPath != nil {
				if data, err := os.ReadFile(filepath.Join(filepath.Dir(*configPath), "p25-engine.log")); err == nil {
					if len(data) > 256_000 {
						data = data[len(data)-256_000:]
					}
					logText := string(data)
					switch {
					case strings.Contains(logText, "wideband front end overloaded"):
						status.Reception = "overloaded"
						status.Note = "Receiver input is clipping; reduce gain or add attenuation. P25 control lock is not confirmed."
					case strings.Contains(logText, "control channel locked"), strings.Contains(logText, "control decode activity"):
						status.Reception = "locked"
						status.Note = "P25 control channel locked; trunk following and voice decoding are active."
					}
				}
			}
			return status
		}
	}
	if engine == "OP25" {
		return m.op25Status()
	}
	if executable, err := findGopherTrunk(); err == nil {
		return P25Status{State: "ready", Engine: "GopherTrunk", Executable: &executable,
			Note: "Bundled P25 Phase 1/2 trunking and voice stack is ready."}
	}
	if _, err := findOP25(); err == nil {
		return m.op25Status()
	}
	return P25Status{State: "setup", Engine: "none", Note: "The complete P25 engine is not present in this package."}
}

func (m *OP25Manager) startGopherTrunk(executable string, profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	assigned := p25DeviceAssignments(plan, devices)
	if len(assigned) == 0 {
		return errors.New("P25 trunk following needs at least one assigned SDR")
	}
	runtimeDirectory := filepath.Join(dataDirectory, "Runtime", "P25", profile.ID)
	if err := os.MkdirAll(runtimeDirectory, 0o700); err != nil {
		return err
	}
	httpPort, err := freeTCPPort()
	if err != nil {
		return fmt.Errorf("reserve P25 API port: %w", err)
	}
	grpcPort, err := freeTCPPort()
	if err != nil {
		return fmt.Errorf("reserve P25 control port: %w", err)
	}
	configuration, err := BuildGopherTrunkConfiguration(profile, assigned, runtimeDirectory, httpPort, grpcPort)
	if err != nil {
		return err
	}
	configPath := filepath.Join(runtimeDirectory, "gp-sdr-p25.yaml")
	if err := os.WriteFile(configPath, configuration, 0o600); err != nil {
		return err
	}
	logPath := filepath.Join(runtimeDirectory, "p25-engine.log")
	// Status diagnostics must describe this receiver session. Appending leaves
	// stale overload warnings behind after hardware or compatibility fixes.
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command := exec.Command(executable, "run", "-config", configPath, "-headless")
	command.Dir = runtimeDirectory
	command.Stdout, command.Stderr = logFile, logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start bundled P25 engine: %w", err)
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
	apiURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	id := profile.ID
	streamStop := make(chan struct{})
	m.mu.Lock()
	m.command, m.done, m.log = command, done, logFile
	m.profileID, m.configPath, m.lastError, m.waitError = &id, &configPath, nil, nil
	m.engine, m.apiURL, m.streamStop = "GopherTrunk", &apiURL, streamStop
	m.audioFeeds = make(map[uint32]struct{})
	m.mu.Unlock()
	if err := waitForP25Health(apiURL, done, 12*time.Second); err != nil {
		m.Stop()
		return fmt.Errorf("bundled P25 engine did not become ready: %w (log: %s)", err, logPath)
	}
	return nil
}

func p25DeviceAssignments(plan []ReceiverPlanItem, devices []SDRDevice) []p25AssignedDevice {
	connected := make(map[string]SDRDevice)
	for _, device := range devices {
		if device.Connected {
			connected[device.ID] = device
		}
	}
	result := make([]p25AssignedDevice, 0, len(plan))
	seen := make(map[string]bool)
	for _, item := range plan {
		if item.DeviceID == nil || seen[*item.DeviceID] {
			continue
		}
		device, ok := connected[*item.DeviceID]
		if !ok {
			continue
		}
		role := item.Role
		if role != "control" && role != "voice" {
			role = ""
		}
		result = append(result, p25AssignedDevice{Device: device, Role: role})
		seen[device.ID] = true
	}
	for _, device := range devices {
		if device.Connected && !seen[device.ID] && len(result) == 0 {
			result = append(result, p25AssignedDevice{Device: device})
			break
		}
	}
	if len(result) == 1 {
		result[0].Role = "wideband"
	} else {
		controlAssigned := false
		for index := range result {
			if result[index].Role == "control" && !controlAssigned {
				controlAssigned = true
				continue
			}
			if result[index].Role == "" || result[index].Role == "control" {
				if !controlAssigned {
					result[index].Role, controlAssigned = "control", true
				} else {
					result[index].Role = "voice"
				}
			}
		}
		if !controlAssigned && len(result) > 0 {
			result[0].Role = "control"
		}
	}
	return result
}

func BuildGopherTrunkConfiguration(profile ScanProfile, devices []p25AssignedDevice, directory string, httpPort, grpcPort int) ([]byte, error) {
	systems := enabledP25Systems(profile)
	if len(systems) == 0 {
		return nil, errors.New("profile has no enabled P25 system")
	}
	if len(devices) == 0 {
		return nil, errors.New("P25 trunk following needs at least one assigned SDR")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	sampleRate := uint32(2_400_000)
	if len(devices) == 1 && devices[0].Device.Kind == "HackRF" {
		sampleRate = 4_000_000
	}
	var text strings.Builder
	fmt.Fprintf(&text, "log:\n  level: info\n  format: json\napi:\n  http_addr: %s\n  grpc_addr: %s\n  auth:\n    mode: auto\nmetrics:\n  enabled: false\nstorage:\n  path: %s\n  cc_cache_file: %s\nrecordings:\n  dir: %s\n  sample_rate: 8000\n  skip_encrypted: true\n  write_raw: true\nretention:\n  call_log_days: %d\n  files_days: %d\naudio:\n  enabled: false\n  device: %s\n  sample_rate: 8000\nscanner:\n  scan_mode: all\nsdr:\n  sample_rate: %d\n  autotune: true\n  devices:\n",
		yamlString(fmt.Sprintf("127.0.0.1:%d", httpPort)), yamlString(fmt.Sprintf("127.0.0.1:%d", grpcPort)),
		yamlString(filepath.Join(directory, "calls.db")), yamlString(filepath.Join(directory, "cc-cache.json")),
		yamlString(filepath.Join(directory, "Recordings")), profile.Settings.MaxRecordingDays, profile.Settings.MaxRecordingDays,
		yamlString("null"), sampleRate)
	allControl := p25ControlChannels(systems)
	if len(devices) == 1 && devices[0].Role == "wideband" && len(allControl) > 1 {
		span := allControl[len(allControl)-1] - allControl[0]
		if float64(span) > float64(sampleRate)*0.9 {
			return nil, fmt.Errorf("P25 control channels span %.3f MHz, wider than this receiver's %.3f MHz usable wideband window; assign another SDR or use a narrower system profile", float64(span)/1e6, float64(sampleRate)*0.9/1e6)
		}
	}
	for index, assigned := range devices {
		role := assigned.Role
		if role == "" {
			if index == 0 {
				role = "control"
			} else {
				role = "voice"
			}
		}
		serial := ""
		if assigned.Device.Serial != nil {
			serial = *assigned.Device.Serial
		}
		gain := "auto"
		ppm := 0
		rfAmp := false
		if calibration := assigned.Device.Calibration; calibration != nil {
			ppm = calibration.PPMCorrection
			rfAmp = calibration.AmpEnabled
			if totalGain := calibration.LNAGainDB + calibration.VGAGainDB; totalGain > 0 {
				if totalGain > 62 {
					totalGain = 62
				}
				gain = fmt.Sprintf("%d", totalGain*10)
			}
		}
		fmt.Fprintf(&text, "    - serial: %s\n      role: %s\n      ppm: %d\n      gain: %s\n      bias_tee: false\n", yamlString(serial), role, ppm, yamlString(gain))
		if assigned.Device.Kind == "HackRF" {
			fmt.Fprintf(&text, "      dc_avoid: true\n      rf_amp: %t\n", rfAmp)
		}
		if role == "wideband" {
			center := allControl[0]
			if len(allControl) > 1 {
				center = (allControl[0] + allControl[len(allControl)-1]) / 2
			}
			fmt.Fprintf(&text, "      center_freq_hz: %d\n      tuner_strategy: auto\n      voice_taps: 6\n      signalling_taps: 4\n      channels:\n", center)
			guard := float64(sampleRate) * 0.45
			for _, system := range systems {
				for _, frequency := range system.ControlChannelsHz {
					if absFloat(frequency-float64(center)) <= guard {
						fmt.Fprintf(&text, "        - frequency_hz: %d\n          system: %s\n", uint32(frequency), yamlString(system.Name))
					}
				}
			}
		}
	}
	text.WriteString("trunking:\n  voice_call_grouping: transmission\n  systems:\n")
	for index, system := range systems {
		csvName := fmt.Sprintf("talkgroups-%02d.csv", index+1)
		if err := writeGopherTalkgroups(filepath.Join(directory, csvName), system); err != nil {
			return nil, err
		}
		fmt.Fprintf(&text, "    - name: %s\n      protocol: p25\n      control_channels:\n", yamlString(system.Name))
		for _, frequency := range system.ControlChannelsHz {
			fmt.Fprintf(&text, "        - %d\n", uint32(frequency))
		}
		fmt.Fprintf(&text, "      talkgroup_file: %s\n      encrypted_calls:\n        mode: ignore\n", yamlString(filepath.Join(directory, csvName)))
	}
	return []byte(text.String()), nil
}

func p25ControlChannels(systems []P25SystemConfig) []uint32 {
	values := make([]uint32, 0)
	for _, system := range systems {
		for _, frequency := range system.ControlChannelsHz {
			values = append(values, uint32(frequency))
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func writeGopherTalkgroups(path string, system P25SystemConfig) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	_ = writer.Write([]string{"Decimal", "Alpha Tag", "Description", "Mode", "Tag", "Group", "Priority", "Lockout", "Scan", "Record", "Mute"})
	for _, talkgroup := range system.Talkgroups {
		mode := talkgroup.Mode
		if mode == "" {
			mode = "D"
		}
		locked := talkgroup.Encrypted
		_ = writer.Write([]string{strconv.Itoa(talkgroup.ID), talkgroup.Name, "", mode, "P25", system.Name, "5",
			strconv.FormatBool(locked), strconv.FormatBool(talkgroup.Enabled && !locked), "true", strconv.FormatBool(locked)})
	}
	writer.Flush()
	writeErr := writer.Error()
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func yamlString(value string) string { return strconv.Quote(value) }

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func freeTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForP25Health(apiURL string, done <-chan struct{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 750 * time.Millisecond}
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return errors.New("engine exited during startup")
		default:
		}
		response, err := client.Get(apiURL + "/api/v1/health")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("startup timed out")
}

func (m *OP25Manager) gopherAPI() (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine != "GopherTrunk" || m.apiURL == nil || m.command == nil {
		return "", false
	}
	return *m.apiURL, true
}

func (m *OP25Manager) Talkgroups() ([]P25TalkgroupState, error) {
	apiURL, ok := m.gopherAPI()
	if !ok {
		return nil, nil
	}
	var response struct {
		Talkgroups []P25TalkgroupState `json:"talkgroups"`
	}
	if err := getP25JSON(apiURL+"/api/v1/talkgroups", &response); err != nil {
		return nil, err
	}
	sort.Slice(response.Talkgroups, func(i, j int) bool { return response.Talkgroups[i].ID < response.Talkgroups[j].ID })
	return response.Talkgroups, nil
}

func (m *OP25Manager) ActiveCalls() ([]P25ActiveCall, error) {
	apiURL, ok := m.gopherAPI()
	if !ok {
		return nil, nil
	}
	var response struct {
		Calls []P25ActiveCall `json:"calls"`
	}
	if err := getP25JSON(apiURL+"/api/v1/calls/active", &response); err != nil {
		return nil, err
	}
	return response.Calls, nil
}

func getP25JSON(endpoint string, value any) error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("P25 engine returned %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 4_000_000)).Decode(value)
}

func (m *OP25Manager) UpdateTalkgroup(id uint32, fields map[string]any) error {
	apiURL, ok := m.gopherAPI()
	if !ok || len(fields) == 0 {
		return nil
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/v1/talkgroups/%d", apiURL, id), bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("P25 talkgroup update failed (%s): %s", response.Status, strings.TrimSpace(string(message)))
	}
	return nil
}

func (m *OP25Manager) EnsureAudio(talkgroupID uint32, channelID string, hub *AudioHub) {
	if talkgroupID == 0 || channelID == "" || hub == nil {
		return
	}
	m.mu.Lock()
	if m.engine != "GopherTrunk" || m.apiURL == nil || m.streamStop == nil {
		m.mu.Unlock()
		return
	}
	if _, exists := m.audioFeeds[talkgroupID]; exists {
		m.mu.Unlock()
		return
	}
	m.audioFeeds[talkgroupID] = struct{}{}
	apiURL, stop := *m.apiURL, m.streamStop
	m.mu.Unlock()
	go streamP25Talkgroup(stop, apiURL, talkgroupID, channelID, hub)
}

func streamP25Talkgroup(stop <-chan struct{}, apiURL string, talkgroupID uint32, channelID string, hub *AudioHub) {
	endpoint := apiURL + "/api/v1/audio/stream?talkgroup=" + url.QueryEscape(strconv.FormatUint(uint64(talkgroupID), 10))
	for {
		select {
		case <-stop:
			return
		default:
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-stop:
				cancel()
			case <-ctx.Done():
			}
		}()
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		response, err := (&http.Client{}).Do(request)
		if err == nil && response.StatusCode == http.StatusOK {
			header := make([]byte, 44)
			_, err = io.ReadFull(response.Body, header)
			buffer := make([]byte, 3200)
			for err == nil {
				var count int
				count, err = response.Body.Read(buffer)
				count -= count % 2
				if count > 0 {
					samples := make([]int16, count/2)
					for index := range samples {
						samples[index] = int16(binary.LittleEndian.Uint16(buffer[index*2:]))
					}
					hub.Publish(AudioFrame{ChannelID: channelID, SampleRate: 8000, Samples: samples})
				}
			}
			_ = response.Body.Close()
		} else if response != nil {
			_ = response.Body.Close()
		}
		cancel()
		select {
		case <-stop:
			return
		case <-time.After(800 * time.Millisecond):
		}
	}
}
