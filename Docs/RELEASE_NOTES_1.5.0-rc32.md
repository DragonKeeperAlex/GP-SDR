# GP-SDR 1.5.0-rc32

This release candidate completes the Linux/Pi decoder and receiver integration
pass that followed rc31.

## Changes

- Routes PlutoSDR and other SoapySDR P25 assignments through OP25 because
  SDRTrunk does not provide a native Pluto tuner.
- Keeps native RTL-SDR and HackRF P25 assignments on SDRTrunk.
- Reports SDRTrunk and OP25 readiness together on the Hardware page.
- Includes a Linux OP25 launcher with all required OP25 application module
  paths.
- Refreshes PiPower5 telemetry asynchronously so an unavailable or slow HAT
  can never stall the status API.
- Adds regression coverage for Pluto-to-OP25 routing and preserves the native
  HackRF/RTL-SDR path.

## Hardware validation

- RTL-SDR: exact 48 MB bounded capture at 2.4 MS/s on 98.1 MHz.
- HackRF: exact 40 MB bounded capture at 10 MS/s on 98.1 MHz.
- PlutoSDR: sustained SoapySDR capture at 4 MS/s on 98.1 MHz.
- OP25 native GNU Radio bindings and the packaged `multi_rx.py` launcher were
  started successfully on Raspberry Pi OS ARM64.

Control-channel lock and decoded voice remain dependent on local RF conditions,
the selected system profile, antenna, and an installed vocoder component.
