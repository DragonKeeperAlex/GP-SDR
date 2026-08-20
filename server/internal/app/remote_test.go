package app

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestRTLTCPStreamHandshakeAndCommands(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := make(chan []byte, 1)
	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()
		header := make([]byte, 12)
		copy(header, "RTL0")
		binary.BigEndian.PutUint32(header[4:], 5)
		_, _ = conn.Write(header)
		data := make([]byte, 20)
		_, _ = io.ReadFull(conn, data)
		commands <- data
		_, _ = conn.Write([]byte{128, 128, 129, 127})
	}()
	address := listener.Addr().(*net.TCPAddr)
	device := SDRDevice{Kind: "RTL-TCP", Host: "127.0.0.1", Port: address.Port}
	stream, err := StartIQStream(device, CaptureSpec{CenterFrequencyHz: 100_100_000, SampleRateHz: 2_400_000, GainDB: 20})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	buffer := make([]byte, 4)
	if _, err = io.ReadFull(stream.Reader, buffer); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-commands:
		if data[0] != 1 || binary.BigEndian.Uint32(data[1:5]) != 100_100_000 || data[5] != 2 || binary.BigEndian.Uint32(data[6:10]) != 2_400_000 || data[10] != 3 || binary.BigEndian.Uint32(data[11:15]) != 1 || data[15] != 4 || binary.BigEndian.Uint32(data[16:20]) != 200 {
			t.Fatalf("unexpected commands: %v", data)
		}
	case <-time.After(time.Second):
		t.Fatal("no rtl_tcp commands")
	}
}
