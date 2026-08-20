package app

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportChannelCSV turns an entire CHIRP-style CSV/TSV file into one channel
// bank. GP-SDR imports receive frequencies only; duplex offsets and transmit
// settings are intentionally ignored.
func (s *ProfileStore) ImportChannelCSV(filename string, data []byte) (ScanProfile, error) {
	return s.importChannelCSV(filename, "", data)
}

func (s *ProfileStore) importChannelCSV(filename, profileID string, data []byte) (ScanProfile, error) {
	if len(data) == 0 {
		return ScanProfile{}, errors.New("channel file is empty")
	}
	if len(data) > 5_000_000 {
		return ScanProfile{}, errors.New("channel file is larger than 5 MB")
	}
	name := strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	if name == "" || name == "." {
		name = "Imported channels"
	}
	channels, err := parseChannelCSV(data)
	if err != nil {
		return ScanProfile{}, err
	}
	target := name
	if strings.TrimSpace(profileID) == "" {
		profileID = NewID()
	}
	profile := ScanProfile{
		SchemaVersion: 1,
		ID:            profileID,
		Name:          truncateText(name, 80),
		Summary:       fmt.Sprintf("%d channels imported in one step", len(channels)),
		Ranges:        []ScanRange{},
		Channels:      channels,
		DeviceAssignments: []DeviceAssignment{{
			ID: NewID(), Role: "channelBank", Target: &target,
		}},
		P25Systems: []P25SystemConfig{},
		Settings:   defaultSettings(),
	}
	return s.Save(profile)
}

func (s *ProfileStore) ImportChannelDatabase(filename string, data []byte) ([]ScanProfile, error) {
	if len(data) == 0 {
		return nil, errors.New("channel file is empty")
	}
	if len(data) > 5_000_000 {
		return nil, errors.New("channel file is larger than 5 MB")
	}
	channels, err := parseChannelCSVWithLimit(data, 100_000)
	if err != nil {
		return nil, err
	}
	return s.saveChannelBanks(filename, filepath.ToSlash(filename), channels)
}

func (s *ProfileStore) saveChannelBanks(filename, stableKey string, channels []ChannelDefinition) ([]ScanProfile, error) {
	baseName := strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	if baseName == "" {
		baseName = "Local channels"
	}
	const bankSize = 4_000
	bankCount := (len(channels) + bankSize - 1) / bankSize
	savedProfiles := make([]ScanProfile, 0, bankCount)
	for bankIndex := 0; bankIndex < bankCount; bankIndex++ {
		start := bankIndex * bankSize
		end := start + bankSize
		if end > len(channels) {
			end = len(channels)
		}
		name := baseName
		if bankCount > 1 {
			name = fmt.Sprintf("%s · Bank %d of %d", baseName, bankIndex+1, bankCount)
		}
		target := name
		profile := ScanProfile{SchemaVersion: 1, ID: localDatabaseProfileID(stableKey, bankIndex), Name: truncateText(name, 80),
			Summary: fmt.Sprintf("Local database · %s · %s channels", stableKey, formatInteger(end-start)), Channels: append([]ChannelDefinition(nil), channels[start:end]...),
			Ranges: []ScanRange{}, DeviceAssignments: []DeviceAssignment{{ID: NewID(), Role: "channelBank", Target: &target}},
			P25Systems: []P25SystemConfig{}, Settings: defaultSettings()}
		saved, err := s.Save(profile)
		if err != nil {
			return savedProfiles, err
		}
		savedProfiles = append(savedProfiles, saved)
	}
	return savedProfiles, nil
}

func parseChannelCSV(data []byte) ([]ChannelDefinition, error) {
	return parseChannelCSVWithLimit(data, 5_000)
}

