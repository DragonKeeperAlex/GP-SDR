# GP-SDR 1.0.10

This release completes the SDRTrunk-based P25 path and focuses on a responsive,
native-first GP-SDR experience.

## Highlights

- Bundled SDRTrunk v0.6.1 and JMBE Creator v1.0.9 for macOS, Windows, and Linux
- P25 Phase 1/2 control-channel tracking, call logging, and talkgroup mixer controls
- Live HackRF tuner audio, spectrum, waterfall, squelch, gain, amplifier, IQ/DC
  correction, saved calibration, and immediate control feedback
- Multi-receiver support for HackRF, RTL-SDR, SoapySDR, and remote `rtl_tcp`
- Mapper Discovery and Decipher modes with occupancy counts, CSV output, optional
  location tags, transcription hooks, and reviewed Google Sheets uploads
- Bulk local radio-database imports and RadioReference range-based integration
- Native macOS location permission flow and a responsive mobile Web UI

## Release-gate validation

- Go tests, race detection, static analysis, web regression checks, and both macOS
  architectures passed
- Universal macOS bundle architecture, signing integrity, bundled P25 discovery,
  and checksum verification passed
- A connected HackRF locked an EBRCS P25 control channel through GP-SDR, loaded
  JMBE, logged a decoded P25 event, and applied talkgroup mute changes
- A short live Mapper sweep completed without false-positive records; RF hits remain
  dependent on the selected range, antenna, location, and dwell time

The macOS preview bundle is ad-hoc signed, not Apple-notarized. Windows may need
the correct WinUSB driver. Hardware acceptance for RTL-SDR and non-macOS packages
remains device- and host-specific.
