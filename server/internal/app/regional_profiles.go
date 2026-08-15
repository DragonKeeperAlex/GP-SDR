package app

func regionalConventionalProfiles() []ScanProfile {
	profile := fixedChannelProfile("1d93d45f-fd97-4f82-810e-64a3c72b57cc", "San Ramon · Verified Local", "Current public-source fire, CHP, interop, parks, public works, and amateur receive channels", "San Ramon verified")
	profile.Channels = []ChannelDefinition{
		channel("SRV Fire VHF Command 1", 153.995, 12_500, "nfm"),
		channel("SRV Fire Tac 22 · Mt Diablo", 460.575, 12_500, "nfm"),
		channel("SRV Fire Tac 23/24 · Highland Peak", 453.425, 12_500, "nfm"),
		channel("SRV Fire Tac 25", 458.3125, 12_500, "nfm"),
		channel("SRV Fire Tac 26", 458.7125, 12_500, "nfm"),
		channel("SRV Fire Tac 27", 458.075, 12_500, "nfm"),
		channel("CHP Maroon 2 Base · Contra Costa", 42.920, 20_000, "nfm"),
		channel("CHP Maroon 2 Mobile · Contra Costa", 42.740, 20_000, "nfm"),
		channel("CHP Maroon 2 CPVE Extender", 769.18125, 12_500, "p25"),
		channel("CHP Maroon 2 EV-20 Extender", 769.43125, 12_500, "p25"),
		channel("CHP Khaki Base · Dublin", 44.940, 20_000, "nfm"),
		channel("CHP Khaki Mobile · Dublin", 42.780, 20_000, "nfm"),
		channel("CHP Khaki CPVE Extender", 769.16875, 12_500, "p25"),
		channel("CHP Khaki EV-20 Extender", 769.41875, 12_500, "p25"),
		channel("CALAW 1", 154.920, 12_500, "nfm"),
		channel("CALCORD", 156.075, 12_500, "nfm"),
		channel("Cal OES Fire V1/V2", 154.160, 12_500, "nfm"),
		channel("Cal OES Fire V3/V4", 154.220, 12_500, "nfm"),
		channel("East Bay Parks Police Dispatch", 44.760, 20_000, "nfm"),
		channel("East Bay Parks Fire Dispatch", 44.640, 20_000, "nfm"),
		channel("East Bay Parks Fire/Law Tac", 44.960, 20_000, "nfm"),
		channel("East Bay Parks Fire Tac 5", 45.000, 20_000, "nfm"),
		channel("East Bay Parks Police Extender", 150.805, 12_500, "nfm"),
		channel("Pleasanton Parks and Rec", 153.005, 12_500, "nfm"),
		channel("Caltrans District 4 · Mt Diablo", 856.9875, 20_000, "nfm"),
		channel("CCRA Highland Peak Repeater", 145.410, 20_000, "nfm"),
	}
	return []ScanProfile{profile}
}

