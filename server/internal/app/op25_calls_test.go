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
