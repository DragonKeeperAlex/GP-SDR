package app

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SampleFormat string

const (
	ComplexSigned8   SampleFormat = "CS8"
	ComplexUnsigned8 SampleFormat = "CU8"
)

type CaptureSpec struct {
	CenterFrequencyHz int64
	SampleRateHz      int
	GainDB            float64
	PPMCorrection     int
	LNAGainDB         int
	VGAGainDB         int
	AmpEnabled        bool
	AntennaPower      bool
	AutoGain          bool
}

type CaptureCommand struct {
	Executable string
	Arguments  []string
	Format     SampleFormat
}

// BuildCaptureCommand translates a receiver assignment into a raw interleaved
// IQ byte stream. Decoder processes never open the SDR directly; the receiver
// manager owns the one capture process for each physical device.
func BuildCaptureCommand(device SDRDevice, spec CaptureSpec) (CaptureCommand, error) {
	if spec.CenterFrequencyHz <= 0 || spec.SampleRateHz <= 0 {
		return CaptureCommand{}, errors.New("center frequency and sample rate must be positive")
	}
	if strings.HasPrefix(device.Driver, "SoapySDR:") {
		tool, err := findTool("gpsdr-soapy")
		if err != nil {
			return CaptureCommand{}, errors.New("GP-SDR's SoapySDR stream helper is not installed")
		}
		args := []string{"--device", soapyDeviceArguments(device), "--frequency", strconv.FormatInt(spec.CenterFrequencyHz, 10), "--rate", strconv.Itoa(spec.SampleRateHz)}
		if spec.GainDB > 0 && !spec.AutoGain {
			args = append(args, "--gain", fmt.Sprintf("%.1f", spec.GainDB))
		}
		return CaptureCommand{Executable: tool, Arguments: args, Format: ComplexSigned8}, nil
	}
	switch device.Kind {
	case "HackRF":
		tool, err := findTool("hackrf_transfer")
		if err != nil {
			return CaptureCommand{}, errors.New("hackrf_transfer is not installed")
		}
		if spec.SampleRateHz < 2_000_000 || spec.SampleRateHz > 20_000_000 {
			return CaptureCommand{}, errors.New("HackRF sample rate must be between 2 and 20 Msps")
		}
		args := []string{"-r", "-", "-f", strconv.FormatInt(spec.CenterFrequencyHz, 10), "-s", strconv.Itoa(spec.SampleRateHz)}
		if device.Serial != nil && *device.Serial != "" {
			args = append(args, "-d", *device.Serial)
		}
		vga := spec.VGAGainDB
		if vga == 0 && spec.GainDB > 0 {
			vga = int(spec.GainDB/2) * 2
		}
		if vga > 0 {
			if vga > 62 {
				vga = 62
			}
			args = append(args, "-g", strconv.Itoa(vga))
		}
		if spec.LNAGainDB > 0 {
			lna := (spec.LNAGainDB / 8) * 8
			if lna > 40 {
				lna = 40
			}
			args = append(args, "-l", strconv.Itoa(lna))
		}
		if spec.AmpEnabled {
			args = append(args, "-a", "1")
		}
		if spec.AntennaPower {
			args = append(args, "-p", "1")
		}
		if spec.PPMCorrection != 0 {
			args = append(args, "-C", strconv.Itoa(spec.PPMCorrection))
		}
		return CaptureCommand{Executable: tool, Arguments: args, Format: ComplexSigned8}, nil
	case "RTL-SDR":
		tool, err := findTool("rtl_sdr")
		if err != nil {
			return CaptureCommand{}, errors.New("rtl_sdr is not installed")
		}
		if spec.SampleRateHz < 225_000 || spec.SampleRateHz > 3_200_000 {
			return CaptureCommand{}, errors.New("RTL-SDR sample rate must be between 225 ksps and 3.2 Msps")
		}
		args := []string{"-f", strconv.FormatInt(spec.CenterFrequencyHz, 10), "-s", strconv.Itoa(spec.SampleRateHz)}
		if pieces := strings.Split(device.ID, "-"); len(pieces) > 1 {
			if _, err := strconv.Atoi(pieces[len(pieces)-1]); err == nil {
				args = append(args, "-d", pieces[len(pieces)-1])
			}
		}
		if spec.GainDB > 0 && !spec.AutoGain {
			args = append(args, "-g", fmt.Sprintf("%.1f", spec.GainDB))
		}
		if spec.PPMCorrection != 0 {
			args = append(args, "-p", strconv.Itoa(spec.PPMCorrection))
		}
		args = append(args, "-")
		return CaptureCommand{Executable: tool, Arguments: args, Format: ComplexUnsigned8}, nil
	default:
		return CaptureCommand{}, fmt.Errorf("a streaming adapter for %s is not installed", device.Kind)
	}
}

