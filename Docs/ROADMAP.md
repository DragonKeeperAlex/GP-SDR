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
      Discovery ranges, simultaneous Discovery/Identify, per-job progress/ETA,
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

## Visual data explorer (initial implementation in 1.5.0-rc10)

- [x] Explore tab with text/date/frequency/location/modulation filters, recorded
      event heatmap, UTC daily timeline, offline coordinate plot, grouped capture
      inspection, bookmarks/tags, and locally saved views. Nearby imported
      reference-area candidates are explicitly not confirmed transmitter sites.
- [x] Conservative digital-silence/static transcription gate and standalone
      non-speech caption cleanup, with synthetic silence/static/quiet-waveform
      regression tests. Does not rewrite old transcripts or guarantee no hallucinations.

The larger goals below remain incomplete; see Wiki/Explore.md for implemented
scope and limits. Explore currently reads up to 25,000 loaded journal events.

- [ ] Add a dedicated Explore tab for visually reviewing and organizing collected
      data; keep views linked by shared frequency, date/time, location, receiver,
      job, modulation, and verification filters.
- [ ] Frequency-versus-time heatmap with activity counts, signal levels, and
      successful-check percentage. Distinguish unobserved periods from checked
      but inactive periods; do not mistake longer monitoring for greater activity.
- [ ] Activity timelines and hourly/day-of-week patterns, showing monitoring time,
      sample counts, and gaps alongside totals and occupancy.
- [ ] Interactive collection map showing receiver observation locations, grouped
      by frequency and receive area, with optional time playback. Never infer a
      transmitter location from a receiver location or signal strength alone.
- [ ] Separate reference layer for geographically applicable RadioReference or
      imported verified records: show source, reference date, match evidence,
      distance, and verification status. Plot transmitter sites only when the
      source explicitly supplies them; distinguish area records from site points.
- [ ] Click heatmap cells, graphs, or map markers to inspect contributing captures,
      transcripts, decoder evidence, and source citations. Support bookmarks,
      tags, and saved filtered views for organizing findings.
- [ ] Scale to large datasets using aggregation and incremental loading; retain
      useful offline views and avoid requiring new API credentials for the core
      explorer. Keep precise location sharing/export explicitly opt-in.

Requested 2026-09-02; initial subset implemented 2026-09-03.

## Next hardening work

- [ ] Further hardening: suppress transcription hallucinations on empty,
      silent, or noise-only audio. User reports invented descriptions such as
      water sounds and engine noises. Investigate speech-activity/no-speech
      gating and confidence checks; leave the transcript blank or show a separate
      "No speech detected" status when appropriate. Verify against silence,
      RF static, weak real speech, and normal speech recordings so useful weak
      transmissions are not discarded. Initial heuristic protection is implemented;
      recorded weak-speech field acceptance and model-confidence gating remain.

- [x] Live HackRF EBRCS control lock with Phase 1/2 grants, traffic channels,
      NAC/WACN/system identifiers, encryption flags, and JMBE codec loading
- [ ] Real-system P25 acceptance captures for additional systems
- [ ] Live multi-SDR P25 soak tests across all packaged operating systems
- [ ] DCS decode and additional adjacent-channel rejection soak tests
- [ ] Secure OS credential-store editor for RadioReference on Linux/Windows
      and any future persistent remote-access tokens
- [ ] Signed/notarized macOS distribution and signed Windows installer
- [ ] Update channel with signed manifests
