package app

import (
	"os"
	"strconv"
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
	devices := DiscoverDevices(false)
	for _, device := range devices {
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
		ControlChannelsHz: []float64{774_456_250, 773_906_250, 774_181_250, 774_731_250},
	}}}
	deviceID := selected.ID
	plan := []ReceiverPlanItem{{DeviceID: &deviceID, Role: "control", State: "assigned"}}
	if frequencies := os.Getenv("GPSDR_TEST_CONTROL_HZ"); frequencies != "" {
		profile.P25Systems[0].ControlChannelsHz = nil
		for _, value := range strings.Split(frequencies, ",") {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				t.Fatal(err)
			}
			profile.P25Systems[0].ControlChannelsHz = append(profile.P25Systems[0].ControlChannelsHz, f)
		}
	}
	// Mute all talkgroups during unattended hardware tests, retaining decode logs.
	manager := &OP25Manager{muted: map[uint32]bool{0: true}}
	dataDirectory := os.Getenv("GPSDR_TEST_DATA")
	if dataDirectory == "" {
		dataDirectory = t.TempDir()
	}
	if err := manager.Start(profile, plan, devices, dataDirectory); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()
	deadline := time.Now().Add(90 * time.Second)
	lockedAt := time.Time{}
	observeSeconds, _ := strconv.Atoi(os.Getenv("GPSDR_TEST_LOCK_HOLD_SECONDS"))
	for time.Now().Before(deadline) {
		status := manager.Status()
		t.Logf("%s: %s", status.Reception, status.Note)
		if status.Reception == "locked" {
			if lockedAt.IsZero() {
				lockedAt = time.Now()
			}
			if time.Since(lockedAt) >= time.Duration(observeSeconds)*time.Second {
				return
			}
		}
		time.Sleep(3 * time.Second)
	}
	status := manager.Status()
	if status.State != "running" {
		t.Fatalf("SDRTrunk stopped during hardware test: %+v", status)
	}
	t.Fatalf("no P25 control lock observed; startup alone is not acceptance: %+v", status)
}