func regionalP25Profiles() []ScanProfile {
	settings := defaultSettings()
	settings.RevisitSeconds = 5
	return []ScanProfile{
		{
			SchemaVersion: 1,
			ID:            "9fb16c2f-1492-4dbc-879d-d4a066775e2a",
			Name:          "San Ramon · EBRCS P25",
			Summary:       "Local police, fire, EMS, interop, and public works with encrypted calls skipped",
			Ranges:        []ScanRange{},
			Channels:      []ChannelDefinition{},
			DeviceAssignments: []DeviceAssignment{
				{ID: NewID(), Role: "control", Target: ptr("EBRCS control")},
				{ID: NewID(), Role: "voice", Target: ptr("EBRCS voice")},
			},
			P25Systems: []P25SystemConfig{
				{
					ID: "ebrcs-alco-east", Name: "EBRCS · ALCO East", NAC: "0x1F2", WACN: "0xBEE00", SystemID: "0x1F1",
					ControlChannelsHz: mhzList(772.76875, 773.66875, 774.14375, 774.89375), Enabled: true,
					Talkgroups: []TalkgroupDefinition{
						tg(1001, "EB CALL", "T", false), tg(1002, "EB INT 1", "T", false), tg(1003, "EB INT 2", "T", false),
						tg(1027, "EB FIRE 1", "T", false), tg(1039, "EB EMS 1", "T", false), tg(1086, "EB MED CALL", "T", false),
						tg(2606, "Alameda Fire Dispatch 2", "T", false), tg(2607, "Alameda EMS Dispatch", "T", false), tg(2703, "ACF Tac 43 · Dublin/Livermore", "T", false),
						tg(5515, "San Ramon Regional Medical Center", "T", false),
						tg(7550, "SRPD Announcement", "T", false), tg(7555, "SRPD Channel 1", "T", false), tg(7556, "SRPD Dispatch", "TE", true),
						tg(7557, "SRPD Tactical 1", "T", false), tg(7558, "SRPD Investigations 1", "TE", true), tg(7559, "SRPD Investigations 2", "TE", true),
						tg(7560, "SRPD Admin", "T", false), tg(8658, "San Ramon Public Works Events", "T", false), tg(8659, "San Ramon Engineering", "T", false),
						tg(8663, "San Ramon Public Services", "T", false), tg(8666, "San Ramon Landscape", "T", false),
					},
				},
				{
					ID: "ebrcs-ccco-central", Name: "EBRCS · CCCO Central", NAC: "0x1F5", WACN: "0xBEE00", SystemID: "0x1F1",
					ControlChannelsHz: mhzList(773.90625, 774.18125, 774.45625, 774.73125), Enabled: true,
					Talkgroups: []TalkgroupDefinition{
						tg(919, "CHP Maroon 2 Patch", "T", false), tg(1007, "CCCO Law Interop 1", "T", false), tg(1008, "CCCO Law Interop 2", "T", false),
						tg(1026, "CCCO Fire Interop 1", "T", false), tg(1030, "CCCO EMS Interop 1", "T", false), tg(6086, "San Ramon Regional Medical Center", "T", false),
						tg(6400, "SRV Fire Announcement", "T", false), tg(6403, "SRV Fire Channel 2", "T", false), tg(6404, "SRV Fire Channel 3", "T", false),
						tg(6405, "SRV Fire Dispatch", "T", false), tg(6406, "SRV Fire Communications 1", "T", false), tg(6407, "SRV Fire Communications 2", "T", false),
						tg(6408, "SRV Fire Channel 4", "T", false), tg(6409, "SRV Fire Channel 5", "T", false), tg(6410, "SRV Fire Channel 6", "T", false),
						tg(6411, "SRV Fire Channel 7", "T", false), tg(6412, "SRV Fire Channel 8", "T", false), tg(6415, "SRV Fire Emergency", "T", false),
						tg(7712, "CCSO Central Dispatch", "TE", true), tg(7719, "CCSO OES 1", "T", false), tg(7720, "CCSO OES 2", "T", false),
						tg(7722, "CCSO Tactical Central", "T", false), tg(8906, "Danville Special Events", "T", false),
					},
				},
			},
			Settings: settings,
			BuiltIn:  true,
		},
		{
			SchemaVersion: 1,
			ID:            "d941f8a8-52d8-497c-a0dc-d3fe08935f6f",
			Name:          "BART · P25",
			Summary:       "Above-ground and underground BART P25 sites with per-talkgroup mixer controls",
			Ranges:        []ScanRange{},
			Channels:      []ChannelDefinition{},
			DeviceAssignments: []DeviceAssignment{
				{ID: NewID(), Role: "control", Target: ptr("BART control")},
				{ID: NewID(), Role: "voice", Target: ptr("BART voice")},
			},
			P25Systems: []P25SystemConfig{
				{
					ID: "bart-above-ground", Name: "BART · Above Ground", NAC: "0x028", WACN: "0x92762", SystemID: "0x338",
					ControlChannelsHz: mhzList(851.0375, 851.3125, 851.5625, 851.8875, 852.0375, 852.2375, 852.5625, 852.8125, 853.0375, 853.3625), Enabled: true,
					Talkgroups: bartTalkgroups(),
				},
				{
					ID: "bart-underground", Name: "BART · Underground", WACN: "0x92762", SystemID: "0x338",
					ControlChannelsHz: mhzList(851.6375, 852.1625, 852.3375, 852.6500, 853.6750, 853.8625), Enabled: true,
					Talkgroups: bartTalkgroups(),
				},
			},
			Settings: settings,
			BuiltIn:  true,
		},
	}
}

func mhzList(values ...float64) []float64 {
	result := make([]float64, len(values))
	for index, value := range values {
		result[index] = value * 1_000_000
	}
	return result
}

func tg(id int, name, mode string, encrypted bool) TalkgroupDefinition {
	return TalkgroupDefinition{ID: id, Name: name, Mode: mode, Encrypted: encrypted, Enabled: true}
}

func bartTalkgroups() []TalkgroupDefinition {
	return []TalkgroupDefinition{
		tg(1017, "BART Police Main", "T", false), tg(1018, "BART Police Dispatch", "TE", true), tg(1020, "BART Police Common", "T", false),
		tg(1021, "BART Police Tac 1", "T", false), tg(1022, "BART Police Tac 2", "T", false), tg(1023, "BART Police Tac 3", "T", false),
		tg(1024, "BART Police Tac 4", "T", false), tg(1025, "BART Police Tac 5", "T", false), tg(1026, "BART Police Tac 6", "T", false),
		tg(1273, "Train Home Emergency", "T", false), tg(1277, "Road A Line · Fremont", "T", false), tg(1278, "Road C Line · Pittsburg", "T", false),
		tg(1280, "Road L Line · Dublin/Pleasanton", "T", false), tg(1282, "Road E Line · Antioch", "T", false), tg(1291, "Dublin Terminal Zone", "T", false),
		tg(1388, "Incident Command 1", "T", false), tg(1389, "Incident Command 2", "T", false),
		tg(34000, "BART Calling", "D", false), tg(34001, "BART Interop 1", "D", false), tg(34002, "BART Interop 2", "D", false),
		tg(34003, "BART Interop 3", "D", false), tg(34004, "BART Interop 4", "D", false), tg(34005, "BART Interop 5", "D", false),
		tg(34006, "BART Fire Primary Below Ground", "D", false), tg(34007, "BART Fire Secondary Below Ground", "D", false),
	}
}
