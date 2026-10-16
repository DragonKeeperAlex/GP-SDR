package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// P25ReceiverStatus is a receiver-specific session contract. Configuration is
// captured from the exact OP25 config sent to the child, not from editable
// profile defaults. It is not hardware readback or RF acceptance evidence.
type P25ReceiverStatus struct {
	DeviceID          string     `json:"deviceID"`
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	Transport         string     `json:"transport,omitempty"`
	StreamIndex       int        `json:"streamIndex"`
	System            string     `json:"system"`
	ConfiguredRateHz  int        `json:"configuredRateHz"`
	UsableBandwidthHz int        `json:"usableBandwidthHz"`
	ConfiguredGains   string     `json:"configuredGains"`
	ConfiguredPPM     float64    `json:"configuredPPM"`
	SettingsSource    string     `json:"settingsSource"`
	Role              string     `json:"role"`
	State             string     `json:"state"`
	FrequencyHz       float64    `json:"frequencyHz,omitempty"`
	TalkgroupID       *uint32    `json:"talkgroupID,omitempty"`
	TDMASlot          *int       `json:"tdmaSlot,omitempty"`
	LastControlAt     *time.Time `json:"lastControlAt,omitempty"`
	Note              string     `json:"note"`
}

type op25SystemEvidence struct {
	System string  `json:"system"`
	Active int     `json:"network_active"`
	Freq   float64 `json:"rxchan"`
	Last   float64 `json:"last_tsbk"`
}

type op25ReceiverEvidence struct {
	System    string  `json:"system"`
	Tag       string  `json:"tag"`
	Freq      float64 `json:"freq"`
	Group     *uint32 `json:"tgid"`
	Slot      *int    `json:"tdma"`
	Encrypted int     `json:"encrypted"`
	HoldGroup uint32  `json:"hold_tgid"`
}

func op25ReceiverConfiguration(config op25Configuration, devices []SDRDevice) []P25ReceiverStatus {
	byName := make(map[string]int, len(config.Devices))
	for index, device := range config.Devices {
		byName[device.Name] = index
	}
	result := make([]P25ReceiverStatus, 0, len(config.Channels))
	for stream, channel := range config.Channels {
		index, ok := byName[channel.Device]
		if !ok || index >= len(devices) {
			continue
		}
		configDevice, device := config.Devices[index], devices[index]
		result = append(result, P25ReceiverStatus{
			DeviceID: device.ID, Name: device.Name, Kind: device.Kind, Transport: device.Transport,
			StreamIndex: stream, System: channel.TrunkingSystem,
			ConfiguredRateHz:  configDevice.Rate,
			UsableBandwidthHz: int(float64(configDevice.Rate) * configDevice.UsablePercent),
			ConfiguredGains:   configDevice.Gains, ConfiguredPPM: configDevice.PPM,
			SettingsSource: "decoder_configuration", Role: "unknown", State: "starting",
			Note: "Waiting for receiver telemetry.",
		})
	}
	return result
}

