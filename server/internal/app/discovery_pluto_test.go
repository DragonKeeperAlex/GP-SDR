package app

import "testing"

func TestPreferredPlutoURIOnlyAcceptsNetworkTransport(t *testing.T) {
	t.Setenv("GPSDR_PLUTO_URI", "ip:192.168.2.1")
	if got := preferredPlutoURI(); got != "ip:192.168.2.1" {
		t.Fatalf("preferredPlutoURI() = %q", got)
	}
	t.Setenv("GPSDR_PLUTO_URI", "usb:1.59.5")
	if got := preferredPlutoURI(); got != "" {
		t.Fatalf("USB URI must not override discovery, got %q", got)
	}
}
