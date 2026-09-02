// Package mobilebridge embeds the actual GP-SDR receiver engine in Android.
// It does not require a desktop GP-SDR server.
package mobilebridge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"gpsdr.local/gpsdr/internal/app"
	"gpsdr.local/gpsdr/web"
)

// USBHost is implemented by Android's permission-aware USB service.
type USBHost interface {
	DevicesJSON() (string, error)
	Open(deviceID, specificationJSON string) (USBStream, error)
}

// USBStream transfers blocks, never individual samples, across JNI.
type USBStream interface {
	ReadBlock() ([]byte, error)
	Close() error
}

type usbHost struct{ host USBHost }

func (h usbHost) Devices() ([]app.SDRDevice, error) {
	data, err := h.host.DevicesJSON()
	if err != nil {
		return nil, fmt.Errorf("Android USB enumeration failed: %w", err)
	}
	var devices []app.SDRDevice
	err = json.Unmarshal([]byte(data), &devices)
	for i := range devices {
		devices[i].Driver = "Android USB"
	}
	return devices, err
}
func (h usbHost) Open(device app.SDRDevice, spec app.CaptureSpec) (io.ReadCloser, app.SampleFormat, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil, "", err
	}
	stream, err := h.host.Open(device.ID, string(data))
	if err != nil {
		return nil, "", err
	}
	if stream == nil {
		return nil, "", errors.New("Android returned no USB stream")
	}
	format := app.ComplexSigned8
	if device.Kind == "RTL-SDR" {
		format = app.ComplexUnsigned8
	}
	return &usbReader{source: stream}, format, nil
}

type usbReader struct {
	source   USBStream
	pending  []byte
	once     sync.Once
	closeErr error
}

func (r *usbReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		data, err := r.source.ReadBlock()
		if err != nil {
			return 0, err
		}
		if len(data) == 0 {
			return 0, io.EOF
		}
		r.pending = data
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
func (r *usbReader) Close() error {
	r.once.Do(func() { r.closeErr = r.source.Close() })
	return r.closeErr
}

type Engine struct {
	url     string
	runtime *app.Runtime
	server  *http.Server
	once    sync.Once
}

// Start creates an authenticated loopback-only endpoint in this process.
func Start(dataDirectory string, host USBHost) (*Engine, error) {
	if host == nil {
		return nil, errors.New("Android USB host is required")
	}
	app.SetPlatformReceiverHost(usbHost{host})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		listener.Close()
		return nil, err
	}
	token := hex.EncodeToString(secret)
	url := "http://" + listener.Addr().String() + "/?token=" + token
	runtime, err := app.NewRuntime(dataDirectory, url, false)
	if err != nil {
		listener.Close()
		return nil, err
	}
	api := app.NewServer(runtime, web.Files, "127.0.0.1", 0, token)
	server := &http.Server{Handler: api.Handler(), MaxHeaderBytes: 1 << 20}
	engine := &Engine{url: url, runtime: runtime, server: server}
	go server.Serve(listener)
	return engine, nil
}
func (e *Engine) URL() string     { return e.url }
func (e *Engine) RefreshDevices() { e.runtime.Refresh() }
func (e *Engine) Stop() {
	e.once.Do(func() {
		e.runtime.StopAllMapperJobs()
		e.runtime.StopCharacterization()
		e.runtime.Stop()
		app.ShutdownCaptureSessions()
		e.server.Close()
	})
}