func parseChannelCSVWithLimit(data []byte, maximumChannels int) ([]ChannelDefinition, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	if bytes.Count(bytes.SplitN(data, []byte("\n"), 2)[0], []byte("\t")) > bytes.Count(bytes.SplitN(data, []byte("\n"), 2)[0], []byte(",")) {
		reader.Comma = '\t'
	}
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("channel file needs a header row")
	}
	columns := make(map[string]int, len(header))
	for index, value := range header {
		columns[normalizeColumn(value)] = index
	}
	frequencyIndex, ok := firstColumn(columns, "frequency", "frequencyhz", "freq", "rxfrequency", "receivefrequency", "frequencyoutput", "outputfrequency")
	if !ok {
		return nil, errors.New("channel file needs a Frequency column")
	}
	nameIndex, hasName := firstColumn(columns, "name", "alphatag", "channel", "description")
	descriptionIndex, hasDescription := firstColumn(columns, "description")
	agencyIndex, hasAgency := firstColumn(columns, "agencycategory", "agency", "category")
	modeIndex, hasMode := firstColumn(columns, "mode", "modulation")
	channels := make([]ChannelDefinition, 0, 256)
	seen := make(map[string]bool)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("invalid channel row %d: %w", rowNumber, readErr)
		}
		if frequencyIndex >= len(record) {
			continue
		}
		frequency, parseErr := parseImportedFrequency(record[frequencyIndex])
		if parseErr != nil || frequency <= 0 {
			continue
		}
		name := fmt.Sprintf("Channel %d", len(channels)+1)
		placeholderName := name
		if hasName && nameIndex < len(record) && strings.TrimSpace(record[nameIndex]) != "" {
			name = strings.TrimSpace(record[nameIndex])
		}
		description, agency := "", ""
		if hasDescription && descriptionIndex < len(record) {
			description = strings.TrimSpace(record[descriptionIndex])
		}
		if hasAgency && agencyIndex < len(record) {
			agency = strings.TrimSpace(record[agencyIndex])
		}
		displayName := name
		if description != "" && !strings.EqualFold(description, name) {
			displayName += " · " + description
		}
		if agency != "" && name == placeholderName {
			displayName = agency + " · " + description
		}
		mode := "auto"
		var decoder *string
		if hasMode && modeIndex < len(record) {
			rawMode := record[modeIndex]
			mode = normalizeImportedMode(rawMode)
			if protocol := normalizeImportedProtocol(rawMode); protocol != "" {
				decoder = &protocol
			}
		}
		key := strings.ToLower(strings.Join([]string{name, description, agency, recordValue(record, modeIndex, hasMode)}, "\x00")) + "\x00" + strconv.FormatFloat(frequency, 'f', 0, 64)
		if seen[key] {
			continue
		}
		seen[key] = true
		bandwidth := 12_500.0
		switch mode {
		case "wfm":
			bandwidth = 180_000
		case "nfm":
			bandwidth = 20_000
		case "am":
			bandwidth = 12_000
		}
		channels = append(channels, ChannelDefinition{ID: NewID(), Name: truncateText(strings.Trim(displayName, " ·"), 80), FrequencyHz: frequency,
			BandwidthHz: bandwidth, Mode: mode, Decoder: decoder, Enabled: true, Priority: 5})
		if len(channels) > maximumChannels {
			return nil, fmt.Errorf("channel file contains more than %s channels", formatInteger(maximumChannels))
		}
	}
	if len(channels) == 0 {
		return nil, errors.New("no valid receive frequencies were found")
	}
	return channels, nil
}

func recordValue(record []string, index int, available bool) string {
	if !available || index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func formatInteger(value int) string {
	text := strconv.Itoa(value)
	for index := len(text) - 3; index > 0; index -= 3 {
		text = text[:index] + "," + text[index:]
	}
	return text
}

func normalizeColumn(value string) string {
	value = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
	return strings.NewReplacer(" ", "", "_", "", "-", "", ".", "").Replace(value)
}

func firstColumn(columns map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if index, ok := columns[name]; ok {
			return index, true
		}
	}
	return 0, false
}

func parseImportedFrequency(value string) (float64, error) {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.NewReplacer("mhz", "", "hz", "", " ", "", "_", "").Replace(clean)
	frequency, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, err
	}
	if frequency < 1_000_000 {
		frequency *= 1_000_000
	}
	return frequency, nil
}

func normalizeImportedMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "am", "air":
		return "am"
	case "wfm", "widefm", "wide fm":
		return "wfm"
	case "fm", "nfm", "fmn", "fm-n", "narrowfm", "narrow fm":
		return "nfm"
	case "p25", "p25e", "digital", "dmr", "nxdn", "nxdn48", "nxdn96", "d-star", "dstar", "ysf", "m17":
		return "digital"
	default:
		return "auto"
	}
}

func normalizeImportedProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "p25", "p25e":
		return "P25"
	case "dmr":
		return "DMR"
	case "nxdn", "nxdn48", "nxdn96":
		return "NXDN"
	case "d-star", "dstar":
		return "D-STAR"
	case "ysf":
		return "YSF"
	case "m17":
		return "M17"
	}
	return ""
}

func truncateText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
