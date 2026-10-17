package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func relayPulseEnvironment() []string {
	env := os.Environ()
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		env = append(env, "XDG_RUNTIME_DIR=/run/user/"+strconv.Itoa(os.Getuid()))
	}
	if os.Getenv("PULSE_SERVER") == "" {
		env = append(env, "PULSE_SERVER=unix:/run/user/"+strconv.Itoa(os.Getuid())+"/pulse/native")
	}
	return env
}

func (r *RelayRuntime) SetSpeakerMuted(muted bool) (RelayStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	value := "0"
	if muted {
		value = "1"
	}
	cmd := exec.CommandContext(ctx, "pactl", "set-sink-mute", "@DEFAULT_SINK@", value)
	cmd.Env = relayPulseEnvironment()
	if output, err := cmd.CombinedOutput(); err != nil {
		return r.Status(), fmt.Errorf("set system speaker mute: %w: %s", err, strings.TrimSpace(string(output)))
	}
	r.mu.Lock()
	r.speakerMuted = muted
	r.mu.Unlock()
	return r.Status(), nil
}

func systemAudioMonitor(ctx context.Context) (string, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, "pactl", "get-default-sink")
	cmd.Env = relayPulseEnvironment()
	data, err := cmd.Output()
	if err != nil {
		return "", errors.New("system audio capture is unavailable; PipeWire/PulseAudio must be running for this user")
	}
	sink := strings.TrimSpace(string(data))
	if sink == "" || strings.ContainsAny(sink, "\n\r ") {
		return "", errors.New("system audio output is unavailable")
	}
	if strings.Contains(strings.ToLower(sink), "usb-") {
		return "", errors.New("select the touchscreen/HDMI system output before using system-audio relay; USB audio is reserved for the radio")
	}
	return sink + ".monitor", nil
}

func startSystemAudioCapture(ctx context.Context, rate int, onError func(error)) (<-chan AudioFrame, func(), error) {
	monitor, err := systemAudioMonitor(ctx)
	if err != nil {
		return nil, nil, err
	}
	captureCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(captureCtx, "parec", "--device="+monitor, "--raw", "--format=s16le", "--rate="+strconv.Itoa(rate), "--channels=1", "--latency-msec=20")
	cmd.Env = relayPulseEnvironment()
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if err = cmd.Start(); err != nil {
		cancel()
		return nil, nil, err
	}
	frames := make(chan AudioFrame, 64)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(frames)
		buf := make([]byte, rate/50*2)
		for {
			_, readErr := io.ReadFull(pipe, buf)
			if readErr != nil {
				if captureCtx.Err() == nil {
					onError(fmt.Errorf("system audio capture ended: %w", readErr))
				}
				break
			}
			samples := make([]int16, len(buf)/2)
			for i := range samples {
				samples[i] = int16(binary.LittleEndian.Uint16(buf[i*2:]))
			}
			frame := AudioFrame{ChannelID: "system-audio", SampleRate: rate, Samples: samples}
			select {
			case frames <- frame:
			case <-captureCtx.Done():
				cancel()
				_ = cmd.Wait()
				return
			}
		}
		cancel()
		_ = cmd.Wait()
	}()
	return frames, func() { cancel(); <-done }, nil
}
