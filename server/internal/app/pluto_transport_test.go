package app

import "testing"

func TestPlutoContextsSeparateUSBNetworkFromEthernet(t *testing.T) {
	contexts := parsePlutoContexts("Available contexts:\n 0: 192.168.2.1 (FISHBall-PlutoSky), serial=ARTAX [ip:pluto.local]\n 1: 192.168.4.1 (FISHBall-PlutoSky), serial=ARTAX [ip:pluto.local]\n 2: 0456:b673 (Pluto), serial=ARTAX [usb:1.11.5]\n")
	options := contexts["ARTAX"]
	if len(options) != 3 || options[0].URI != "ip:192.168.2.1" || options[1].URI != "ip:192.168.4.1" || options[2].URI != "usb:1.11.5" {
		t.Fatalf("wrong transport inventory: %+v", options)
	}
}
func TestPlutoSavedTransportDoesNotSilentlyFallback(t *testing.T) {
	d := SDRDevice{Connected: true, Transport: "usb:1.2.3", Transports: []PlutoTransport{{URI: "usb:1.2.3"}}}
	applyPlutoPreference(&d, "ip:192.168.4.1")
	if d.Connected || d.Transport != "ip:192.168.4.1" {
		t.Fatalf("silently fell back: %+v", d)
	}
}
