# GP-SDR 1.1 feature status

This document distinguishes implementation from live-signal proof. A process
starting successfully or an RF-energy reading alone is not decoder acceptance.

## Verified for this release

- Native macOS app launch, navigation, tuner controls, responsive companion UI,
  authenticated API, import/export, persistence, indexed search, retention,
  storage reporting, and receiver health reporting.
- Simultaneous HackRF and RTL-SDR Mapper jobs with independent ownership,
  frequency progress, ETA, results, provenance, stop, and device release.
- Live analog FM activity with both attached receivers: 88.5 MHz on HackRF and
  99.7 MHz on RTL-SDR.
- AM, NFM, and WFM DSP plus squelch, DC removal, IQ correction controls, gain,
  HackRF amplifier control, receiver-centered/listening-frequency separation,
  master audio, channel/talkgroup mute, solo, volume, pan, and priority ducking.
- Bundled SDRTrunk P25 Phase 1/2 stack and JMBE Creator. Prior live EBRCS
  acceptance recorded control lock, NAC/WACN/system identity, channel grants,
  encryption flags, traffic channels, and JMBE loading.
- Normalization bridges for DSD-FME, rtl_433, dump1090, multimon-ng, acarsdec,
  and AIS-catcher. Every installed bridge passed an executable smoke test using
  representative input and parser output.
- Unit, HTTP acceptance, browser regression, JavaScript syntax, race-detector,
  static-analysis, cross-compile, packaging, checksum, and package-content gates.

## Implemented, but dependent on local signal or equipment

- Live DMR/P25 conventional, sensor, ADS-B, pager/signaling, ACARS, and AIS
  event decoding. The bridges are tested, but a release cannot promise that a
  compatible transmission is present at a user's antenna during a test.
- P25 talkgroup audio quality. RF gain, overload, oscillator error, antenna,
  USB throughput, system channel plan, encryption, and available traffic all
  affect field results; the app exposes the controls and diagnostics needed to
  distinguish them.
- SoapySDR hardware other than the attached HackRF and RTL-SDR, and remote
  `rtl_tcp` servers. Adapters and acceptance tests are included, but each model
  and remote server still requires its corresponding driver and hardware.
- Offline transcription. GP-SDR supports local whisper.cpp and provides an
  install action, but the executable and model are not silently downloaded.
- Location tagging. macOS authorization is requested when enabled; an operator
  may deny it or enter coordinates manually.

## External requirements intentionally not bypassed

- RadioReference live API access requires the user's Premium account and an
  approved application key. Local user-exported database folders work without
  a key.
- Public macOS notarization, Windows signing, and Apple Developer ID signing
  require suitable certificates. The current machine does not have a usable
  Developer ID Application identity, so public packages remain ad-hoc signed
  and include SHA-256 checksums.
- GitHub-hosted Actions cannot start while the repository owner's GitHub
  account is locked for billing. The workflow itself validates manual and tag
  releases; local builds and connector uploads provide a release fallback.
- Encrypted P25 audio is not decrypted. GP-SDR identifies and logs encryption.
- GP-SDR remains receive-only. HackRF transmit controls are deliberately not
  part of 1.1; safe transmitting requires authorization, band-specific limits,
  interlocks, test loads, and a substantially different safety model.

## Remaining engineering work

- DCS subtone decoding and broader adjacent-channel rejection soak testing.
- Live multi-SDR P25 soak tests on several trunk systems and on packaged Linux
  and Windows hosts.
- Secure credential-store editing for RadioReference on Linux/Windows.
- Signed update manifests, notarized macOS distribution, and signed Windows
  installation packages when the required credentials are available.
