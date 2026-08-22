# GP-SDR Mobile

This folder contains the receive-only GP-SDR port for iPadOS and Android. The
desktop application remains unchanged. Mobile profiles retain the versioned
GP-SDR JSON schema so a profile can move between desktop and tablet without a
conversion service.

## Current status

- Shared Swift models for profiles, P25 systems, events, radio configuration,
  and receiver telemetry
- Validated GP-SDR profile import/export
- Hardware-independent receiver interface for RTL-SDR, HackRF, and simulation
- Synthetic IQ source that is always identified as simulated
- Initial spectrum processing and automated tests
- AM/NFM/WFM/SSB demodulation core, signal-level estimation, and bounded scan planning
- Pinned RTL-SDR DriverKit reference as a Git submodule under `Drivers/`
- Tested RTL-SDR transport adapter that converts the DriverKit shared ring into
  bounded `IQFrame` streams and exposes dropped-sample telemetry
- Tested receive-only HackRF transport boundary with signed-IQ conversion,
  bounded mobile sample rates, and dropped-sample telemetry. A working HackRF
  DriverKit user client is still required for live hardware.
- iPadOS shell and USBDriverKit integration are under active development

## Hardware boundary

The application is receive-only. Exactly one receiver implementation owns an
attached radio at a time. USB drivers publish IQ through bounded shared-memory
buffers; decoding, recording, persistence, and UI stay in the app sandbox.

RTL-SDR DriverKit work is based on the GPL-2.0 project at
https://github.com/arvedviehweger/RTL-SDR-USB-iPadOS. Any imported derivative
source must retain its license and attribution. Public distribution requires a
separate license and App Store review assessment; private developer-signed use
is the initial target.

## Test

From this directory:

```sh
swift test
```

## iPad app

Generate the installable Xcode project with `xcodegen generate`, then build or
run the `GPSDRMobile` scheme on an iPad or iPad simulator. The generated project
is intentionally ignored; `project.yml` is the reviewable source of truth.
