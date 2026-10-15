package app

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RequestSystemPower delegates only a clean host shutdown or reboot to the
// root-owned helper. The short delay lets the HTTP response reach the touch UI.
func RequestSystemPower(action string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "shutdown" && action != "reboot" {
		return fmt.Errorf("unsupported system power action")
	}
	go func() {
		time.Sleep(750 * time.Millisecond)
		_ = exec.Command("/usr/bin/sudo", "/usr/local/lib/gp-sdr/system-power", action).Run()
	}()
	return nil
}


// RequestDisplaySession switches only the local touchscreen session. GP-SDR
// continues running so receiver work and recordings are not interrupted.
func RequestDisplaySession(action string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "desktop" && action != "kiosk" {
		return fmt.Errorf("unsupported display session action")
	}
	go func() {
		time.Sleep(750 * time.Millisecond)
		_ = exec.Command("/usr/bin/sudo", "/usr/local/lib/gp-sdr/display-session", action).Run()
	}()
	return nil
}
