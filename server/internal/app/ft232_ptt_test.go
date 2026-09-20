package app

import "testing"

func TestValidateFT232Device(t *testing.T) {
	if err := ValidateFT232Device("/dev/ttyUSB0"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFT232Device(""); err == nil {
		t.Fatal("expected missing device error")
	}
}
