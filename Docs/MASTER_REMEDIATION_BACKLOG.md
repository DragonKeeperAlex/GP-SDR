# GP-SDR master remediation backlog

Approved by the user September 16, 2026. Saved from the complete outstanding-work overview in the GP-SDR task. Baseline source: `0fd40fc`.

## Mandatory working rule

**Resolve this backlog before implementing any new features outside it.** Missing features already listed here are in scope; unrelated feature expansion is frozen. Reference this file at the start of each GP-SDR work session and when handing work between tasks. It supersedes older roadmap completion claims and feature-gap reports where they disagree.

Keep stable item IDs. Do not remove an item because it is difficult or blocked. Record evidence, source revision, installed version/host, and remaining limitations before checking it off. A build, menu, running process, RF energy, or valid PCM alone is not end-to-end acceptance. Distinguish software/API tests from physical RF tests and subjective audio/video acceptance. User-dependent items remain open until their requirements are met; never silently waive them.

Current installation scope: Pi only. No new public release or Mac replacement unless requested. Preserve recordings, results, profiles, calibration, credentials, and recovery paths. Keep test audio out of the user's headphones. RF transmit acceptance requires an appropriate authorized, attenuated/dummy-load bench setup; user permission alone does not establish lawful operation.

Platform priority confirmed September 16, 2026: Raspberry Pi is the primary development and acceptance target. Android and iOS/iPadOS ports (PLAT-01/02) are ON HOLD until the main app is fully functional. Retain their backlog entries; do not spend implementation time on them meanwhile. Keep other desktop platforms compatible but focus active testing/configuration on Pi.

## Acceptance states

`OPEN`, `IN PROGRESS`, `FAILED — reproducible`, `BLOCKED — named requirement`, `PASSED — software/API only`, `PASSED — physically verified`. Checkboxes represent complete acceptance for the stated scope, not code presence. Append dated evidence below; broader platform acceptance remains open if only Pi testing passes.

## P25 and receivers

- [ ] RF-01 P25 audio quality: listening tests establish clear, continuous Pluto/RTL voice playback through GP-SDR. Baseline: Pluto 818 valid PCM frames; RTL genuine voice recordings; intelligibility unverified.
- [ ] RF-02 Independently obtain HackRF P25 control lock and voice on each working unit. Earlier fallback-based attribution is invalid; latest isolated good-unit test searched.
- [ ] RF-03 Bridge live SDRTrunk audio into headless GP-SDR. OP25 PCM bridge exists; SDRTrunk native output/recordings are not a live web audio bridge.
- [ ] RF-04 Associate OP25 audio with talkgroups; verify individual volume, pan, mute, solo. Baseline: receiver-stream controls, not complete per-talkgroup routing.
- [ ] RF-05 Hardware-test simultaneous separate-system receiver assignments; provide a clear assignment UI. Backend targeting exists.
- [ ] RF-06 Verify multiple simultaneous P25 calls, bandwidth allocation, mixing, and resource limits.
- [ ] RF-07 Verify Phase 1/2, encryption exclusion, control rotation, signal loss, USB loss, decoder crash, and reacquisition.
- [ ] RF-08 Refine 76 Bay Area presets with current control channels, identifiers, talkgroup names, source dates. Imported site frequencies remain unverified control candidates.
- [ ] RF-09 Verify Pluto Ethernet streaming, reconnect, sustained throughput, USB comparison, and thermal behavior. USB P25 lock/grants/PCM confirmed.
- [ ] RF-10 Expose and safely test Pluto RX1/RX2 and TX1/TX2 with compatible firmware/direct IIO. Current Soapy exposes 1 RX/1 TX; shared tuning constraints must be shown.
- [ ] RF-11 Resolve RTL disappearance; compare direct/hub paths, cables, prolonged combined-radio operation. Baseline: 30-minute hub soak, small loss, earlier USB -110/-62.

## UI, tuner, Band Monitor, spectrum

