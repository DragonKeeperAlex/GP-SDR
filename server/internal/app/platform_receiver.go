package app

import (
	"errors"
	"io"
	"sync"
)

// PlatformReceiverHost supplies permission-gated USB transports in embedded apps.
// Desktop capture remains unchanged when no host is registered.
type PlatformReceiverHost interface {
	Devices() ([]SDRDevice, error)
	Open(SDRDevice, CaptureSpec) (io.ReadCloser, SampleFormat, error)
}

var platformReceiver struct {
	sync.RWMutex
	host PlatformReceiverHost
}

func SetPlatformReceiverHost(host PlatformReceiverHost) {
	platformReceiver.Lock()
	defer platformReceiver.Unlock()
	platformReceiver.host = host
}

func platformHost() PlatformReceiverHost {
	platformReceiver.RLock()
	defer platformReceiver.RUnlock()
	return platformReceiver.host
}

func openPlatformReceiver(device SDRDevice, spec CaptureSpec) (*IQStream, error) {
	host := platformHost()
	if host == nil {
		return nil, errors.New("mobile USB receiver service is unavailable")
	}
	reader, format, err := host.Open(device, spec)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, errors.New("USB receiver returned no sample stream")
	}
	return &IQStream{Reader: reader, Format: format}, nil
}
