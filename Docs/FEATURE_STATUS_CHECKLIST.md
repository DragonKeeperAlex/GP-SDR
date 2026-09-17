# GP-SDR categorized feature checklist

Updated September 16, 2026. Mandatory companion to
`MASTER_REMEDIATION_BACKLOG.md`: read both before any GP-SDR work until the user
revokes this instruction. Keep stable IDs and append evidence; never delete
blocked items or check them off for build/process success alone.

This is an evidence-based open-work inventory, not a claim that every untested
feature is broken. Existing APIs/builds are not proof of every GUI button or RF
protocol. Mobile ports remain on hold; deployment remains Pi only.

## Partially implemented or awaiting acceptance

- [ ] RF-01 Clear P25 audio: PCM bridge works; subjective continuity/intelligibility acceptance remains.
- [ ] RF-02 HackRF P25: isolated good-unit lock and voice remain unaccepted.
- [ ] RF-04 Talkgroup controls: metadata/mixer exists, PCM still receiver-stream routed.
- [ ] RF-05 Separate P25 systems: backend targeting exists; simultaneous hardware/UI acceptance remains.
- [ ] RF-06 Concurrent P25 calls: complete allocation/mixing/resource acceptance remains.
- [ ] RF-07 P25 lifecycle: Phase 1/2, encrypted exclusion, rotation/crash/USB-loss recovery coverage remains.
- [ ] RF-08 Bay Area presets: control-channel and talkgroup provenance validation remains.
- [ ] RF-09 Pluto transports: USB works; Ethernet sustained/reconnect/thermal comparison remains.
- [ ] UI-01 Page organization, clipping and narrow/mobile layout acceptance remains.
- [ ] UI-02 Every button/field, full target, focus, disconnected and busy state acceptance remains.
- [ ] UI-03 Capability-aware controls need complete cross-page acceptance.
- [ ] UI-04 Applied-versus-requested settings and fallbacks need complete verification.
- [ ] UI-05 AM/NFM/WFM audio/gain/squelch/recording/reconnect acceptance on each good receiver remains.
- [ ] UI-06 Digit/spectrum tuning, software VFO, capture bounds and IQ correction acceptance remains.
- [ ] UI-07 GMRS/FRS concurrent audio/mixing/CTCSS and rejection acceptance remains.
- [ ] UI-09 Standalone analyzer interactions, multireceiver performance and disconnect acceptance remains.
- [ ] MAP-01 Identification accuracy needs labeled real-signal/noise evaluation.
- [ ] MAP-02 Live/deferred decoded payload display needs protocol-by-protocol real frames.
- [ ] MAP-04 Parallel capture needs measured missed bursts, coverage and resource limits.
- [ ] MAP-05 Offline analysis needs all-file, ETA/parallelism/cancel/recovery acceptance.
- [ ] MAP-06 Scheduler phase/overlap/ownership/restart acceptance remains.
- [ ] MAP-07 Hit denominators and eligibility filters need full live/deferred acceptance.
- [ ] MAP-08 Location grouping and distance matching need complete acceptance.
- [ ] MAP-09 Storage policies/compression/deletion/offline/NAS recovery need expendable-fixture acceptance.
- [ ] MAP-10 Media clear preserving results needs cleanup/restart acceptance.
- [ ] MAP-11 Eight-hour combined-radio soak remains.
- [ ] AI-01 Speech hallucination suppression needs measured silence/static and weak-speech evaluation.
- [ ] AI-02 Model presets/remote interruption need accuracy and performance evaluation.
- [ ] AI-04 Dataset labels/provenance exist in part; independent evaluation splits and actual training remain.
- [ ] AI-05 Explorer aggregation, occupancy/history/filter/drilldown workflow remains incomplete.
- [ ] AI-06 Reference/Sheets geographic verification, deduplication and failure handling need acceptance.
- [ ] DEC-01 DSD-FME/mbelib installed and discovered; real DMR/conventional P25 acceptance remains.
- [ ] DEC-02 Every advertised decoder needs independent real-frame GUI/live/deferred acceptance.
- [ ] TX-01 Analog waveform generation exists; controlled TX/RX spectral, level and stop acceptance remains.
- [ ] VID-01 Analog FPV picture lock, latency and quality acceptance remains.
- [ ] CAL-01 Ambient measurement exists; calibrated sensitivity/antenna workflow remains incomplete.
- [ ] PLAT-01 Android preview is not an accepted functioning RC; on hold.
- [ ] PLAT-02 iPad remote/signing/USB feasibility and actual acceptance remain; on hold.
- [ ] PLAT-03 Mobile web end-to-end, reconnect/accessibility acceptance remains.
- [ ] PLAT-04 Power-HAT telemetry needs physical acceptance.
- [ ] DIST-01 Clean-machine install/update/rollback acceptance across desktop targets remains.
- [ ] DIST-02 Updater lacks complete secure manifest/rollback/replay protection acceptance.
- [ ] DIST-03 Cross-platform credential storage/migration/security acceptance remains.
- [ ] DIST-04 Clean dependency bundles need rebuild and artifact acceptance.
- [ ] DIST-05 Wiki/setup/license/source-offer information needs alignment with shipped evidence.

