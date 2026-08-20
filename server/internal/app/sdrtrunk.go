package app

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const bundledSDRTrunkVersion = "0.6.1"

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

func findSDRTrunk() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("GPSDR_SDRTRUNK")); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured, nil
		}
	}
	if executable, err := findTool("sdr-trunk", "sdr-trunk.bat"); err == nil {
		return executable, nil
	}
	var candidates []string
	if current, err := os.Executable(); err == nil {
		base := filepath.Dir(current)
		candidates = append(candidates,
			filepath.Join(base, "sdrtrunk", "bin", sdrTrunkLauncher()),
			filepath.Join(base, "..", "sdrtrunk-"+runtime.GOARCH, "bin", sdrTrunkLauncher()),
			filepath.Join(base, "..", "sdrtrunk", "bin", sdrTrunkLauncher()),
		)
	}
	if runtime.GOOS == "darwin" {
		matches, _ := filepath.Glob("/Applications/sdrtrunk/sdr-trunk-osx-*-v*/bin/sdr-trunk")
		candidates = append(candidates, matches...)
	} else if runtime.GOOS != "windows" {
		candidates = append(candidates, "/usr/lib/gp-sdr/sdrtrunk/bin/sdr-trunk", "/opt/sdrtrunk/bin/sdr-trunk")
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Clean(candidate)); err == nil && !info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", errors.New("SDRTrunk is not installed")
}

func sdrTrunkLauncher() string {
	if runtime.GOOS == "windows" {
		return "sdr-trunk.bat"
	}
	return "sdr-trunk"
}

// Start launches SDRTrunk in its supported headless mode with an isolated
// GP-SDR playlist. SDRTrunk owns the selected radios for the lifetime of the
// P25 profile and sends decoded audio to the normal system output device.
func (m *OP25Manager) Start(profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	m.restartMu.Lock()
	defer m.restartMu.Unlock()
	return m.start(profile, plan, devices, dataDirectory)
}

