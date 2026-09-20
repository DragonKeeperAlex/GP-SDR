package app

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FT232PTT drives the existing repeater-compatible FT232 RTS line. The helper
// deliberately creates the serial object unopened, drives RTS/DTR low, and
// only then opens the device; pySerial otherwise asserts RTS during open.
type FT232PTT struct {
	Device string
	Python string
}

func (p FT232PTT) SetPTT(asserted bool) error {
	device := p.Device
	if strings.TrimSpace(device) == "" {
		device = "/dev/ttyUSB0"
	}
	python := p.Python
	if strings.TrimSpace(python) == "" {
		python = "python3"
	}
	script := `import serial, sys
p=serial.Serial(port=None, baudrate=9600, timeout=0, rtscts=False, dsrdtr=False)
p.rts=False; p.dtr=False; p.port=sys.argv[1]; p.open()
p.rts=(sys.argv[2] == "1")
if not p.rts: p.dtr=False
p.close()
`
	value := "0"
	if asserted {
		value = "1"
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-c", script, device, value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("FT232 RTS %s: %w (%s)", strconv.FormatBool(asserted), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func ValidateFT232Device(device string) error {
	if strings.TrimSpace(device) == "" {
		return errors.New("FT232 device path is required")
	}
	return nil
}
