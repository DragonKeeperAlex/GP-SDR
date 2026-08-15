package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRangeSyncMinutes = 360
	minimumRangeSyncMinutes = 15
	maximumRangeSyncMinutes = 10_080
	maximumRangeSheetBytes  = 5_000_000
)

type RangeSyncConfig struct {
	Enabled         bool   `json:"enabled"`
	SheetURL        string `json:"sheetURL"`
	IntervalMinutes int    `json:"intervalMinutes"`
}

type RangeSyncStatus struct {
	Config       RangeSyncConfig `json:"config"`
	Syncing      bool            `json:"syncing"`
	LastAttempt  *time.Time      `json:"lastAttempt,omitempty"`
	LastSync     *time.Time      `json:"lastSync,omitempty"`
	NextSync     *time.Time      `json:"nextSync,omitempty"`
	LastError    *string         `json:"lastError,omitempty"`
	ProfileCount int             `json:"profileCount"`
	RangeCount   int             `json:"rangeCount"`
	UsingCache   bool            `json:"usingCache"`
}

type rangeSyncCache struct {
	FetchedAt time.Time     `json:"fetchedAt"`
	Profiles  []ScanProfile `json:"profiles"`
}

type RangeSyncManager struct {
	mu           sync.RWMutex
	dir          string
	profiles     *ProfileStore
	client       *http.Client
	config       RangeSyncConfig
	lastAttempt  *time.Time
	lastSync     *time.Time
	lastError    *string
	profileCount int
	rangeCount   int
	usingCache   bool
	syncing      bool
	wake         chan struct{}
}

func NewRangeSyncManager(dataDirectory string, profiles *ProfileStore) (*RangeSyncManager, error) {
	dir := filepath.Join(dataDirectory, "RangeSync")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	manager := &RangeSyncManager{
		dir: dir, profiles: profiles, wake: make(chan struct{}, 1),
		config: RangeSyncConfig{IntervalMinutes: defaultRangeSyncMinutes},
		client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: safeGoogleRedirect},
	}
	manager.loadConfig()
	manager.loadCache()
	go manager.loop()
	if manager.config.Enabled && manager.config.SheetURL != "" {
		manager.notify()
	}
	return manager, nil
}

func safeGoogleRedirect(request *http.Request, via []*http.Request) error {
	if len(via) >= 6 {
		return errors.New("Google Sheets redirected too many times")
	}
	if request.URL.Scheme != "https" || (request.URL.Port() != "" && request.URL.Port() != "443") || !allowedGoogleSheetHost(request.URL.Hostname()) {
		return errors.New("Google Sheets redirected to an unsupported host")
	}
	return nil
}

func allowedGoogleSheetHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "docs.google.com" || host == "docs.googleusercontent.com" || strings.HasSuffix(host, ".googleusercontent.com")
}

func (m *RangeSyncManager) Status() RangeSyncStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := RangeSyncStatus{Config: m.config, Syncing: m.syncing, LastAttempt: m.lastAttempt, LastSync: m.lastSync,
		LastError: m.lastError, ProfileCount: m.profileCount, RangeCount: m.rangeCount, UsingCache: m.usingCache}
	if m.config.Enabled && m.lastAttempt != nil {
		next := m.lastAttempt.Add(time.Duration(m.config.IntervalMinutes) * time.Minute)
		status.NextSync = &next
	}
	return status
}

func (m *RangeSyncManager) Update(config RangeSyncConfig) (RangeSyncStatus, error) {
	config.SheetURL = strings.TrimSpace(config.SheetURL)
	if config.IntervalMinutes == 0 {
		config.IntervalMinutes = defaultRangeSyncMinutes
	}
	if config.IntervalMinutes < minimumRangeSyncMinutes || config.IntervalMinutes > maximumRangeSyncMinutes {
		return m.Status(), fmt.Errorf("sync interval must be between %d and %d minutes", minimumRangeSyncMinutes, maximumRangeSyncMinutes)
	}
	if config.SheetURL != "" {
		if _, err := googleSheetCSVURL(config.SheetURL); err != nil {
			return m.Status(), err
		}
	}
	if config.Enabled && config.SheetURL == "" {
		return m.Status(), errors.New("a shared Google Sheets link is required")
	}
	m.mu.Lock()
	m.config = config
	if !config.Enabled {
		m.lastError = nil
	}
	m.mu.Unlock()
	if err := m.persistConfig(); err != nil {
		return m.Status(), err
	}
	if config.Enabled {
		m.notify()
	}
	return m.Status(), nil
}

