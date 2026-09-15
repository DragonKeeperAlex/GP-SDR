//go:build !linux

package app

import "testing"

func TestPiPowerHatIsAbsentOutsideLinux(t *testing.T) {
	status := readPiPowerHat()
	if status.Available || status.Error != "" {
		t.Fatalf("non-Linux builds must not probe Pi power hardware: %#v", status)
	}
}
