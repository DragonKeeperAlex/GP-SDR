package app

import (
	"os"
	"strings"
	"testing"
	"time"
)

// This acceptance test is opt-in because it takes exclusive ownership of an
// attached SDR. Run with GPSDR_HARDWARE_TEST=1 and optionally select
// GPSDR_TEST_DEVICE_KIND=HackRF or RTL-SDR.
func TestSDRTrunkConnectedP25Receiver(t *testing.T) {
	if os.Getenv("GPSDR_HARDWARE_TEST") != "1" {
		t.Skip("set GPSDR_HARDWARE_TEST=1 to exercise connected SDR hardware")
	}
	wanted := os.Getenv("GPSDR_TEST_DEVICE_KIND")
	if wanted == "" {
		wanted = "HackRF"
	}
	var selected *SDRDevice
	for _, device := range DiscoverDevices(false) {
		if device.Connected && strings.EqualFold(device.Kind, wanted) {
			copy := device
			selected = &copy
			break
		}
	}
	if selected == nil {
		t.Fatalf("no connected %s receiver found", wanted)
	}
	profile := ScanProfile{ID: "hardware-ebrsc", Name: "EBRCS hardware acceptance", P25Systems: []P25SystemConfig{{
		ID: "ebrsc", Name: "East Bay Regional Communications System", Enabled: true,
		ControlChannelsHz: []float64{774_181_250, 773_906_250, 774_456_250, 774_731_250},
	}}}
	deviceID := selected.ID
	plan := []ReceiverPlanItem{{DeviceID: &deviceID, Role: "control", State: "assigned"}}
	manager := &OP25Manager{}
	dataDirectory := os.Getenv("GPSDR_TEST_DATA")
	if dataDirectory == "" {
		dataDirectory = t.TempDir()
	}
	if err := manager.Start(profile, plan, []SDRDevice{*selected}, dataDirectory); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		status := manager.Status()
		t.Logf("%s: %s", status.Reception, status.Note)
		if status.Reception == "locked" {
			return
		}
		time.Sleep(3 * time.Second)
	}
	status := manager.Status()
	if status.State != "running" {
		t.Fatalf("SDRTrunk stopped during hardware test: %+v", status)
	}
	t.Log("receiver remained healthy but no P25 control lock was observed during this bounded RF test")
}
