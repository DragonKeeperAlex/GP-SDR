//go:build linux

package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var piPowerHatCache struct {
	sync.Mutex
	status     PiPowerHatStatus
	checkedAt  time.Time
	refreshing bool
}

func readPiPowerHat() PiPowerHatStatus {
	piPowerHatCache.Lock()
	cacheFor := 30 * time.Second
	if piPowerHatCache.status.Available {
		cacheFor = 3 * time.Second
	}
	if time.Since(piPowerHatCache.checkedAt) >= cacheFor && !piPowerHatCache.refreshing {
		piPowerHatCache.refreshing = true
		go func() {
			status := queryPiPowerHat()
			piPowerHatCache.Lock()
			piPowerHatCache.status = status
			piPowerHatCache.checkedAt = time.Now()
			piPowerHatCache.refreshing = false
			piPowerHatCache.Unlock()
		}()
	}
	status := piPowerHatCache.status
	piPowerHatCache.Unlock()
	return status
}

func queryPiPowerHat() PiPowerHatStatus {
	python := "/opt/pipower5/venv/bin/python3"
	if _, err := os.Stat(python); err != nil {
		var lookupErr error
		python, lookupErr = exec.LookPath("python3")
		if lookupErr != nil {
			return PiPowerHatStatus{}
		}
	}
	const script = `import json
try:
 from pipower5.pipower5 import PiPower5
except (ImportError, ModuleNotFoundError):
 raise SystemExit(3)
d=PiPower5().read_all()
def n(k): return float(d.get(k,0) or 0)
source="Battery" if int(n("power_source")) == 1 else "External"
print(json.dumps({"available":True,"inputVoltage":n("input_voltage")/1000,"inputCurrent":n("input_current")/1000,"inputPower":n("input_voltage")*n("input_current")/1000000,"outputVoltage":n("output_voltage")/1000,"outputCurrent":n("output_current")/1000,"outputPower":n("output_voltage")*n("output_current")/1000000,"batteryVoltage":n("battery_voltage")/1000,"batteryCurrent":n("battery_current")/1000,"batteryPower":n("battery_voltage")*n("battery_current")/1000000,"batteryPercentage":n("battery_percentage"),"powerSource":source,"inputPluggedIn":bool(d.get("is_input_plugged_in",False)),"charging":bool(d.get("is_charging",False))}))`
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	data, err := exec.CommandContext(ctx, python, "-c", script).Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return PiPowerHatStatus{Error: "PiPower5 telemetry timed out"}
	}
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 3 {
			return PiPowerHatStatus{}
		}
		return PiPowerHatStatus{Error: "PiPower5 telemetry unavailable"}
	}
	var status PiPowerHatStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return PiPowerHatStatus{Error: "PiPower5 returned invalid telemetry"}
	}
	status.PowerSource = strings.TrimSpace(status.PowerSource)
	status.UpdatedAt = time.Now()
	return status
}
