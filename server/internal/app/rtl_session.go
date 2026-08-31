package app

import (
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

// localRTLSession keeps rtl_tcp (and therefore the physical USB device) open
// across short scan windows. The per-device mutex guarantees that Live,
// Tuner, Mapper, and calibration cannot claim the same receiver concurrently.
type localRTLSession struct {
	mu      sync.Mutex
	stateMu sync.Mutex
	cmd     *exec.Cmd
	done    chan struct{}
	address string
	conn    net.Conn
	stderr  lockedBuffer
}

type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(data)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func (b *lockedBuffer) Reset() {
	b.mu.Lock()
	b.b.Reset()
	b.mu.Unlock()
}

var localRTLSessions = struct {
	sync.Mutex
	items map[string]*localRTLSession
}{items: make(map[string]*localRTLSession)}

func startLocalRTLStream(device SDRDevice, spec CaptureSpec) (*IQStream, error) {
	if spec.CenterFrequencyHz <= 0 || spec.SampleRateHz < 225_000 || spec.SampleRateHz > 3_200_000 {
		return nil, errors.New("RTL-SDR requires a positive frequency and 225 ksps to 3.2 Msps")
	}
	localRTLSessions.Lock()
	session := localRTLSessions.items[device.ID]
	if session == nil {
		session = &localRTLSession{}
		localRTLSessions.items[device.ID] = session
	}
	localRTLSessions.Unlock()

	session.mu.Lock()
	stream, err := session.open(device, spec)
	if err != nil {
		session.mu.Unlock()
		return nil, err
	}
	stream.release = session.mu.Unlock
	return stream, nil
}

func (s *localRTLSession) open(device SDRDevice, spec CaptureSpec) (*IQStream, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		address, err := s.ensureRunning(device)
		if err == nil {
			err = s.ensureConnected(address, 3*time.Second)
		}
		if err == nil {
			s.stateMu.Lock()
			conn := s.conn
			s.stateMu.Unlock()
			err = configureRTLTCPConnection(conn, spec)
			if err == nil {
				err = discardRTLSettlingSamples(conn, spec.SampleRateHz, 75*time.Millisecond)
			}
			if err == nil {
				return &IQStream{Reader: &persistentRTLReader{session: s, conn: conn}, Format: ComplexUnsigned8, done: make(chan error, 1)}, nil
			}
		}
		lastErr = err
		if detail := strings.TrimSpace(s.stderr.String()); detail != "" {
			lastErr = fmt.Errorf("%w: %s", err, detail)
		}
		s.stop()
		if attempt == 0 {
			time.Sleep(150 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("persistent RTL-SDR session failed after recovery: %w", lastErr)
}

func (s *localRTLSession) ensureConnected(address string, timeout time.Duration) error {
	s.stateMu.Lock()
	if s.conn != nil {
		s.stateMu.Unlock()
		return nil
	}
	s.stateMu.Unlock()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			err = initializeRTLTCPConnection(conn)
		}
		if err == nil {
			s.stateMu.Lock()
			s.conn = conn
			s.stateMu.Unlock()
			return nil
		}
		if conn != nil {
			_ = conn.Close()
		}
		lastErr = err
		time.Sleep(35 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("server did not accept a receiver connection")
	}
	return lastErr
}

func discardRTLSettlingSamples(conn net.Conn, sampleRate int, duration time.Duration) error {
	if conn == nil {
		return errors.New("persistent RTL-SDR connection is unavailable")
	}
	byteCount := int64(float64(sampleRate*2) * duration.Seconds())
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err := io.CopyN(io.Discard, conn, byteCount)
	_ = conn.SetReadDeadline(time.Time{})
	return err
}

type persistentRTLReader struct {
	session *localRTLSession
	conn    net.Conn
}

func (r *persistentRTLReader) Read(data []byte) (int, error) {
	count, err := r.conn.Read(data)
	if err != nil {
		r.session.markConnectionBroken(r.conn)
	}
	return count, err
}

func (r *persistentRTLReader) Close() error { return nil }

func (s *localRTLSession) markConnectionBroken(conn net.Conn) {
	s.stateMu.Lock()
	if s.conn == conn {
		s.conn = nil
	}
	s.stateMu.Unlock()
	_ = conn.Close()
}

func (s *localRTLSession) ensureRunning(device SDRDevice) (string, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil && s.address != "" {
		return s.address, nil
	}
	tool, err := findTool("rtl_tcp")
	if err != nil {
		return "", errors.New("rtl_tcp is required for persistent RTL-SDR sessions")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	address := listener.Addr().String()
	_ = listener.Close()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}
	s.stderr.Reset()
	command := exec.Command(tool, "-a", "127.0.0.1", "-p", port, "-d", rtlDeviceSelector(device))
	command.Stderr = &s.stderr
	command.Stdout = &s.stderr
	if err := command.Start(); err != nil {
		return "", err
	}
	done := make(chan struct{})
	s.cmd, s.done, s.address = command, done, address
	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
		var conn net.Conn
		s.stateMu.Lock()
		if s.cmd == cmd {
			conn = s.conn
			s.cmd, s.done, s.address, s.conn = nil, nil, "", nil
		}
		s.stateMu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		close(done)
	}(command)
	return address, nil
}

func rtlDeviceSelector(device SDRDevice) string {
	if device.Serial != nil && strings.TrimSpace(*device.Serial) != "" {
		return strings.TrimSpace(*device.Serial)
	}
	pieces := strings.Split(device.ID, "-")
	if len(pieces) > 1 {
		if _, err := strconv.Atoi(pieces[len(pieces)-1]); err == nil {
			return pieces[len(pieces)-1]
		}
	}
	return "0"
}

func (s *localRTLSession) stop() {
	s.stateMu.Lock()
	command := s.cmd
	done := s.done
	conn := s.conn
	s.cmd, s.done, s.address, s.conn = nil, nil, "", nil
	s.stateMu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	if command != nil && command.Process != nil {
		_ = command.Process.Signal(interruptSignal())
		if done != nil {
			select {
			case <-done:
				return
			case <-time.After(2 * time.Second):
			}
		}
		_ = command.Process.Kill()
		if done != nil {
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
}

func shutdownLocalRTLSessions() {
	localRTLSessions.Lock()
	sessions := make([]*localRTLSession, 0, len(localRTLSessions.items))
	for _, session := range localRTLSessions.items {
		sessions = append(sessions, session)
	}
	localRTLSessions.items = make(map[string]*localRTLSession)
	localRTLSessions.Unlock()
	for _, session := range sessions {
		session.stop()
	}
}