## Broken or previously reproduced faults requiring closure

- [ ] RF-11 RTL USB disappearance/dropouts: prior -110/-62, direct/hub longevity remains unresolved.
- [ ] RF-02 Hardware prerequisite: c5cb produced pinned-negative/corrupt IQ; exclude from RF acceptance until repaired and retested. This is not proof of a GP-SDR decoder bug.
- [ ] AI-01 Prior non-speech hallucinations: suppression changes exist, but latest batch must demonstrate closure.
- [ ] UI-01/02 Prior clipped controls and unresponsive/click-target reports: full page-by-page GUI retest remains.

Fixed in this work tranche (bounded software scope only):

- [x] Malformed WAV uploads are rejected before persistent storage; truncated chunks/incomplete PCM frames are rejected. Shared parser regression passes. Multipart temporary upload cleanup is explicit.

- [x] Concurrent/repeated transmit preparation is rejected before allocating IQ or saving files; active-job rejection also moved before generation. This does not establish cross-workflow atomic receiver reservations.
- [x] Saved benchmark library added to Transmit: bounded metadata listing, invalid-manifest counts and synthetic eligibility/paths/checksums. Listing never starts RF, loads IQ, promotes verification or trains a model. Label review/export/replay remain open.

- [x] TX fixture memory amplification: removed two full float64 ideal-sample arrays; 60 seconds at 2 MS/s now requires ~240 MB output rather than ~2.16 GB waveform buffers. Waveform math preserved.
- [x] TX same-time fixture file collision: unique output filenames prevent overwriting an earlier benchmark.
- [x] TX nonfinite duration/fixture parameters: reject NaN/Infinity before generation.
- [x] Dataset ambiguity: manifest records generator/schema/seed and marks synthetic-only eligibility; no automatic training or receiver verification claimed.

## Missing implementations

- [ ] RF-03 Headless live SDRTrunk PCM bridge.
- [ ] RF-10 Validated direct-IIO Pluto RX1/RX2 and TX1/TX2 support with shared constraints.
- [ ] UI-08 DCS decoding/confidence/age with vectors and real radios.
- [ ] MAP-03 Non-overlapping capability-aware allocation/preview/failover (not cloned jobs).
- [ ] AI-03 Evaluated RF-trained classifier, distinct from metadata-only LLM suggestions.
- [ ] AI-04 Actual dataset training pipeline and held-out evaluation; storage alone is not training.
- [ ] TX-02 Microphone transmit and physical acceptance.
- [ ] TX-03 Standards-compliant digital voice/data TX and independent protocol/spectral acceptance. Generic keyed fixtures are not P25/DMR packets.
- [ ] TX-04 Standard image/fax/modem transmit/receive and progressive display.
- [ ] VID-02 A specifically supported digital FPV protocol.
- [ ] DIST-06 Public signing/notarization with appropriate credentials.
- [ ] TX-01..04 Lab orchestration: atomic TX/RX reservations, receiver-first start, paired emergency stop/timeout, captured IQ versus truth comparison, persisted paired-run results.
- [ ] AI-04 Lab review/export: corrected labels, real/synthetic separation, capture provenance, independent decoder evidence and dataset split controls.

## This session's evidence and limits

- Pi deployment: `1.5.0-rc34-pi-test4`, SHA-256 `9ad24e45eefc264664cf8319087ebe7133d8bc0a346c05bc33d538e7ceba0c0e`. Rollback binary plus Data/Profiles snapshot: `/home/sdr/gpsdr-before-pi-test4-H6JZiN`. Existing service/configuration retained. Service active; DSD-FME ready; benchmark library endpoint and served UI present. Restored existing `pluto-p25-hardware-test` profile on its prior Pluto. No TX started. An empty production fixture library is expected: isolated test fixtures were not copied into user data or promoted to training samples.

- Local Go unit suite, race suite and vet passed after fixture changes.
- Audio scheduling and Band Monitor control/ownership JavaScript regression tests passed; app.js syntax passed.
- Running Pi service remained active with zero service restarts; device/decoder/integration/local-AI/transmit status GETs returned HTTP 200.
- Standalone analyzer's correct GET `/api/spectrum-analyzer` returned HTTP 200 (the nonexistent `/status` suffix returned 404, not an analyzer failure).
- Pi ARM64 `1.5.0-rc34-lab-test3` isolated HTTP test generated all ten fixture families (CW/AM/NFM/WFM/OOK/2FSK/GFSK/GMSK/BPSK/QPSK), each 200,000 complex samples, dry-run complete with checksummed provenance. Evidence: `/mnt/gp-sdr-data/gpsdr-lab-acceptance-ek6lyjx7`. Test server terminated afterward; production service/binary and user data unchanged. This is physical-host execution of synthetic fixtures, not a physical RF test.
- M3 Pro benchmark: 200,000-sample QPSK generation took ~6 ms and allocated 410,272 bytes; benchmark speed is not Pi throughput acceptance.
- No RF transmitted or test audio played. No attenuated wiring/input-limit evidence was available, so physical TX acceptance is blocked, not passed.
- These checks do not constitute exhaustive installed GUI or real-hardware protocol acceptance. The coordinated lab is not finished; no public release is claimed.
