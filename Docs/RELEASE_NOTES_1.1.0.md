# GP-SDR 1.1.0

GP-SDR 1.1 is a major app-first feature and hardening update. It keeps the
native macOS app as the primary experience while retaining the authenticated,
mobile-responsive companion interface.

## Multi-receiver Mapper

- Independent saved Mapper jobs assign one physical receiver each.
- Multiple receivers can sweep different ranges concurrently, split Discovery
  coverage, or run Discovery and Identify at the same time.
- Every job reports its current frequency, channel/pass progress, checks,
  activity, elapsed time, and next-pass ETA.
- Identify listening is adjustable from five seconds through multiple days.
- Combined results retain receiver/job provenance and can be filtered,
  expanded, exported, saved locally, or queued to a configured Google Sheet.

## Identification and history

- Added normalized live handoffs for DSD-FME, rtl_433, dump1090, multimon-ng,
  acarsdec, and AIS-catcher. HackRF signed IQ is shifted, resampled where needed,
  and converted before decoders that require centered unsigned samples.
- Added analog CTCSS detection with narrow-peak validation.
- Timeline search uses an in-memory index covering transcripts, callsigns,
  labels, systems, protocols, decoder output, and frequencies.
- Profile logging exposes audio recording, unknown-signal IQ evidence, offline
  transcription, and one-day through forever retention controls.

## Reliability and setup

- Hardware cards show active Mapper ownership and live receiver telemetry.
- Live health notices report overload, dropped sample blocks, noise-only state,
  and excessive GP-SDR capture storage. Settings shows bounded storage usage for
  GP-SDR-owned data, recording, IQ, and profile folders.
- The macOS app bundles a one-click, revision-pinned optional decoder-suite
  installer. Receiver tools and transcription retain their existing Install or
  platform-specific How to actions.
- The release workflow now supports validated manual dispatch, semantic-version
  checks, artifact-count/package verification, and release-note publishing.

## Release-gate evidence

- Go unit, HTTP acceptance, web regression, race, vet, JavaScript syntax, and
  cross-compile tests cover the complete source tree.
- Installed DSD-FME, rtl_433, dump1090, multimon-ng, acarsdec, and AIS-catcher
  binaries pass GP-SDR bridge smoke tests on macOS.
- Concurrent live Mapper sweeps used the HackRF at 8 MS/s and RTL-SDR at
  2.4 MS/s. They independently detected 88.5 and 99.7 MHz FM activity,
  preserved correct receiver/job provenance, advanced simultaneously, and
  released both devices immediately after Stop.
- Simulator acceptance separately covers two simultaneous receiver jobs and
  the mobile layout.

## Known external boundaries

- RadioReference live lookup still requires the service's Premium subscription
  and separately approved application key; local exported databases work
  without an API key.
- Public macOS notarization and Windows code signing require suitable signing
  certificates. Checksums are included for every artifact.
- Encrypted P25 calls are identified and skipped; GP-SDR does not decrypt or
  transmit.
