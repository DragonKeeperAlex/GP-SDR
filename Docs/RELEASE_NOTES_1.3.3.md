# GP-SDR 1.3.3

## Hardware and P25 reliability

- HackRF discovery now keeps each `hackrf_info` self-test result attached to the correct radio. A failed unit is shown as connected but unavailable and is not assigned to P25, Live, Tuner, or Mapper jobs.
- SDRTrunk tuner selection now prefers persisted USB bus/port IDs and avoids inventing a serial preference when multiple HackRFs are configured. This prevents the wrong HackRF from being selected in a two-radio setup.
- The P25 hardware probe continues to report receiver health separately from control-channel lock; a running decoder is not treated as a lock without control data.

## Guarded transmit workspace

- Added a Transmit page for HackRF AM, NFM, and WFM WAV playback.
- Dry-run IQ generation is enabled by default and can be tested without RF output.
- Real RF output is local-only, capped at 60 seconds, requires an available HackRF and an explicit safety confirmation, and is never available through a remote web client. Use a dummy load and comply with local regulations.

## Validation

- Go unit, race, vet, web regression, and optional-decoder smoke tests pass.
- Release packages are built for macOS arm64/x86_64/universal, Windows x86_64, and Linux amd64/arm64.
