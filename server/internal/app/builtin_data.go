package app

import (
	"embed"
	"fmt"
)

//go:embed data/*.csv
var builtInChannelFiles embed.FS

func mustBuiltInChannelProfile(path, id, name, summary, target string) ScanProfile {
	data, err := builtInChannelFiles.ReadFile(path)
	if err != nil {
		panic(err)
	}
	channels, err := parseChannelCSV(data)
	if err != nil {
		panic(fmt.Errorf("built-in channel data %s: %w", path, err))
	}
	return ScanProfile{
		SchemaVersion: 1,
		ID:            id,
		Name:          name,
		Summary:       summary,
		Ranges:        []ScanRange{},
		Channels:      channels,
		DeviceAssignments: []DeviceAssignment{{
			ID: NewID(), Role: "channelBank", Target: &target,
		}},
		P25Systems: []P25SystemConfig{},
		Settings:   defaultSettings(),
		BuiltIn:    true,
	}
}

func handheldProfiles() []ScanProfile {
	return []ScanProfile{
		mustBuiltInChannelProfile("data/san_ramon_scanner.csv", "93d9d500-ea6f-4d08-b72d-72173e1ca48d", "San Ramon · Handheld Bank", "137-channel local CHP, fire, air, marine, rail, and interop bank", "San Ramon handheld"),
		mustBuiltInChannelProfile("data/norcal_travel.csv", "5927c125-6655-46d6-98ac-9c70fb833f18", "NorCal · Travel Bank", "199-memory Northern California travel and monitoring bank", "NorCal travel"),
		mustBuiltInChannelProfile("data/rubicon_travel.csv", "942434c1-5bea-4b24-bd65-a1d6e858bda0", "NorCal · Rubicon Bank", "122-memory GMRS, trail, interop, weather, and travel bank", "Rubicon travel"),
		mustBuiltInChannelProfile("data/california_gmrs_repeaters.csv", "776a2f54-738d-4550-bad3-610299c3dbd8", "California · GMRS Repeaters", "All 84 California repeater outputs from the local programming archive", "California GMRS repeaters"),
	}
}
