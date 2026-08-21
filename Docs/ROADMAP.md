# GP-SDR roadmap

## Implemented

- [x] Native macOS app with all seven primary workspaces
- [x] Responsive desktop/mobile web interface and authenticated headless mode
- [x] Apple Silicon, Intel, Universal macOS, Linux DEB, and Windows packages
- [x] HackRF, RTL-SDR, and Soapy IQ adapters
- [x] Concurrent receiver planning and explicit device roles
- [x] Real tuner FFT/waterfall, AM/NFM/WFM DSP, squelch, WAV, and live audio
- [x] Wideband channel bank with per-channel mute, solo, volume, and activity
- [x] Bundled P25 Phase 1/2 engine, trunk following, call/talkgroup monitoring,
      per-talkgroup mixer state, and encrypted-call exclusion
- [x] Persistent activity journal and signal grouping
- [x] Concurrent Mapper job workspace with explicit receiver ownership, split
      Discovery ranges, simultaneous Discovery/Decipher, per-job progress/ETA,
      saved job controls, combined result provenance and filters, adjustable
      per-channel listening, hourly activity, optional location, and Sheets payloads
- [x] Built-in common bands plus arbitrary custom ranges/channels
- [x] Versioned profile and single-channel import/export
- [x] RadioReference ZIP search with preset and custom distance filters
- [x] Offline whisper.cpp transcription integration
- [x] Dedicated workspace for every advertised decoder
- [x] In-app install action or platform-specific instructions for components
- [x] Normalized live event bridges for every advertised optional non-P25 decoder
- [x] Indexed transcript/callsign/protocol/frequency search and configurable
      recording/IQ retention enforcement
- [x] CTCSS detection, stereo pan, priority ducking, and receiver health/storage notices

## Next hardening work

- [x] Live HackRF EBRCS control lock with Phase 1/2 grants, traffic channels,
      NAC/WACN/system identifiers, encryption flags, and JMBE codec loading
- [ ] Real-system P25 acceptance captures for additional systems
- [ ] Live multi-SDR P25 soak tests across all packaged operating systems
- [ ] DCS decode and additional adjacent-channel rejection soak tests
- [ ] Secure OS credential-store editor for RadioReference on Linux/Windows
      and any future persistent remote-access tokens
- [ ] Signed/notarized macOS distribution and signed Windows installer
- [ ] Update channel with signed manifests
