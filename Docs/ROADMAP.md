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
- [x] Mapper Discovery/Decipher workflows, live scan progress, adjustable
      per-channel listening, hourly activity, expandable identification evidence,
      optional location tags, and Google Sheets payloads
- [x] Built-in common bands plus arbitrary custom ranges/channels
- [x] Versioned profile and single-channel import/export
- [x] RadioReference ZIP search with preset and custom distance filters
- [x] Offline whisper.cpp transcription integration
- [x] Dedicated workspace for every advertised decoder
- [x] In-app install action or platform-specific instructions for components

## Next hardening work

- [x] Live HackRF EBRCS control lock with Phase 1/2 grants, traffic channels,
      NAC/WACN/system identifiers, encryption flags, and JMBE codec loading
- [ ] Real-system P25 acceptance captures for additional systems
- [ ] Live multi-SDR P25 soak tests across all packaged operating systems
- [ ] CTCSS/DCS decode, pan, priority ducking, and improved channelizer rejection
- [ ] Normalized live event bridges for every optional non-P25 decoder
- [ ] Secure OS credential-store editor for RadioReference and remote tokens
- [ ] Indexed transcript search and configurable storage retention enforcement
- [ ] Signed/notarized macOS distribution and signed Windows installer
- [ ] Update channel with signed manifests
- [ ] Dropped-sample, disk-use, and receiver-health notifications
