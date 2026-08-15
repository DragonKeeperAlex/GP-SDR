# Signal Harbor 0.2.0-preview.1

This preview turns Signal Harbor's original simulator into an early live receiver pipeline while keeping all hardware-dependent features explicit about their readiness.

## Highlights

- Built-in AM, narrow FM, and wide FM demodulation with WAV recording and live browser audio
- Sequential scan and wideband channel-bank modes for compatible HackRF, RTL-SDR, and SoapySDR receivers
- Independent mute, solo, and volume controls for simultaneously active channels
- OP25 process supervision, generated trunking configuration, control/voice receiver assignments, and encrypted-talkgroup exclusion
- Optional offline transcription through `whisper-cli`
- Authorized RadioReference ZIP import with 5, 10, 25, 50, and 100 mile presets plus a custom 1–100 mile range
- Conventional and individual P25 imports saved as separate portable profiles
- Driver, decoder, account, and model readiness shown in the interface

## Reliability fixes

- Fixed the immediate interface freeze caused by drawing a spectrum grid while its hidden canvas had zero height
- Starting without a connected SDR now fails quickly with a useful status message
- Dialog close buttons and navigation have browser regression coverage
- Live audio fan-out is bounded so a slow browser cannot stall the receiver pipeline

## Packaging

- Universal, Apple Silicon, and Intel macOS applications
- Linux amd64 and arm64 Debian packages
- Windows 10/11 x86-64 archive
- License, notice, and third-party attribution files in every package

## Known limits

- The macOS applications are ad-hoc signed, not notarized. Windows packages are not code-signed.
- Vendor drivers are not embedded into the main executable. Signal Harbor discovers compatible system tools and libraries; optional SoapySDR helpers are included only when they are built in the packaging environment.
- Live RadioReference use requires a Premium account and a separately approved API key.
- OP25 and whisper.cpp remain separate programs under their own licenses and must be installed/configured by the user unless a future release explicitly records and bundles a compatible build.
- Hardware-in-the-loop acceptance still needs to be repeated on each supported receiver, OS, P25 system, and RadioReference account configuration.