func (m *OP25Manager) start(profile ScanProfile, plan []ReceiverPlanItem, devices []SDRDevice, dataDirectory string) error {
	m.stop(false)
	executable, err := findSDRTrunk()
	if err != nil {
		return errors.New("the SDRTrunk P25 engine is missing; reinstall the complete GP-SDR package")
	}
	assigned := p25DeviceAssignments(plan, devices)
	if len(assigned) == 0 {
		return errors.New("P25 trunk following needs at least one assigned SDR")
	}
	runtimeDirectory := filepath.Join(dataDirectory, "Runtime", "P25", profile.ID)
	applicationRoot := filepath.Join(runtimeDirectory, "home", "SDRTrunk")
	for _, directory := range []string{"playlist", "configuration", "settings", "event_logs", "recordings", "jmbe"} {
		if err := os.MkdirAll(filepath.Join(applicationRoot, directory), 0o700); err != nil {
			return err
		}
	}
	m.mu.Lock()
	muted := cloneMuteMap(m.muted)
	m.mu.Unlock()
	preferred := preferredSDRTrunkTuner(assigned)
	playlist, err := BuildSDRTrunkPlaylist(profile, preferred, muted)
	if err != nil {
		return err
	}
	playlistPath := filepath.Join(applicationRoot, "playlist", "default.xml")
	if err := os.WriteFile(playlistPath, playlist, 0o600); err != nil {
		return err
	}
	_ = importExistingSDRTrunkTunerConfiguration(applicationRoot)
	jmbePath := findJMBELibrary(dataDirectory, applicationRoot)
	preferencesRoot := filepath.Join(runtimeDirectory, "java-preferences")
	if jmbePath != "" {
		if err := writeJMBEPreferences(preferencesRoot, jmbePath); err != nil {
			return err
		}
		if runtime.GOOS == "darwin" {
			if err := configureMacJMBEPreference(executable, runtimeDirectory, jmbePath); err != nil {
				return fmt.Errorf("configure P25 voice codec: %w", err)
			}
		}
	}
	logPath := filepath.Join(runtimeDirectory, "sdrtrunk.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	command := platformCommand(executable)
	command.Dir = filepath.Dir(filepath.Dir(executable))
	command.Stdout, command.Stderr = logFile, logFile
	home := filepath.Join(runtimeDirectory, "home")
	javaOptions := []string{
		"-Djava.awt.headless=true",
		"-Duser.home=" + home,
		"-Xmx1200m",
	}
	// SDRTrunk's macOS runtime uses the native CFPreferences implementation;
	// replacing it with FileSystemPreferences triggers an UnsatisfiedLinkError.
	// Linux and Windows can safely use an isolated preferences tree.
	if runtime.GOOS != "darwin" {
		javaOptions = append(javaOptions,
			"-Djava.util.prefs.PreferencesFactory=java.util.prefs.FileSystemPreferencesFactory",
			"-Djava.util.prefs.userRoot="+preferencesRoot,
		)
	}
	quotedJavaOptions := make([]string, 0, len(javaOptions))
	for _, option := range javaOptions {
		quotedJavaOptions = append(quotedJavaOptions, quoteLauncherOption(option))
	}
	command.Env = append(os.Environ(), "JAVA_OPTS="+strings.Join(quotedJavaOptions, " "))
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start SDRTrunk: %w", err)
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
	m.profileID, m.configPath, m.lastError, m.waitError = &id, &playlistPath, nil, nil
	m.engine, m.apiURL = "SDRTrunk", nil
	m.profile, m.plan, m.devices, m.dataRoot = &profile, append([]ReceiverPlanItem(nil), plan...), append([]SDRDevice(nil), devices...), dataDirectory
	if m.muted == nil {
		m.muted = make(map[uint32]bool)
	}
	m.mu.Unlock()
	if err := waitForSDRTrunkReady(logPath, done, 20*time.Second); err != nil {
		diagnostics := strings.TrimSpace(tailText(logPath, 32_000))
		m.stop(false)
		return fmt.Errorf("SDRTrunk did not become ready: %w (log: %s)\n%s", err, logPath, diagnostics)
	}
	return nil
}

func quoteLauncherOption(value string) string {
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return `'` + strings.ReplaceAll(value, `'`, `'"'"'`) + `'`
}

func (m *OP25Manager) Stop() {
	m.restartMu.Lock()
	defer m.restartMu.Unlock()
	m.stop(true)
}

func (m *OP25Manager) stop(clearSession bool) {
	m.mu.Lock()
	if clearSession && m.restartTimer != nil {
		m.restartTimer.Stop()
		m.restartTimer = nil
	}
	m.mu.Unlock()
	m.stopProcess()
	m.mu.Lock()
	m.engine, m.apiURL = "", nil
	if clearSession {
		m.profile, m.plan, m.devices, m.dataRoot = nil, nil, nil, ""
	}
	m.mu.Unlock()
}

func (m *OP25Manager) Status() P25Status {
	m.mu.Lock()
	engine, command, done, waitError := m.engine, m.command, m.done, m.waitError
	profileID, configPath := m.profileID, m.configPath
	m.mu.Unlock()
	if engine == "SDRTrunk" && command != nil {
		select {
		case <-done:
			note := "SDRTrunk stopped."
			if waitError != nil {
				note = waitError.Error()
			}
			return P25Status{State: "error", Engine: engine, Executable: ptr(command.Path), ProfileID: profileID, ConfigPath: configPath, Note: note}
		default:
			status := P25Status{State: "running", Engine: engine, Executable: ptr(command.Path), ProfileID: profileID, ConfigPath: configPath,
				Reception: "searching", Note: "SDRTrunk is checking the configured P25 control channels."}
			if configPath != nil {
				root := filepath.Dir(filepath.Dir(*configPath))
				locked, calls := inspectSDRTrunkEvents(filepath.Join(root, "event_logs"))
				logText := tailText(filepath.Join(filepath.Dir(filepath.Dir(root)), "sdrtrunk.log"), 256_000)
				if strings.Contains(logText, "JMBE audio conversion library") && strings.Contains(logText, "successfully loaded") {
					status.Note += " P25 voice codec is loaded."
				} else {
					status.Note += " Control decoding is available; create the JMBE library in Setup to enable P25 voice."
				}
				if locked || calls > 0 {
					status.Reception = "locked"
					status.Note = "P25 control channel locked; SDRTrunk is following traffic and decoding available voice."
				}
			}
			return status
		}
	}
	if executable, err := findSDRTrunk(); err == nil {
		return P25Status{State: "ready", Engine: "SDRTrunk", Executable: &executable, Note: "SDRTrunk P25 Phase 1/2 trunk following is ready."}
	}
	return P25Status{State: "setup", Engine: "none", Note: "The SDRTrunk component is not present in this package."}
}

func waitForSDRTrunkReady(logPath string, done <-chan struct{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return errors.New("engine exited during startup")
		default:
		}
		text := tailText(logPath, 96_000)
		if strings.Contains(text, "starting main application headless") && strings.Contains(text, "Auto-starting channel") {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("startup timed out")
}

func BuildSDRTrunkPlaylist(profile ScanProfile, preferred string, muted map[uint32]bool) ([]byte, error) {
	systems := enabledP25Systems(profile)
	if len(systems) == 0 {
		return nil, errors.New("profile has no enabled P25 system")
	}
	var text strings.Builder
	text.WriteString("<playlist version=\"4\">\n")
	for _, system := range systems {
		listName := system.Name + " Talkgroups"
		for _, talkgroup := range system.Talkgroups {
			if talkgroup.ID <= 0 {
				continue
			}
			priority := 100
			if talkgroup.Encrypted || !talkgroup.Enabled || muted[uint32(talkgroup.ID)] {
				priority = -1
			}
			fmt.Fprintf(&text, "  <alias name=\"%s\" list=\"%s\" group=\"%s\">\n", xmlValue(talkgroup.Name), xmlValue(listName), xmlValue(system.Name))
			fmt.Fprintf(&text, "    <id type=\"talkgroup\" protocol=\"APCO25\" value=\"%d\"/>\n", talkgroup.ID)
			fmt.Fprintf(&text, "    <id type=\"priority\" priority=\"%d\"/>\n", priority)
			text.WriteString("  </alias>\n")
		}
		fmt.Fprintf(&text, "  <channel system=\"%s\" enabled=\"true\" site=\"%s\" order=\"1\" name=\"%s\">\n", xmlValue(system.Name), xmlValue(system.Name), xmlValue(system.Name))
		text.WriteString("    <aux_decode_configuration/>\n")
		text.WriteString("    <decode_configuration type=\"decodeConfigP25Phase1\" modulation=\"CQPSK\" traffic_channel_pool_size=\"30\" ignore_data_calls=\"false\"/>\n")
		text.WriteString("    <event_log_configuration><logger>DECODED_MESSAGE</logger><logger>TRAFFIC_DECODED_MESSAGE</logger><logger>CALL_EVENT</logger><logger>TRAFFIC_CALL_EVENT</logger></event_log_configuration>\n")
		text.WriteString("    <record_configuration/>\n")
		fmt.Fprintf(&text, "    <source_configuration type=\"sourceConfigTunerMultipleFrequency\" frequency_rotation_delay=\"400\" source_type=\"TUNER_MULTIPLE_FREQUENCIES\"")
		if preferred != "" {
			fmt.Fprintf(&text, " preferred_tuner=\"%s\"", xmlValue(preferred))
		}
		text.WriteString(">\n")
		for _, frequency := range system.ControlChannelsHz {
			fmt.Fprintf(&text, "      <frequency>%d</frequency>\n", uint64(frequency))
		}
		text.WriteString("    </source_configuration>\n")
		fmt.Fprintf(&text, "    <alias_list_name>%s</alias_list_name>\n", xmlValue(listName))
		text.WriteString("  </channel>\n")
	}
	text.WriteString("</playlist>\n")
	return []byte(text.String()), nil
}

func xmlValue(value string) string {
	var text strings.Builder
	_ = xml.EscapeText(&text, []byte(value))
	return text.String()
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
		if device, ok := connected[*item.DeviceID]; ok {
			result = append(result, p25AssignedDevice{Device: device, Role: item.Role})
			seen[device.ID] = true
		}
	}
	if len(result) == 0 {
		for _, device := range devices {
			if device.Connected {
				result = append(result, p25AssignedDevice{Device: device, Role: "control"})
				break
			}
		}
	}
	return result
}

func preferredSDRTrunkTuner(devices []p25AssignedDevice) string {
	if len(devices) != 1 || devices[0].Device.Kind != "HackRF" || devices[0].Device.Serial == nil {
		return ""
	}
	serial := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(*devices[0].Device.Serial), "-", ""))
	if len(serial) == 32 {
		serial = serial[0:8] + "-" + serial[8:16] + "-" + serial[16:24] + "-" + serial[24:32]
	}
	return "HackRF ONE " + serial
}

