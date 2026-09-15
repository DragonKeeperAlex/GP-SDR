# GP-SDR 1.5.0-rc31

## Band Monitor

- Adds a live spectrum and waterfall for the receiver's shared wideband capture.
- Shows the exact frequency and level under the pointer and marks enabled bank channels.
- Adds AAR railroad, public-safety interoperability, marine calling/safety, and
  emergency/calling built-in channel banks.
- Keeps per-channel mute, solo, volume, CTCSS/DCS evidence, and receiver controls together.
- Handles a stale or disconnected selected receiver without breaking the interface.

## Raspberry Pi and PiPower5

- Adds live PiPower5 input, output, battery, charge, and source telemetry to Hardware.
- The HAT reader is Linux-only, cached, and time-bounded. macOS and Windows return no HAT
  data and do not invoke Python.
- Linux ARM64 packaging includes the Soapy stream helper used by PlutoSDR receivers.

## Validation

- Go unit tests, vet, formatting checks, and cross-compilation pass for macOS Intel/Apple
  Silicon, Linux AMD64/ARM64, and Windows AMD64.
- On the Raspberry Pi, live 98.1 MHz captures were verified with PlutoSDR and HackRF.
- A 10 MS/s HackRF GMRS Band Monitor run produced a 512-bin live spectrum while exposing
  all 30 configured mixer channels.
- PiPower5 telemetry was read as the unprivileged GP-SDR service account.

These checks validate the cited paths and attached hardware; they do not claim that every
decoder protocol had a matching live transmission during release acceptance.