- [ ] UI-01 Complete page-by-page UI restructuring: sensible settings, reduced clutter, unclipped text, cohesive layouts, narrow/mobile widths, useful feedback.
- [ ] UI-02 Test every visible button/field, full click target, keyboard/focus, busy/disconnected states. Specific refresh/receiver-selection fixes are not full acceptance.
- [ ] UI-03 Consistently expose only supported receiver gain, AGC, bias power, bandwidth, correction, and transport controls on every page.
- [ ] UI-04 Show actual applied rate/gain and backend fallback rather than merely requested values.
- [ ] UI-05 Hardware-test AM/NFM/WFM on each receiver: audio, squelch, immediate gain, output selection, recording, stop/reconnect.
- [ ] UI-06 Verify spectrum click and digit tuning/software VFO without unnecessary retuning, with capture bounds and DC/IQ corrections.
- [ ] UI-07 Verify full GMRS/FRS concurrent listening, channel identity, mute/solo, adjacent-channel rejection, CTCSS reporting.
- [ ] UI-08 Implement and verify DCS with confidence/age metadata, vectors and real radios.
- [ ] UI-09 Verify standalone Spectrum Analyzer: full/narrow sweeps, multiple receivers, zoom/scroll/hover/max-hold/reset/export/disconnect, bounded CPU, no Mapper jobs or retained IQ.

## Mapper, analysis, storage

- [ ] MAP-01 Establish identification accuracy against labeled real analog/digital signals and noise; require real decoded frames or geographically valid authoritative matches.
- [ ] MAP-02 Reliably display actual digital payload/fields in live and deferred results, not detection banners alone.
- [ ] MAP-03 Implement capability-aware non-overlapping multi-SDR partitioning, assignment preview, failover; do not just clone templates.
- [ ] MAP-04 Benchmark high parallel-channel capture: usable coverage, missed bursts, CPU/USB/storage limits; disclose practical limits.
- [ ] MAP-05 Verify capture-now/analyze-later across all relevant stored files without an SDR; progress, ETA, parallelism, cancel, restart/recovery.
- [ ] MAP-06 Verify scheduling/automatic Discovery→Identify phases, timing, overlap, ownership, restart.
- [ ] MAP-07 Verify hits/check denominators, Identify history updates, minimum-hit/percentage filters, repeat-only results; prevent one-off self-promotion.
- [ ] MAP-08 Verify location-separated grouping, same-frequency combination where appropriate, distance-aware reference matching and opt-in location sharing.
- [ ] MAP-09 Test every storage policy/cap, compression, quarantine expiry, post-analysis deletion, removable/offline media, NAS outage/recovery; use expendable fixtures and checksums before deleting offloaded originals.
- [ ] MAP-10 Verify clearing IQ/audio preserves findings through cleanup/restart; results deleted only through explicit results-clear action.
- [ ] MAP-11 Eight-hour combined-radio Mapper/analysis soak with bounded memory/storage, receiver release/reacquisition and no reset loops.

## AI, transcription, data review

- [ ] AI-01 Validate hallucination suppression on silence/static/non-speech while preserving weak genuine speech; measure false positives/abstentions.
- [ ] AI-02 Compare model presets for accuracy/speed/memory, confidence/output sanity, remote Ollama interruption/recovery.
- [ ] AI-03 Implement/evaluate an RF-trained classifier if feasible within the requested scope. Current LLM sees bounded DSP/decoder text metadata, not raw IQ/audio; do not describe example storage as automatic training.
- [ ] AI-04 Complete confirmed-sample dataset workflow: label correction, provenance, review, evaluation splits, actual training pipeline.
- [ ] AI-05 Complete scalable Explorer aggregation, checked/unobserved occupancy, hourly/day/week patterns, location/time playback, shared filters, evidence drill-down/bookmarks/views.
- [ ] AI-06 Regression-test RadioReference/offline import and Sheets: geographic verification, verified-only uploads, duplicates, source evidence and failure handling.

