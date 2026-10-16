package app

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOP25CallsExcludeControlEncryptedAndExpire(t *testing.T) {
	now := time.Now()
	decode := func(text string) []map[string]json.RawMessage {
		var v []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(text), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	updates := decode(`[{"json_type":"channel_update","0":{"freq":774456250,"tgid":null},"1":{"freq":772881250,"tgid":3255,"srcaddr":3176271,"system":"EBRCS","encrypted":0,"tdma":1},"2":{"freq":773156250,"tgid":7505,"encrypted":1}}]`)
	calls := mergeOP25Calls(nil, updates, now)
	if len(calls) != 1 || calls[1].Grant.GroupID != 3255 || calls[1].Grant.Protocol != "P25 Phase 2" {
		t.Fatalf("bad calls: %#v", calls)
	}
	next := mergeOP25Calls(calls, updates, now.Add(time.Second))
	if !next[1].StartedAt.Equal(now) {
		t.Fatal("same call lost stable start time")
	}
	if len(mergeOP25Calls(next, nil, now.Add(4*time.Second))) != 0 {
		t.Fatal("stale calls never expired")
	}
	if len(mergeOP25Calls(calls, decode(`[{"json_type":"channel_update","1":{"freq":774456250,"tgid":null}}]`), now)) != 0 {
		t.Fatal("control transition retained a voice call")
	}
}

func TestOP25CallIdentitySeparatesSlotSourceAndReceiver(t *testing.T) {
	now := time.Now()
	updates := sessionUpdates(t, `[{"json_type":"channel_update","0":{"freq":772881250,"tgid":3255,"srcaddr":55,"system":"site","tdma":0},"1":{"freq":772881250,"tgid":3255,"srcaddr":55,"system":"site","tdma":1}}]`)
	calls := mergeOP25Calls(nil, updates, now)
	if p25CallKey(calls[0]) == p25CallKey(calls[1]) {
		t.Fatal("overlapping receivers/slots collapsed")
	}
	switched := mergeOP25Calls(calls, sessionUpdates(t, `[{"json_type":"channel_update","0":{"freq":772881250,"tgid":3255,"srcaddr":56,"system":"site","tdma":1}}]`), now.Add(time.Second))
	if switched[0].StartedAt.Equal(calls[0].StartedAt) {
		t.Fatal("slot/source switch retained previous identity")
	}
	for _, tag := range []string{"Idle", "Control Channel"} {
		updates := sessionUpdates(t, `[{"json_type":"channel_update","0":{"freq":772881250,"tgid":3255,"tag":"`+tag+`"}}]`)
		if len(mergeOP25Calls(nil, updates, now)) != 0 {
			t.Fatal("non-voice role was promoted")
		}
	}
}
