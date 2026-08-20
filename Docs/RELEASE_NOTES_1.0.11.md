# GP-SDR 1.0.11

This maintenance release strengthens shutdown reliability, acceptance coverage,
and first-run setup without changing GP-SDR's native-first workflow.

## Changes

- Fixed clean shutdown of remote `rtl_tcp` streams so a normal disconnect no
  longer closes the network reader twice or reports a false receiver error
- Added end-to-end API acceptance tests for hardware state, tuner controls,
  calibration persistence, remote receivers, profiles, bulk imports, Mapper,
  mixer controls, and safe setup failures
- Added audio fan-out and offline transcription command-path tests
- Increased race-enabled server statement coverage to 48 percent
- Added a complete cross-platform installation, first-signal, P25, headless,
  driver, checksum, and troubleshooting guide
- Added release-build protection against cloud-created duplicate source files

## Validation scope

- Race detection, complete Go tests, static analysis, web syntax checks, native
  page navigation, every decoder page, immediate settings, and a 390 x 844
  mobile viewport passed
- API authorization, profile import/export, Mapper export/upload safety, tuner
  start/stop, P25 demo orchestration, recording lookup, and LAN security headers
  passed
- macOS Universal, Apple Silicon, Intel, Linux amd64/arm64, and Windows amd64
  packages are checksum-verified during the release build

The previously validated live HackRF EBRCS P25 control-channel lock and decoded
event remain documented evidence. Hardware was not considered revalidated unless
the host could enumerate the connected device during this release gate. Windows,
Linux, live RTL-SDR, and optional decoder execution remain host-specific tests.