## Additional decoders, transmit, video, calibration

- [ ] DEC-01 Finish Pi DSD-FME/MBE installation after explicit upstream-notice review; real DMR/conventional P25 acceptance.
- [ ] DEC-02 Real-frame acceptance through GUI/API/live/deferred Mapper for DMR, rtl_433 sensors, ADS-B/Mode S, paging/signaling, ACARS, AIS and other advertised protocols; parser tests do not prove reception.
- [ ] TX-01 Controlled analog TX bench acceptance: modulation/filtering, frequency accuracy, level, duration/stop, receiver protection.
- [ ] TX-02 Microphone transmit implementation and physical acceptance.
- [ ] TX-03 Standardized digital voice/data transmit implementation and spectral/protocol acceptance.
- [ ] TX-04 Standardized image/fax/modem lab, send/receive/progressive display and stored-image review; safely attenuated setup.
- [ ] VID-01 Live analog NTSC/PAL FPV acceptance: picture lock, latency, quality; source tests alone are insufficient.
- [ ] VID-02 Digital FPV feasibility/implementation for a specific supported protocol, not generic frequency reception.
- [ ] CAL-01 Controlled calibrated sensitivity/antenna measurement workflow; ambient response is not absolute gain, sensitivity, SWR or range.

## Platforms, security, documentation, distribution

- [ ] PLAT-01 Android functioning RC: direct RTL USB/native P25 strategy, HackRF/audio/DSP, physical SD-card access, performance/thermal tests, packaging; no dead controls. Baseline is preview only.
- [ ] PLAT-02 iOS/iPadOS target: signing/USB-entitlement feasibility, remote Pi path, actual audio/storage/DSP/P25 tests; paid membership alone is not entitlement approval.
- [ ] PLAT-03 Mobile web UI end-to-end controls, authenticated LAN, reconnect, spectrum refresh, accessibility.
- [ ] PLAT-04 Pi power-HAT telemetry physical acceptance.
- [ ] DIST-01 Clean-machine install/upgrade/rollback/uninstall/startup/dependency prompts on Apple Silicon/Intel Mac, Windows 10/11, Linux AMD64/ARM64; packaged app tests, not only source.
- [ ] DIST-02 Secure update manifests/package verification, rollback, downgrade/replay protection; baseline update UI is not a secure finished channel.
- [ ] DIST-03 Linux/Windows secure credential editing/storage and safe migration; no secrets in logs/profiles/exports.
- [ ] DIST-04 Rebuild/verify clean dependency bundles; never publish contaminated old Java-cache artifacts. Detection fix exists; clean release rebuild remains.
- [ ] DIST-05 Synchronize wiki/setup/server/optional-component instructions, screenshots, feature reports, licenses/source offers with tested shipped versions; older reports contain stale claims.
- [ ] DIST-06 Public signing/notarization when suitable credentials are available; remain explicitly blocked until supplied/approved.

## Execution order and dependencies

Next release ordering requested September 16: start with tuner, Band Monitor and interface reliability (UI-01 through UI-07), then hardware support, decoders/AI, clear audio/P25, and Mapper completion. Prioritize broken major features over small enhancements. Pi remains the active deployment target; mobile ports remain paused. All acceptance gates below still apply.

Start with easy bounded fixes and regression tests, then work toward difficult hardware/architecture items. Core priorities: dependable P25 audio/HackRF lock, Mapper correctness/storage, comprehensive UI acceptance, multi-receiver operation. Do not let cosmetic completion substitute for RF acceptance. Listed mobile/TX/video items remain on this backlog but follow core reliability. User-dependent physical signals, notice acknowledgements, signing/entitlement requirements must be tracked honestly, not treated as passed.

## Evidence log

- 2026-09-16 decoder evidence follow-up MAP-01/02: reject empty decoder text and messages with blank protocol before live/deferred verification. Regression exercises empty ADS-B/rtl_433/DSD/paging evidence while retaining a nonempty frame. This is false-positive filtering, not physical protocol acceptance.