func importExistingSDRTrunkTunerConfiguration(applicationRoot string) error {
	destination := filepath.Join(applicationRoot, "configuration", "tuner_configuration.json")
	if _, err := os.Stat(destination); err == nil {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	source := filepath.Join(home, "SDRTrunk", "configuration", "tuner_configuration.json")
	data, err := os.ReadFile(source)
	if err != nil {
		return nil
	}
	var value any
	if json.Unmarshal(data, &value) != nil {
		return nil
	}
	return os.WriteFile(destination, data, 0o600)
}

func findJMBELibrary(dataDirectory, applicationRoot string) string {
	if configured := strings.TrimSpace(os.Getenv("GPSDR_JMBE")); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured
		}
	}
	home, _ := os.UserHomeDir()
	patterns := []string{
		filepath.Join(dataDirectory, "Components", "JMBE", "jmbe-*.jar"),
		filepath.Join(applicationRoot, "jmbe", "jmbe-*.jar"),
		filepath.Join(home, "SDRTrunk", "jmbe", "jmbe-*.jar"),
	}
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			sort.Strings(matches)
			return matches[len(matches)-1]
		}
	}
	return ""
}

func writeJMBEPreferences(root, jmbePath string) error {
	directory := filepath.Join(root, ".java", ".userPrefs", "io", "github", "dsheirer", "preference", "decoder")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	content := "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n<!DOCTYPE map SYSTEM \"http://java.sun.com/dtd/preferences.dtd\">\n<map MAP_XML_VERSION=\"1.0\"><entry key=\"path.jmbe.library.1.0.0\" value=\"" + xmlValue(jmbePath) + "\"/></map>\n"
	return os.WriteFile(filepath.Join(directory, "prefs.xml"), []byte(content), 0o600)
}