func (m *RangeSyncManager) SyncNow() RangeSyncStatus {
	_ = m.sync()
	return m.Status()
}

func (m *RangeSyncManager) notify() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *RangeSyncManager) loop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.wake:
			_ = m.sync()
		case <-ticker.C:
			status := m.Status()
			if !status.Config.Enabled || status.Syncing {
				continue
			}
			if status.LastAttempt == nil || time.Since(*status.LastAttempt) >= time.Duration(status.Config.IntervalMinutes)*time.Minute {
				_ = m.sync()
			}
		}
	}
}

func (m *RangeSyncManager) sync() error {
	m.mu.Lock()
	if m.syncing {
		m.mu.Unlock()
		return errors.New("range sync is already running")
	}
	config := m.config
	if config.SheetURL == "" {
		message := "a shared Google Sheets link is required"
		m.lastError = &message
		m.mu.Unlock()
		return errors.New(message)
	}
	m.syncing = true
	now := time.Now().UTC()
	m.lastAttempt = &now
	m.mu.Unlock()

	profiles, err := m.fetch(config.SheetURL)
	m.mu.Lock()
	m.syncing = false
	if err != nil {
		message := err.Error()
		m.lastError = &message
		m.usingCache = m.profileCount > 0
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	if err = m.profiles.ReplaceSyncedProfiles(profiles); err != nil {
		m.setError(err)
		return err
	}
	fetchedAt := time.Now().UTC()
	cache := rangeSyncCache{FetchedAt: fetchedAt, Profiles: profiles}
	if err = writeJSONAtomic(filepath.Join(m.dir, "cache.json"), cache); err != nil {
		m.setError(err)
		return err
	}
	ranges := 0
	for _, profile := range profiles {
		ranges += len(profile.Ranges)
	}
	m.mu.Lock()
	m.lastSync = &fetchedAt
	m.lastError = nil
	m.profileCount = len(profiles)
	m.rangeCount = ranges
	m.usingCache = false
	m.mu.Unlock()
	return nil
}

func (m *RangeSyncManager) setError(err error) {
	message := err.Error()
	m.mu.Lock()
	m.syncing = false
	m.lastError = &message
	m.mu.Unlock()
}

func (m *RangeSyncManager) fetch(sheetURL string) ([]ScanProfile, error) {
	csvURL, err := googleSheetCSVURL(sheetURL)
	if err != nil {
		return nil, err
	}
	request, _ := http.NewRequest(http.MethodGet, csvURL, nil)
	request.Header.Set("User-Agent", "GP-SDR/1.0 range-sync")
	response, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not download the Google Sheet: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Sheets returned %s; confirm link sharing is enabled", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maximumRangeSheetBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maximumRangeSheetBytes {
		return nil, errors.New("Google Sheet export is larger than 5 MB")
	}
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("<")) {
		return nil, errors.New("Google returned a web page instead of CSV; share the sheet with anyone who has the link")
	}
	return parseRangeSheetCSV(data)
}

func googleSheetCSVURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "docs.google.com") || (parsed.Port() != "" && parsed.Port() != "443") {
		return "", errors.New("use an HTTPS docs.google.com spreadsheet link")
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 3 || segments[0] != "spreadsheets" || segments[1] != "d" {
		return "", errors.New("the link is not a Google Sheets document")
	}
	query := parsed.Query()
	if query.Get("gid") == "" && parsed.Fragment != "" {
		fragment, _ := url.ParseQuery(parsed.Fragment)
		query.Set("gid", fragment.Get("gid"))
	}
	gid := query.Get("gid")
	if gid == "" {
		gid = "0"
	}
	if segments[2] == "e" {
		if len(segments) < 4 || strings.TrimSpace(segments[3]) == "" {
			return "", errors.New("the published Google Sheets link is incomplete")
		}
		return "https://docs.google.com/spreadsheets/d/e/" + url.PathEscape(segments[3]) + "/pub?output=csv&gid=" + url.QueryEscape(gid), nil
	}
	if strings.TrimSpace(segments[2]) == "" {
		return "", errors.New("the Google Sheets link is incomplete")
	}
	return "https://docs.google.com/spreadsheets/d/" + url.PathEscape(segments[2]) + "/export?format=csv&gid=" + url.QueryEscape(gid), nil
}

