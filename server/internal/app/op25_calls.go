package app

import (
	"encoding/json"
	"strconv"
	"time"
)

// Control-channel metadata is not a voice call. Only active, unencrypted
// channel_update entries are promoted, and stale transport state expires.
func (m *OP25Manager) readOP25Calls() []P25ActiveCall {
	updates := readOP25Updates()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine != "OP25" {
		return nil
	}
	m.op25Calls = mergeOP25Calls(m.op25Calls, updates, time.Now())
	result := make([]P25ActiveCall, 0, len(m.op25Calls))
	for _, call := range m.op25Calls {
		result = append(result, call)
	}
	return result
}

func mergeOP25Calls(previous map[int]P25ActiveCall, updates []map[string]json.RawMessage, now time.Time) map[int]P25ActiveCall {
	result := make(map[int]P25ActiveCall)
	for index, call := range previous {
		if now.Sub(call.LastHeardAt) < 2*time.Second {
			result[index] = call
		}
	}
	for _, update := range updates {
		var kind string
		_ = json.Unmarshal(update["json_type"], &kind)
		if kind != "channel_update" {
			continue
		}
		for key, raw := range update {
			index, err := strconv.Atoi(key)
			if err != nil {
				continue
			}
			var channel struct {
				Frequency float64 `json:"freq"`
				Group     *uint32 `json:"tgid"`
				Source    uint32  `json:"srcaddr"`
				System    string  `json:"system"`
				Tag       string  `json:"tag"`
				Encrypted int     `json:"encrypted"`
				Emergency int     `json:"emergency"`
				Slot      *int    `json:"tdma"`
			}
			if json.Unmarshal(raw, &channel) != nil {
				continue
			}
			if channel.Group == nil || *channel.Group == 0 || channel.Encrypted != 0 || channel.Frequency <= 0 {
				delete(result, index)
				continue
			}
			call := P25ActiveCall{Grant: p25GrantState{System: channel.System, Protocol: "P25 Phase 1", GroupID: *channel.Group, SourceID: channel.Source, FrequencyHz: uint32(channel.Frequency), Emergency: channel.Emergency != 0}, StartedAt: now, LastHeardAt: now, Following: true}
			if channel.Slot != nil {
				call.Grant.Protocol = "P25 Phase 2"
			}
			if old, ok := previous[index]; ok && old.Grant.GroupID == call.Grant.GroupID && old.Grant.System == call.Grant.System && old.Grant.FrequencyHz == call.Grant.FrequencyHz && now.Sub(old.LastHeardAt) < 2*time.Second {
				call.StartedAt = old.StartedAt
			}
			call.Talkgroup = &P25TalkgroupState{ID: *channel.Group, AlphaTag: channel.Tag, Scan: true, Discovered: true}
			result[index] = call
		}
	}
	return result
}
