package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FixtureLibrary struct {
	Fixtures  []SignalFixtureManifest `json:"fixtures"`
	Invalid   int                     `json:"invalid"`
	Truncated bool                    `json:"truncated"`
}

// FixtureLibrary lists bounded manifest metadata only: it never loads IQ,
// promotes a sample to verified, starts a receiver, or trains a model.
func (r *Runtime) FixtureLibrary() (FixtureLibrary, error) {
	result := FixtureLibrary{Fixtures: []SignalFixtureManifest{}}
	directory := filepath.Join(r.dataDirectory, "Transmit", "fixtures")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	inspected := 0
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".truth.json") {
			continue
		}
		inspected++
		if inspected > 2000 {
			result.Truncated = true
			break
		}
		if len(result.Fixtures) >= 200 {
			result.Truncated = true
			break
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1<<20 {
			result.Invalid++
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		var fixture SignalFixtureManifest
		if err != nil || json.Unmarshal(data, &fixture) != nil || fixture.ID == "" || fixture.SHA256 == "" {
			result.Invalid++
			continue
		}
		result.Fixtures = append(result.Fixtures, fixture)
	}
	sort.Slice(result.Fixtures, func(i, j int) bool { return result.Fixtures[i].CreatedAt.After(result.Fixtures[j].CreatedAt) })
	return result, nil
}
