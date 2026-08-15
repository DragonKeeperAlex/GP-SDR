# GP-SDR 0.3.0-preview.1

This preview renames the project to GP-SDR and makes the native application the
primary product while retaining the same responsive interface for headless and
mobile use.

## Major additions

- Bundled, checksum-pinned GopherTrunk v0.9.8 P25 Phase 1/2 stack on every
  packaged platform
- P25 control/voice receiver assignment, wideband tap planning, supervised
  engine lifecycle, active-call logging, and per-talkgroup audio controls
- Real direct tuner with live FFT spectrum and waterfall
- Dedicated pages for Analog, P25, DSD-FME, rtl_433, dump1090, multimon-ng,
  acarsdec, and AIS-catcher
- Native macOS View menu and shortcuts for every main workspace
- Mobile bottom navigation and responsive tuner, mixer, tables, and dialogs
- In-app install or how-to actions for receiver tools and optional components
- Built-in GMRS and other common band profiles, custom ranges, profile sharing,
  and RadioReference preset/custom-radius import

## Packaging

- macOS arm64, x86_64, and Universal `.app` ZIPs
- Debian/Ubuntu amd64 and arm64 `.deb` packages
- Windows 10/11 amd64 ZIP
- SHA-256 manifest for every artifact
- GopherTrunk Apache-2.0 and third-party notice files in every package

The macOS previews are ad-hoc signed, not notarized. Receiver vendor/user-space
tools, Soapy modules, Whisper models, and optional non-P25 decoders are not
silently bundled; GP-SDR provides setup actions in the interface.

## Validation boundary

The live tuner was exercised on connected HackRF and RTL-SDR devices. The
bundled P25 engine was exercised with a connected HackRF through engine startup,
local API readiness, wideband voice/signaling taps, and talkgroup mixer updates.
No claim of real control-channel lock or decoded voice is made without a known
active local system and compatible RF conditions.