func soapyDeviceArguments(device SDRDevice) string {
	driver := strings.TrimPrefix(device.Driver, "SoapySDR:")
	parts := []string{"driver=" + driver}
	if device.Serial != nil && *device.Serial != "" {
		parts = append(parts, "serial="+*device.Serial)
	}
	return strings.Join(parts, ",")
}

type IQStream struct {
	Reader io.ReadCloser
	Format SampleFormat
	cmd    *exec.Cmd
	done   chan error
	once   sync.Once
}

func StartIQStream(device SDRDevice, spec CaptureSpec) (*IQStream, error) {
	if device.Kind == "RTL-TCP" {
		return startRTLTCPStream(device, spec)
	}
	definition, err := BuildCaptureCommand(device, spec)
	if err != nil {
		return nil, err
	}
	command := exec.Command(definition.Executable, definition.Arguments...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		_ = stdout.Close()
		return nil, err
	}
	stream := &IQStream{Reader: stdout, Format: definition.Format, cmd: command, done: make(chan error, 1)}
	go func() { stream.done <- command.Wait(); close(stream.done) }()
	return stream, nil
}

func startRTLTCPStream(device SDRDevice, spec CaptureSpec) (*IQStream, error) {
	if spec.CenterFrequencyHz <= 0 || spec.SampleRateHz < 225_000 || spec.SampleRateHz > 3_200_000 {
		return nil, errors.New("remote RTL-SDR requires a positive frequency and 225 ksps to 3.2 Msps")
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(device.Host, strconv.Itoa(device.Port)), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("rtl_tcp connection failed: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	header := make([]byte, 12)
	if _, err = io.ReadFull(conn, header); err != nil || string(header[:4]) != "RTL0" {
		_ = conn.Close()
		return nil, errors.New("remote server did not provide a valid rtl_tcp stream")
	}
	commands := [][2]uint32{{1, uint32(spec.CenterFrequencyHz)}, {2, uint32(spec.SampleRateHz)}}
	if spec.AutoGain {
		commands = append(commands, [2]uint32{3, 0})
	} else {
		commands = append(commands, [2]uint32{3, 1})
		if spec.GainDB > 0 {
			commands = append(commands, [2]uint32{4, uint32(int32(spec.GainDB * 10))})
		}
	}
	if spec.PPMCorrection != 0 {
		commands = append(commands, [2]uint32{5, uint32(int32(spec.PPMCorrection))})
	}
	for _, command := range commands {
		var packet [5]byte
		packet[0] = byte(command[0])
		binary.BigEndian.PutUint32(packet[1:], command[1])
		if _, err = conn.Write(packet[:]); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	_ = conn.SetDeadline(time.Time{})
	return &IQStream{Reader: conn, Format: ComplexUnsigned8, done: make(chan error, 1)}, nil
}

func (s *IQStream) Close() error {
	var closeError error
	s.once.Do(func() {
		if s.cmd == nil {
			closeError = s.Reader.Close()
			return
		}
		_ = s.Reader.Close()
		if s.cmd.Process == nil {
			return
		}
		_ = s.cmd.Process.Signal(interruptSignal())
		select {
		case closeError = <-s.done:
		case <-time.After(2 * time.Second):
			closeError = s.cmd.Process.Kill()
		}
	})
	return closeError
}
