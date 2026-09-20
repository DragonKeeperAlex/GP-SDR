package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RelayPTT is the small hardware boundary used by the radio relay. The
// production implementation will drive the existing FT232 RTS line; tests
// can use a recording implementation without opening a serial device.
type RelayPTT interface {
	SetPTT(asserted bool) error
}

// RelayConfig contains deliberately conservative relay timing limits.
type RelayConfig struct {
	MaxTransmit time.Duration
	KeyDelay    time.Duration
}

func (c RelayConfig) withDefaults() RelayConfig {
	if c.MaxTransmit <= 0 {
		c.MaxTransmit = 60 * time.Second
	}
	if c.KeyDelay < 0 {
		c.KeyDelay = 0
	}
	return c
}

// RadioRelay owns the PTT lifecycle. Audio delivery is intentionally supplied
// by the caller so the same controller can later be connected to the Pi audio
// output without changing the FT232 safety behavior.
type RadioRelay struct {
	mu     sync.Mutex
	ptt    RelayPTT
	config RelayConfig
	active bool
}

func NewRadioRelay(ptt RelayPTT, config RelayConfig) (*RadioRelay, error) {
	if ptt == nil {
		return nil, errors.New("radio relay requires a PTT driver")
	}
	return &RadioRelay{ptt: ptt, config: config.withDefaults()}, nil
}

// Run keys the radio, waits for the configured settling delay, and invokes
// sendAudio. PTT is always released, including when the context is cancelled,
// the audio callback fails, or the maximum transmit time expires.
func (r *RadioRelay) Run(ctx context.Context, sendAudio func(context.Context) error) error {
	if sendAudio == nil {
		return errors.New("radio relay requires an audio callback")
	}
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return errors.New("radio relay is already active")
	}
	r.active = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.active = false
		r.mu.Unlock()
	}()

	if err := r.ptt.SetPTT(true); err != nil {
		return fmt.Errorf("assert relay PTT: %w", err)
	}
	defer func() { _ = r.ptt.SetPTT(false) }()

	if r.config.KeyDelay > 0 {
		timer := time.NewTimer(r.config.KeyDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	transmitCtx, cancel := context.WithTimeout(ctx, r.config.MaxTransmit)
	defer cancel()
	if err := sendAudio(transmitCtx); err != nil {
		return fmt.Errorf("relay audio: %w", err)
	}
	if err := transmitCtx.Err(); err != nil {
		return fmt.Errorf("relay transmit limit: %w", err)
	}
	return nil
}
