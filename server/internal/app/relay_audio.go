package app

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

type RelayAudioConfig struct {
	Device     string `json:"device"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

func (c RelayAudioConfig) normalized() RelayAudioConfig {
	if c.Device == "" {
		c.Device = "plughw:2,0"
	}
	if c.SampleRate <= 0 {
		c.SampleRate = 8000
	}
	if c.Channels <= 0 {
		c.Channels = 1
	}
	return c
}

func (c RelayAudioConfig) AplayArgs() []string {
	c = c.normalized()
	return []string{"-D", c.Device, "-q", "-t", "raw", "-f", "S16_LE", "-r", strconv.Itoa(c.SampleRate), "-c", strconv.Itoa(c.Channels)}
}

type RelayAudioOutput struct {
	config RelayAudioConfig
	start  func(context.Context, string, ...string) *exec.Cmd
	cmd    *exec.Cmd
	input  io.WriteCloser
}

func NewRelayAudioOutput(config RelayAudioConfig) *RelayAudioOutput {
	return &RelayAudioOutput{config: config.normalized(), start: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, args...)
	}}
}

func (o *RelayAudioOutput) Start(ctx context.Context) error {
	if o.cmd != nil {
		return fmt.Errorf("relay audio output is already running")
	}
	o.cmd = o.start(ctx, "aplay", o.config.AplayArgs()...)
	input, err := o.cmd.StdinPipe()
	if err != nil {
		o.cmd = nil
		return fmt.Errorf("open relay audio input: %w", err)
	}
	if err := o.cmd.Start(); err != nil {
		_ = input.Close()
		o.cmd = nil
		return fmt.Errorf("start relay audio output: %w", err)
	}
	o.input = input
	return nil
}

func (o *RelayAudioOutput) Write(samples []int16) error {
	if o.input == nil {
		return fmt.Errorf("relay audio output is not running")
	}
	buf := make([]byte, len(samples)*2)
	for i, sample := range samples {
		buf[i*2] = byte(sample)
		buf[i*2+1] = byte(sample >> 8)
	}
	_, err := o.input.Write(buf)
	return err
}

func (o *RelayAudioOutput) Stop() error {
	if o.cmd == nil {
		return nil
	}
	_ = o.input.Close()
	err := o.cmd.Wait()
	o.cmd, o.input = nil, nil
	return err
}