// projectOP25Session is pure: a coherent decoder reply plus the immutable
// session configuration produces one status snapshot. Never attribute another
// receiver's control lock to every radio, or promote a held talkgroup to voice.
func projectOP25Session(configured []P25ReceiverStatus, updates []map[string]json.RawMessage, started, now time.Time) []P25ReceiverStatus {
	systems := make(map[string]op25SystemEvidence)
	channels := make(map[int]op25ReceiverEvidence)
	for _, update := range updates {
		var kind string
		_ = json.Unmarshal(update["json_type"], &kind)
		for key, raw := range update {
			index, err := strconv.Atoi(key)
			if err != nil {
				continue
			}
			switch kind {
			case "trunk_update":
				var system op25SystemEvidence
				if json.Unmarshal(raw, &system) == nil && system.System != "" {
					systems[system.System] = system
				}
			case "channel_update":
				var channel op25ReceiverEvidence
				if json.Unmarshal(raw, &channel) == nil {
					channels[index] = channel
				}
			}
		}
	}
	result := append([]P25ReceiverStatus(nil), configured...)
	for index := range result {
		receiver := &result[index]
		receiver.Role, receiver.State = "unknown", "telemetry_unavailable"
		receiver.FrequencyHz, receiver.TalkgroupID, receiver.TDMASlot, receiver.LastControlAt = 0, nil, nil, nil
		receiver.Note = "Decoder telemetry unavailable; check decoder health."
		channel, ok := channels[receiver.StreamIndex]
		if !ok || channel.System != receiver.System {
			continue
		}
		receiver.FrequencyHz = channel.Freq
		switch {
		case channel.Tag == "Control Channel":
			receiver.Role, receiver.State = "control", "searching"
			receiver.Note = "Searching control candidates; no recent decoded control messages. Check site, antenna, gain and frequency correction."
			system := systems[receiver.System]
			last := time.Unix(0, int64(system.Last*1e9))
			if system.Active == 1 && system.Freq > 0 && channel.Freq == system.Freq && system.Last > 0 && !last.Before(started) && !last.After(now) && now.Sub(last) <= 10*time.Second {
				receiver.State, receiver.LastControlAt = "control_locked", &last
				receiver.Note = "Recent control messages decoded by this receiver."
			}
		case channel.Tag == "Idle":
			receiver.Role, receiver.State = "idle", "idle"
			receiver.Note = "Available to the decoder scheduler; no call assigned."
		case channel.Group != nil && *channel.Group > 0 && channel.HoldGroup == 0 && channel.Freq > 0:
			receiver.Role, receiver.State = "voice", "following"
			receiver.TalkgroupID, receiver.TDMASlot = channel.Group, channel.Slot
			receiver.Note = "Following a voice assignment; decoded audio is separate evidence."
			if channel.Encrypted != 0 {
				receiver.State, receiver.Note = "encrypted", "Encrypted assignment; excluded from recording and transcription."
			}
		default:
			receiver.State, receiver.Note = "unconfirmed", "Receiver telemetry has no confirmed control or voice role."
		}
	}
	return result
}

func summarizeOP25Session(status *P25Status, configured []P25ReceiverStatus, updates []map[string]json.RawMessage, started, now time.Time) {
	status.ObservedAt = &now
	if !started.IsZero() {
		status.StartedAt = &started
	}
	status.Receivers = projectOP25Session(configured, updates, started, now)
	status.TelemetryState = "available"
	locked, missing := 0, 0
	for _, receiver := range status.Receivers {
		if receiver.State == "telemetry_unavailable" {
			missing++
		}
		if receiver.State == "control_locked" {
			locked++
			if status.ControlChannelHz == 0 {
				status.ControlChannelHz = receiver.FrequencyHz
			}
		}
	}
	if len(updates) == 0 || missing > 0 {
		status.TelemetryState = "unavailable"
	}
	if missing > 0 && missing < len(status.Receivers) {
		status.TelemetryState = "partial"
	}
	if locked > 0 {
		status.Reception, status.ControlSource = "locked", "OP25 decoded control messages"
		status.Note = "Recent P25 control messages decoded; inspect receiver roles for call assignments."
	} else if status.TelemetryState != "available" {
		status.Reception = "unknown"
		status.Note = "Decoder telemetry is incomplete or unavailable; reception cannot be confirmed."
	}
	if len(status.Receivers) > 0 {
		// Legacy clients must not label mixed devices using the first one's rate.
		status.CaptureRateHz = status.Receivers[0].ConfiguredRateHz
		parts := make([]string, 0, len(status.Receivers))
		for _, receiver := range status.Receivers {
			if receiver.ConfiguredRateHz != status.CaptureRateHz {
				status.CaptureRateHz = 0
			}
			parts = append(parts, fmt.Sprintf("%s: %s", receiver.Name, receiver.ConfiguredGains))
		}
		status.FrontendState = strings.Join(parts, " · ")
	}
}