func configureMacJMBEPreference(sdrTrunkExecutable, runtimeDirectory, jmbePath string) error {
	if helper, err := findTool("gpsdr-mac-prefs"); err == nil {
		output, runErr := exec.Command(helper, "set-jmbe", jmbePath).CombinedOutput()
		if runErr != nil {
			return fmt.Errorf("%w: %s", runErr, strings.TrimSpace(string(output)))
		}
		return nil
	}
	// Development builds can use a full system JDK. The runtime bundled with
	// SDRTrunk intentionally omits jdk.compiler, so it cannot launch source files.
	java, err := findTool("java")
	if err != nil || filepath.Clean(java) == filepath.Join(filepath.Dir(sdrTrunkExecutable), "java") {
		return errors.New("GP-SDR's macOS preference helper is missing")
	}
	sourcePath := filepath.Join(runtimeDirectory, "JmbePreferenceWriter.java")
	const source = `import java.util.prefs.Preferences;
public class JmbePreferenceWriter {
  public static void main(String[] args) throws Exception {
    Preferences node = Preferences.userRoot().node("/io/github/dsheirer/preference/decoder");
    node.put("path.jmbe.library.1.0.0", args[0]);
    node.flush();
  }
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		return err
	}
	output, err := exec.Command(java, sourcePath, jmbePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func inspectSDRTrunkEvents(directory string) (bool, int) {
	files, _ := filepath.Glob(filepath.Join(directory, "*_decoded_messages.log"))
	locked := false
	for _, file := range files {
		for _, line := range strings.Split(tailText(file, 256_000), "\n") {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, ",PASSED,NAC:") && (strings.Contains(upper, "TSBK") || strings.Contains(upper, "PDU") || strings.Contains(upper, "TDULC")) {
				locked = true
				break
			}
		}
	}
	callFiles, _ := filepath.Glob(filepath.Join(directory, "*_call_events.log"))
	calls := 0
	for _, file := range callFiles {
		reader := csv.NewReader(strings.NewReader(tailText(file, 512_000)))
		for {
			row, err := reader.Read()
			if err != nil {
				break
			}
			if len(row) > 3 && row[0] != "TIMESTAMP" && strings.Contains(strings.ToUpper(row[3]), "CALL") {
				calls++
			}
		}
	}
	return locked, calls
}

func tailText(path string, limit int64) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	if info, err := file.Stat(); err == nil && info.Size() > limit {
		_, _ = file.Seek(info.Size()-limit, io.SeekStart)
	}
	data, _ := io.ReadAll(io.LimitReader(file, limit))
	return string(data)
}

func (m *OP25Manager) Talkgroups() ([]P25TalkgroupState, error) {
	m.mu.Lock()
	profile, configPath, muted := m.profile, m.configPath, cloneMuteMap(m.muted)
	m.mu.Unlock()
	if profile == nil {
		return nil, nil
	}
	items := make([]P25TalkgroupState, 0)
	seen := make(map[uint32]bool)
	for _, system := range enabledP25Systems(*profile) {
		for _, talkgroup := range system.Talkgroups {
			id := uint32(talkgroup.ID)
			if id == 0 || seen[id] {
				continue
			}
			seen[id] = true
			items = append(items, P25TalkgroupState{ID: id, AlphaTag: talkgroup.Name, Group: system.Name, Mode: talkgroup.Mode,
				Priority: 100, Lockout: talkgroup.Encrypted, Scan: talkgroup.Enabled && !talkgroup.Encrypted, Record: true,
				Mute: talkgroup.Encrypted || !talkgroup.Enabled || muted[id]})
		}
	}
	if configPath != nil {
		root := filepath.Dir(filepath.Dir(*configPath))
		files, _ := filepath.Glob(filepath.Join(root, "event_logs", "*_call_events.log"))
		defaultSystem := "P25"
		if systems := enabledP25Systems(*profile); len(systems) > 0 {
			defaultSystem = systems[0].Name
		}
		for _, file := range files {
			reader := csv.NewReader(strings.NewReader(tailText(file, 1_000_000)))
			for {
				row, err := reader.Read()
				if err != nil {
					break
				}
				if len(row) < 6 || !strings.Contains(strings.ToUpper(row[3]), "GROUP CALL") {
					continue
				}
				id := parseLastDecimal(row[5])
				if id == 0 || seen[id] {
					continue
				}
				seen[id] = true
				items = append(items, P25TalkgroupState{ID: id, AlphaTag: fmt.Sprintf("Talkgroup %d", id), Group: defaultSystem,
					Mode: "D", Priority: 100, Scan: true, Record: true, Mute: muted[id], Discovered: true})
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

var decimalIdentifier = regexp.MustCompile(`\d+`)

func (m *OP25Manager) ActiveCalls() ([]P25ActiveCall, error) {
	m.mu.Lock()
	profile, configPath := m.profile, m.configPath
	m.mu.Unlock()
	if profile == nil || configPath == nil {
		return nil, nil
	}
	talkgroups, _ := m.Talkgroups()
	byID := make(map[uint32]*P25TalkgroupState)
	for index := range talkgroups {
		item := talkgroups[index]
		byID[item.ID] = &item
	}
	root := filepath.Dir(filepath.Dir(*configPath))
	files, _ := filepath.Glob(filepath.Join(root, "event_logs", "*_call_events.log"))
	now := time.Now()
	byEventID := make(map[string]P25ActiveCall)
	for _, file := range files {
		reader := csv.NewReader(strings.NewReader(tailText(file, 512_000)))
		for {
			row, err := reader.Read()
			if err != nil {
				break
			}
			if len(row) < 11 || row[0] == "TIMESTAMP" || !strings.Contains(strings.ToUpper(row[3]), "CALL") {
				continue
			}
			started, err := parseSDRTrunkTimestamp(row[0])
			if err != nil {
				continue
			}
			durationMS, _ := strconv.ParseInt(strings.TrimSpace(row[1]), 10, 64)
			last := started.Add(time.Duration(durationMS) * time.Millisecond)
			if durationMS == 0 {
				last = started.Add(2 * time.Second)
			}
			if now.Sub(last) > 3*time.Second || last.Sub(now) > time.Minute {
				continue
			}
			group := parseLastDecimal(row[5])
			source := parseLastDecimal(row[4])
			frequencyMHz, _ := strconv.ParseFloat(strings.TrimSpace(row[7]), 64)
			systemName := "P25"
			if systems := enabledP25Systems(*profile); len(systems) > 0 {
				systemName = systems[0].Name
			}
			protocol := row[2]
			if strings.Contains(strings.ToUpper(strings.Join(row[6:], " ")), "PHASE 2") {
				protocol = "P25 Phase 2"
			}
			call := P25ActiveCall{Grant: p25GrantState{System: systemName, Protocol: protocol, GroupID: group, SourceID: source,
				FrequencyHz: uint32(frequencyMHz * 1e6), Encrypted: strings.Contains(strings.ToUpper(row[3]+" "+row[9]), "ENCRYPT")},
				Talkgroup: byID[group], StartedAt: started, LastHeardAt: last, Following: true}
			eventID := row[len(row)-1]
			if existing, ok := byEventID[eventID]; !ok || call.LastHeardAt.After(existing.LastHeardAt) {
				byEventID[eventID] = call
			}
		}
	}
	result := make([]P25ActiveCall, 0, len(byEventID))
	for _, call := range byEventID {
		result = append(result, call)
	}
	return result, nil
}

func parseSDRTrunkTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006:01:02:15:04:05", "20060102 150405"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported SDRTrunk timestamp %q", value)
}

func parseLastDecimal(value string) uint32 {
	values := decimalIdentifier.FindAllString(value, -1)
	if len(values) == 0 {
		return 0
	}
	parsed, _ := strconv.ParseUint(values[len(values)-1], 10, 32)
	return uint32(parsed)
}

// SDRTrunk reads aliases when a playlist starts. Applying a talkgroup mute
// regenerates the isolated playlist and performs a short, supervised restart.
func (m *OP25Manager) UpdateTalkgroup(id uint32, fields map[string]any) error {
	value, present := fields["mute"]
	if !present {
		return nil
	}
	muted, ok := value.(bool)
	if !ok {
		return errors.New("mute must be true or false")
	}
	m.mu.Lock()
	if m.muted == nil {
		m.muted = make(map[uint32]bool)
	}
	m.muted[id] = muted
	if m.profile != nil && m.dataRoot != "" {
		if m.restartTimer != nil {
			m.restartTimer.Stop()
		}
		m.restartTimer = time.AfterFunc(350*time.Millisecond, func() {
			m.mu.Lock()
			profile, plan, devices, dataRoot := m.profile, append([]ReceiverPlanItem(nil), m.plan...), append([]SDRDevice(nil), m.devices...), m.dataRoot
			m.mu.Unlock()
			if profile == nil || dataRoot == "" {
				return
			}
			m.restartMu.Lock()
			err := m.start(*profile, plan, devices, dataRoot)
			m.restartMu.Unlock()
			if err != nil {
				message := err.Error()
				m.mu.Lock()
				m.lastError = &message
				m.mu.Unlock()
			}
		})
	}
	m.mu.Unlock()
	return nil
}

func cloneMuteMap(source map[uint32]bool) map[uint32]bool {
	result := make(map[uint32]bool, len(source))
	for id, value := range source {
		result[id] = value
	}
	return result
}

// Audio is played by SDRTrunk through the selected native system output. The
// GP-SDR audio hub remains available for analog and remote PCM receivers.
func (m *OP25Manager) EnsureAudio(uint32, string, *AudioHub) {}
