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
	profile := ScanProfile{
		SchemaVersion: 1,
		ID:            NewID(),
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

func parseChannelCSV(data []byte) ([]ChannelDefinition, error) {
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
	frequencyIndex, ok := firstColumn(columns, "frequency", "frequencyhz", "freq", "rxfrequency", "receivefrequency")
	if !ok {
		return nil, errors.New("channel file needs a Frequency column")
	}
	nameIndex, _ := firstColumn(columns, "name", "alphatag", "channel", "description")
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
		if nameIndex < len(record) && strings.TrimSpace(record[nameIndex]) != "" {
			name = strings.TrimSpace(record[nameIndex])
		}
		mode := "auto"
		if hasMode && modeIndex < len(record) {
			mode = normalizeImportedMode(record[modeIndex])
		}
		key := strings.ToLower(name) + "\x00" + strconv.FormatFloat(frequency, 'f', 0, 64)
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
		channels = append(channels, ChannelDefinition{ID: NewID(), Name: truncateText(name, 80), FrequencyHz: frequency,
			BandwidthHz: bandwidth, Mode: mode, Enabled: true, Priority: 5})
		if len(channels) > 5_000 {
			return nil, errors.New("channel file contains more than 5,000 channels")
		}
	}
	if len(channels) == 0 {
		return nil, errors.New("no valid receive frequencies were found")
	}
	return channels, nil
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
	case "fm", "nfm", "fm-n", "narrowfm", "narrow fm":
		return "nfm"
	case "p25":
		return "p25"
	case "digital", "dmr", "nxdn", "d-star", "dstar", "ysf", "m17":
		return "digital"
	default:
		return "auto"
	}
}

func truncateText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
