package app

import (
	"os"
	"testing"
	"time"
)

func TestP25FinalizationRetainsPCMAndAttributionOnStop(t *testing.T) {
	root := t.TempDir()
	events, err := NewEventStore(root)
	if err != nil {
		t.Fatal(err)
	}
	r := &Runtime{Events: events, dataDirectory: root}
	archive := newP25CallArchive(2)
	archive.Append(AudioFrame{ChannelID: archive.streamID, SampleRate: 8000, Samples: make([]int16, 8000)})
	call := P25ActiveCall{StreamIndex: 2, DeviceSerial: "rtl", TDMASlot: ptr(1), StartedAt: time.Now(), Grant: p25GrantState{System: "site", GroupID: 3255, FrequencyHz: 772881250}}
	stop := make(chan struct{})
	close(stop)
	r.finishP25Call(stop, ScanProfile{}, call, archive)
	all := events.Recent(10)
	if len(all) != 1 || all[0].DurationSeconds != 1 || all[0].DeviceID != "rtl" || *all[0].P25TDMASlot != 1 || *all[0].P25StreamIndex != 2 || all[0].AudioPath == nil {
		t.Fatalf("bad finalized event: %+v", all)
	}
	if _, err := os.Stat(*all[0].AudioPath); err != nil {
		t.Fatal(err)
	}
	call.Grant.Encrypted = true
	r.finishP25Call(stop, ScanProfile{}, call, archive)
	if events.Recent(1)[0].AudioPath != nil {
		t.Fatal("encrypted audio archived")
	}
}