- 2026-09-16 post-rc34 work: RF-01 OP25 two-byte DRAIN/DROP control packets are no longer published as single-sample PCM. Test sends drain/drop/malformed packets before valid PCM and requires only the real audio frame. Full DROP queue semantics and per-talkgroup routing remain OPEN. MAP-01/AI-06 authoritative identification source/reason now survive subsequent unverified model/band guesses instead of allowing new provenance to inherit an old verification badge; regression added. No findings or media deleted.

- 2026-09-16 next-release start: Band Monitor automatic receiver/control updates now require its own active profile, including a second ownership check after debounce; Stop is disabled for unrelated sessions. Non-HackRF requests no longer send RF amp/LNA/VGA values. Receiver rate controls enforce reported minimum as well as maximum, and missing limits no longer disable all explicit rates. Regression script `Scripts/test_band_controls.cjs` covers ownership and control serialization; full physical GUI acceptance remains OPEN.

- 2026-09-16 transition follow-up, RF-01/UI-05: real RTL WFM→Pluto P25 test exposed stopped tuner audio bleed due to persistent-reader close semantics. Explicit cancellation fixes installed on Pi; fresh 12-second capture contains only OP25 8 kHz stream (211 frames, 3.803 seconds intermittent voice), control lock 774.45625 MHz, no stopped WFM audio. Unit/race/vet pass; broader all-mode listening acceptance remains OPEN. Evidence `/tmp/gpsdr-p25-clean-transition.bin`; implementation revision `b1ed5b8`.

- 2026-09-16 audio priority: UI-05/07 continuous analog DSP state retained across tuner/Band Monitor blocks; AM/NFM/WFM chunk equivalence and VFO-reset vectors pass. RF-01/UI-05 browser scheduled-source backlog/Stop/disconnect cleanup corrected with mock-context regression tests. Go unit/vet/race and JS syntax checks passed; Pi rc33-test updated. Live Pluto 98.1 MHz WFM delivered 701 complete 48 kHz PCM frames (17.525 seconds in startup-inclusive 20-second capture); no subjective intelligibility claim. UI-05/07 and RF-01/03/04 remain OPEN. Evidence and limitations in Pi status document.

- 2026-09-16 continuation: RF-02 raw captures on Pi reproduced c5cb pinned-sign corruption (100% negative I/Q); a447 returned centered samples but its isolated P25 test still searched. Evidence `/tmp/gpsdr-{c5cb,a447}-current.iq` and companion logs; no HackRF acceptance claimed. RF-04: corrected OP25 active-call metadata source, stale/control/encrypted filtering and discovered mixer entries; unit/vet/race checks passed, Pi rc33-test installed, real active TG 2436 visible in API. Receiver-stream PCM is not yet per-talkgroup routing; RF-01/02/04 remain OPEN. See Pi status document for numeric results and remaining gates.

- 2026-09-16: Pi made primary platform; PLAT-01/02 mobile implementation on hold. MAP-09/DIST-01: installed mount/writable-directory startup guard without replacing private base service. RF-02/03/UI-04: explicit Pi-only OP25 backend configuration, forwarded HackRF gain/amp and saved PPM calibration, 100 kHz HackRF DC-avoidance offset; software regression tests pass. Isolated OP25 HackRF initially still searched before offset change. Full hardware/audio/long-soak gates remain OPEN. See `PI_REMEDIATION_STATUS.md`.

- 2026-09-16: Backlog saved on explicit user request. No item marked complete merely by saving it. Current Pi-only test build includes OP25 audio/site/targeting changes and decoder refresh/selection/audio-port cleanup. Entire app is not yet accepted.

For each update append: date; item IDs; revision/build/host; commands or evidence paths (no secrets); software vs hardware results; failure/blocker; next acceptance step. Retain the full list across releases.