func parseRangeSheetCSV(data []byte) ([]ScanProfile, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("the range sheet needs a header row")
	}
	columns := make(map[string]int, len(header))
	for index, value := range header {
		columns[normalizeColumn(value)] = index
	}
	startIndex, startOK := firstColumn(columns, "startmhz", "lowmhz", "startfrequency", "startfrequencyhz", "starthz", "start")
	endIndex, endOK := firstColumn(columns, "endmhz", "highmhz", "endfrequency", "endfrequencyhz", "endhz", "end")
	if !startOK || !endOK {
		return nil, errors.New("the sheet needs Start MHz and End MHz columns")
	}
	profileIndex, hasProfile := firstColumn(columns, "profile", "profilename", "bank", "group")
	nameIndex, hasName := firstColumn(columns, "name", "rangename", "range", "label")
	stepIndex, hasStep := firstColumn(columns, "stepkhz", "stephz", "step")
	modeIndex, hasMode := firstColumn(columns, "mode", "preferredmode", "modulation")
	dwellIndex, hasDwell := firstColumn(columns, "dwellms", "dwellmilliseconds", "dwell")
	enabledIndex, hasEnabled := firstColumn(columns, "enabled", "active", "scan")
	summaryIndex, hasSummary := firstColumn(columns, "summary", "note", "description")

	type profileRows struct {
		summary string
		ranges  []ScanRange
	}
	groups := make(map[string]*profileRows)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("invalid range row %d: %w", rowNumber, readErr)
		}
		if startIndex >= len(record) || endIndex >= len(record) || (strings.TrimSpace(record[startIndex]) == "" && strings.TrimSpace(record[endIndex]) == "") {
			continue
		}
		start, startErr := parseRangeFrequency(record[startIndex], header[startIndex], false)
		end, endErr := parseRangeFrequency(record[endIndex], header[endIndex], false)
		if startErr != nil || endErr != nil || start <= 0 || end <= start {
			return nil, fmt.Errorf("range row %d has invalid start or end frequencies", rowNumber)
		}
		profileName := "Google Sheet Ranges"
		if hasProfile && profileIndex < len(record) && strings.TrimSpace(record[profileIndex]) != "" {
			profileName = truncateText(strings.TrimSpace(record[profileIndex]), 80)
		}
		rangeName := fmt.Sprintf("Range %d", rowNumber-1)
		if hasName && nameIndex < len(record) && strings.TrimSpace(record[nameIndex]) != "" {
			rangeName = truncateText(strings.TrimSpace(record[nameIndex]), 80)
		}
		step := 12_500.0
		if hasStep && stepIndex < len(record) && strings.TrimSpace(record[stepIndex]) != "" {
			step, err = parseRangeFrequency(record[stepIndex], header[stepIndex], true)
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("range row %d has an invalid step", rowNumber)
			}
		}
		dwell := 180
		if hasDwell && dwellIndex < len(record) && strings.TrimSpace(record[dwellIndex]) != "" {
			dwell, err = strconv.Atoi(strings.TrimSpace(record[dwellIndex]))
			if err != nil || dwell < 20 || dwell > 60_000 {
				return nil, fmt.Errorf("range row %d has an invalid dwell time", rowNumber)
			}
		}
		mode := "auto"
		if hasMode && modeIndex < len(record) {
			mode = normalizeImportedMode(record[modeIndex])
		}
		enabled := true
		if hasEnabled && enabledIndex < len(record) && strings.TrimSpace(record[enabledIndex]) != "" {
			enabled, err = parseSheetBoolean(record[enabledIndex])
			if err != nil {
				return nil, fmt.Errorf("range row %d has an invalid Enabled value", rowNumber)
			}
		}
		group := groups[profileName]
		if group == nil {
			group = &profileRows{}
			groups[profileName] = group
		}
		if hasSummary && summaryIndex < len(record) && group.summary == "" {
			group.summary = truncateText(strings.TrimSpace(record[summaryIndex]), 160)
		}
		group.ranges = append(group.ranges, ScanRange{ID: stableSheetID("range", profileName, rangeName, fmt.Sprintf("%.0f", start), fmt.Sprintf("%.0f", end)),
			Name: rangeName, StartHz: start, EndHz: end, StepHz: step, DwellMilliseconds: dwell, PreferredMode: mode, Enabled: enabled})
		if len(group.ranges) > 100 {
			return nil, fmt.Errorf("profile %q contains more than 100 ranges", profileName)
		}
	}
	if len(groups) == 0 {
		return nil, errors.New("the Google Sheet contains no valid range rows")
	}
	if len(groups) > 100 {
		return nil, errors.New("the Google Sheet contains more than 100 profiles")
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	profiles := make([]ScanProfile, 0, len(names))
	for _, name := range names {
		group := groups[name]
		summary := group.summary
		if summary == "" {
			summary = "Automatically synced from Google Sheets"
		}
		target := name
		profile := ScanProfile{SchemaVersion: 1, ID: stableSheetID("sheet", name), Name: name, Summary: summary, Ranges: group.ranges,
			Channels: []ChannelDefinition{}, DeviceAssignments: []DeviceAssignment{{ID: stableSheetID("receiver", name), Role: "discovery", Target: &target}},
			P25Systems: []P25SystemConfig{}, Settings: defaultSettings(), BuiltIn: true}
		if err := validateProfile(profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func parseRangeFrequency(value, heading string, step bool) (float64, error) {
	clean := strings.ToLower(strings.TrimSpace(value))
	multiplier := 1.0
	switch {
	case strings.Contains(clean, "ghz"):
		multiplier, clean = 1e9, strings.ReplaceAll(clean, "ghz", "")
	case strings.Contains(clean, "mhz"):
		multiplier, clean = 1e6, strings.ReplaceAll(clean, "mhz", "")
	case strings.Contains(clean, "khz"):
		multiplier, clean = 1e3, strings.ReplaceAll(clean, "khz", "")
	case strings.Contains(clean, "hz"):
		clean = strings.ReplaceAll(clean, "hz", "")
	case strings.Contains(normalizeColumn(heading), "mhz"):
		multiplier = 1e6
	case strings.Contains(normalizeColumn(heading), "khz"):
		multiplier = 1e3
	case !strings.Contains(normalizeColumn(heading), "hz"):
		if step {
			multiplier = 1e3
		} else {
			multiplier = 1e6
		}
	}
	clean = strings.NewReplacer(",", "", " ", "", "_", "").Replace(clean)
	number, err := strconv.ParseFloat(clean, 64)
	return number * multiplier, err
}

func parseSheetBoolean(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true, nil
	case "0", "false", "no", "n", "off", "disabled":
		return false, nil
	default:
		return false, errors.New("invalid boolean")
	}
}

func stableSheetID(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(strings.Join(parts, "\x00"))))
	return "sheet-" + hex.EncodeToString(hash[:10])
}

func (m *RangeSyncManager) loadConfig() {
	data, err := os.ReadFile(filepath.Join(m.dir, "config.json"))
	if err != nil {
		return
	}
	var config RangeSyncConfig
	if json.Unmarshal(data, &config) == nil && config.IntervalMinutes >= minimumRangeSyncMinutes && config.IntervalMinutes <= maximumRangeSyncMinutes {
		m.config = config
	}
}

func (m *RangeSyncManager) loadCache() {
	data, err := os.ReadFile(filepath.Join(m.dir, "cache.json"))
	if err != nil {
		return
	}
	var cache rangeSyncCache
	if json.Unmarshal(data, &cache) != nil || len(cache.Profiles) == 0 {
		return
	}
	if m.profiles.ReplaceSyncedProfiles(cache.Profiles) != nil {
		return
	}
	ranges := 0
	for _, profile := range cache.Profiles {
		ranges += len(profile.Ranges)
	}
	m.lastSync = &cache.FetchedAt
	m.profileCount = len(cache.Profiles)
	m.rangeCount = ranges
	m.usingCache = true
}

func (m *RangeSyncManager) persistConfig() error {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()
	return writeJSONAtomic(filepath.Join(m.dir, "config.json"), config)
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
