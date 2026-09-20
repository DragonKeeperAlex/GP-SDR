# GP-SDR master remediation backlog

Approved by the user September 16, 2026. Saved from the complete outstanding-work overview in the GP-SDR task. Baseline source: `0fd40fc`.

## Mandatory working rule

September 17 instruction: prioritize hardware compatibility and the existing Pi
power-HAT integration, then acceptance of all currently implemented functionality.
Do not implement additional missing features (even those already backlogged)
until existing functionality is adequately repaired and tested. Retain missing
items below, but defer feature expansion.

Also read [categorized feature status](FEATURE_STATUS_CHECKLIST.md) at the start
of every work session until the user says otherwise. Update both documents when
evidence changes; check off only the specifically accepted scope.

**Resolve this backlog before implementing any new features outside it.** Missing features already listed here are in scope; unrelated feature expansion is frozen. Reference this file at the start of each GP-SDR work session and when handing work between tasks. It supersedes older roadmap completion claims and feature-gap reports where they disagree.

Keep stable item IDs. Do not remove an item because it is difficult or blocked. Record evidence, source revision, installed version/host, and remaining limitations before checking it off. A build, menu, running process, RF energy, or valid PCM alone is not end-to-end acceptance. Distinguish software/API tests from physical RF tests and subjective audio/video acceptance. User-dependent items remain open until their requirements are met; never silently waive them.

Current installation scope: Pi only. No new public release or Mac replacement unless requested. Preserve recordings, results, profiles, calibration, credentials, and recovery paths. Keep test audio out of the user's headphones. RF transmit acceptance requires an appropriate authorized, attenuated/dummy-load bench setup; user permission alone does not establish lawful operation.

September 17 hardware exclusion: damaged HackRF serial ending `c5cb` is excluded
from all testing and acceptance until the user independently retests it and
explicitly returns it to the pool. Do not spend remediation effort on that unit.
Keep its hardware issue deferred; use `a447`, RTL-SDR and Pluto for current work.

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
- [ ] AI-07 Tiered local→remote deep analysis: run basic DSP/decoders and lightweight model analysis on-device; optionally queue promising captures for a separately configured larger-model Ollama server when available. Persist queue across restarts/outages, allow configurable eligibility and manual review, deduplicate submissions, bound concurrency/storage and show progress/retry/cancel controls. Retain queued IQ until deep analysis succeeds or explicit discard; preserve both first-pass and deep results with model/server provenance. Current Ollama identification consumes derived metadata, not raw IQ: any IQ/audio upload or RF-model preprocessing must be explicit and capability-validated, with opt-in transfer/location controls. Accept with unavailable→available server, interruption/retry, restart and retention tests. Deferred until existing core reliability work is accepted.

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
- [ ] IMG-01 Receive-only NOAA APT/weather-image workflow, deterministic image fixtures, live-pass capture, and actual image acceptance; keep output separate from packet decoders.
- [ ] IMG-02 Receive-only digital SSTV for one explicitly selected mode, deterministic/negative fixtures, live-frame capture, and visual image acceptance; keep output separate from packet decoders.

## Platforms, security, documentation, distribution

- [ ] PLAT-01 Android functioning RC: direct RTL USB/native P25 strategy, HackRF/audio/DSP, physical SD-card access, performance/thermal tests, packaging; no dead controls. Baseline is preview only.
- [ ] PLAT-02 iOS/iPadOS target: signing/USB-entitlement feasibility, remote Pi path, actual audio/storage/DSP/P25 tests; paid membership alone is not entitlement approval.
- [ ] PLAT-03 Mobile web UI end-to-end controls, authenticated LAN, reconnect, spectrum refresh, accessibility.
- [ ] PLAT-04 Pi power-HAT telemetry physical acceptance, including the
  optional read-only Pi-local status display. Verify actual panel readability,
  orientation, refresh stability, PiPower values, receiver status and clean
  recovery without the panel; a created framebuffer/service alone is not
  acceptance.
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

- 2026-09-19 AI-02/transcription Pi runtime readiness: the installed Pi
  reports Ollama 0.34.0 active with lightweight `qwen2.5:1.5b` (Q4_K_M,
  approximately 986 MB). A direct structured JSON generation succeeded, and
  GP-SDR's `/api/local-ai` returned `ready` with bounded metadata-only mode.
  The existing whisper executable/model installation was incomplete because
  its shared libraries were absent; whisper.cpp was rebuilt for ARM64,
  installed under `/opt/gp-sdr`, and the service now carries an explicit
  library path and `GPSDR_WHISPER_MODEL` pointing to `ggml-tiny.en.bin`.
  `/api/integrations` now reports transcription `ready`; a direct tone-file
  smoke test completed successfully. This proves runtime readiness, not real
  speech intelligibility or AI-01 hallucination-rate acceptance.

- 2026-09-18 RF-02 current HackRF P25 repetition: a temporary receive-only
  copy of the known profile used the good HackRF `a447` at 5 MS/s, RF amp on,
  LNA 8 dB and VGA 0 dB. It locked EBRCS at 774.456250 MHz for 18 of 19
  consecutive one-second status observations and produced 178 OP25 PCM
  frames. The test profile was deleted and the runtime stopped afterward;
  service remained active and the Pi was unthrottled. This is repeatable
  control-lock/PCM evidence for the good HackRF, not subjective voice or
  talkgroup-mixer acceptance.

- 2026-09-18 UI-04/UI-05 Band Monitor applied-rate repair: HackRF channel-bank
  monitoring formerly imposed a 10 MS/s floor even when a narrower, valid
  capture rate was explicitly selected. Automatic wide-bank selection still
  prefers 10 MS/s, but an explicit 2–20 MS/s rate now applies when it covers
  the requested channel span. A temporary two-channel 98.1/98.3 MHz Pi run
  applied and reported 5 MS/s with LNA 8/VGA 0/amp off. It also verified the
  new Band Monitor receiver telemetry path; temporary profile and job were
  removed. Full audio/control user acceptance remains open.

- 2026-09-18 RF-02/UI-04 input-integrity diagnosis: direct receive-only
  HackRF captures initially showed a Q path near the negative rail at the
  EBRCS control channel, but controlled file/stdout repeats at the same 5 MS/s
  setting were centered and did not reproduce the fault.  This is therefore a
  transient capture observation, not a hardware-defect conclusion. GP-SDR now
  checks raw I/Q before DC/IQ correction, reports a distinct receiver-input
  warning if either component is rail-pinned for at least 95% of samples, and
  presents that separately from normal overload. Local full/race/vet checks
  passed and the Pi build was installed. A subsequent GP-SDR HackRF Tuner
  probe did not flag a pinned path. P25 control-lock/voice acceptance remains
  open.

- 2026-09-18 current Pi deployment: `go test ./...`, `go test -race
  ./internal/app`, `go vet ./...`, P25-focused regression and diff checks
  passed from the current source before cross-compiling the Pi arm64 binary.
  Binary SHA-256 `8be6e6fcd11b380fa5e7b004674873429e330a9bbc619e473e69debe9947b901`
  is installed with a timestamped rollback binary retained.  GP-SDR restarted
  active and Pi throttling was `0x0`; no profiles, findings, captures or
  calibrations were cleared.

- 2026-09-18 RF-02 current same-window comparison: an isolated good HackRF
  `a447` run of the known receive-only P25 profile started normally at its
  applied 5 MS/s but was still searching after eleven seconds.  Immediately
  afterward, the identical profile on the connected PlutoSDR locked
  774.456250 MHz at 2.4 MS/s and reported 284 OP25 PCM frames in the same
  eleven-second observation window.  Both runs stopped cleanly, GP-SDR stayed
  active, and Pi throttling remained `0x0`.  This is evidence against a
  general OP25/service outage; it does not establish a HackRF hardware defect
  or subjective voice-audio acceptance.

- 2026-09-18 RF-02/UI-04 HackRF P25 rate and gain evidence: the good HackRF
  `a447` was measured at the known EBRCS 774.45625 MHz control channel. RF amp
  on at LNA 32/VGA 24 produced about 12.3% clipped IQ samples, so the amp must
  remain off on this current input. Automatic HackRF P25 rate now uses 5 MS/s
  (explicit 8/10/20 MS/s choices remain available). A pre-deployment isolated
  5 MS/s test locked and delivered OP25 PCM; after deployment, two fresh
  15/35-second checks with the known profile stayed searching at 5 MS/s even
  after the antenna change. In the same session Pluto locked the identical
  control channel and delivered 910 8 kHz PCM frames. This is an honest
  configuration/overload repair plus an unresolved live HackRF lock result,
  not voice-quality acceptance.

- 2026-09-18 RF-02 continued controlled checks: the good HackRF `a447`
  started OP25 cleanly with the same known profile at 8 MS/s, and again at
  5 MS/s with LNA 40/VGA 32/amp off. Both completed a full control-list cycle
  while searching; the generated device configuration, process log and Pi
  throttling state showed no backend startup or resource failure. Do not make
  an untested gain/rate change to user profiles from this result. The remaining
  differentiators are RF source/antenna/feed placement, frequency correction,
  and a controlled same-antenna comparison against the already locking Pluto.

- 2026-09-18 RF-02 HackRF amp acceptance evidence: a temporary receive-only
  profile with the good HackRF RF amp on, LNA 8/VGA 0 and 5 MS/s locked the
  known 774.45625 MHz EBRCS control channel and delivered 1,336 OP25 8 kHz PCM
  frames. This validates a safe low-gain amp-on operating point for this
  antenna/source after the earlier high-gain clipping result. It does not make
  amp-on universally safe, alter saved user settings, or establish
  user-listened voice intelligibility.

- 2026-09-18 RF-02/RF-03 HackRF live-audio bridge evidence: the same
  low-gain amp-on 5 MS/s HackRF configuration locked the P25 control channel,
  reported 168 OP25 PCM frames, and delivered 15,048 bytes through the
  authenticated GP-SDR live-audio endpoint during an eight-second no-playback
  probe. Temporary profile and probe media were removed. This establishes the
  tested decoder-to-app PCM route, not browser/player intelligibility or
  per-talkgroup isolated audio.

- 2026-09-18 UI-09 bounded Spectrum Analyzer hardware evidence: a standalone
  receive-only good-HackRF 98–100 MHz run at 5 MS/s produced 104 sweep slices
  and 1,024 bins in seven seconds (peak -44.1 dBFS), with no analyzer error or
  Pi throttling. Clear reset sweeps/slices to zero while retaining the 1,024
  bin display, and stop completed cleanly. This verifies the basic analyzer
  data path and clear action; UI zoom/hover/export, multi-receiver merging,
  long CPU limits and disconnect behavior remain open.

- 2026-09-18 UI-05 Band Monitor receive-path repair: `StartOnDevice` now
  enables a runtime-only open monitor flag, never persists it in the profile,
  and the sequential survey loop publishes analog PCM plus receiver telemetry
  below the event/squelch threshold without creating false events or
  recordings. The deployed Pi binary SHA-256
  `bed2a72bb975133487ec1dd58dfa87214229ebe43de586daffef29256838549b`
  ran a temporary good-HackRF 98.1 MHz WFM channel-bank profile and delivered
  351,392 bytes of authenticated live audio in eight seconds. The temporary
  profile and test stream were deleted and the Pi remained unthrottled. This
  is stream delivery evidence; user-listened audio, AM/NFM and full control
  acceptance remain open.

- 2026-09-18 MAP-09 storage-hardware blocker: the Pi's active GP-SDR data
  mount `/dev/sda1` reported device-offline read/write failures and an exFAT
  forced read-only remount before later appearing mounted read/write again.
  The kernel explicitly requests an fsck after the improper unmount. Do not
  run repair against the mounted active data disk: stop GP-SDR, preserve a
  copy, unmount it, then perform an offline filesystem check before relying on
  it for long capture/analysis acceptance. This is a storage transport/media
  fault requiring a controlled maintenance window, not proof that Mapper
  cleanup deleted results.

- 2026-09-18 UI-05 bounded analog stream evidence: receive-only good-HackRF
  tuner sessions with amp off and open monitor produced 267,026 bytes of NFM
  PCM at 162.550 MHz (5 MS/s) and 267,024 bytes of AM PCM at 1.000 MHz
  (2 MS/s) over each eight-second authenticated live-audio probe. Earlier WFM
  Band Monitor evidence remains separate. No local output device was opened;
  these prove UI/API-to-PCM delivery and safe stop, not station intelligibility,
  squelch quality, recording, reconnect or per-receiver subjective acceptance.

- 2026-09-18 UI-03/UI-04 applied Band Monitor controls: a sequential
  one-channel monitor had ignored requested controls until the survey path was
  repaired. The Pi binary SHA-256
  `98d72cd9670db9ad0b8844d07a54491f5ea5839eba00a02a8e9164a5d13bca5f`
  now reports the requested 10 MS/s/LNA 16/VGA 8/amp off values in live
  telemetry. A repeat at 5 MS/s produced 495,242 bytes of live PCM and reported
  5 MS/s with the same gain/amp values. A direct six-second HackRF 5 MS/s
  stream also sustained about 10 MiB/s. The earlier blank 5 MS/s telemetry
  read was an early startup observation, not a persistent transfer failure.
  Temporary profiles and probe files were removed. Full cross-page capability
  and subjective control acceptance remain open.

- 2026-09-18 RF-02/RF-03 controlled HackRF amp check: the good HackRF `a447`
  locked the known 774.45625 MHz P25 control channel with RF amp enabled at
  the safe low-gain point (LNA 8 dB, VGA 0 dB, 5 MS/s), then delivered live
  OP25 PCM through GP-SDR without opening a local playback device. High-gain
  amp-on remains unsuitable for this input due to clipping. This is a tested
  profile-level starting point, not a universal amp-on default or subjective
  voice-quality acceptance.

- 2026-09-18 RF-02 fresh deployed HackRF amp result: a separate receive-only
  16-second run after the current deployment reported OP25 `locked` on
  774.456250 MHz at 5 MS/s with RF amp on, LNA 8 dB and VGA 0 dB. GP-SDR
  reported 122 8 kHz audio frames and `receiving`; the decoder note correctly
  stated that it was awaiting an enabled unencrypted call. This closes neither
  end-user voice intelligibility nor per-talkgroup mixing, but establishes a
  repeatable good-HackRF control-lock configuration.

- 2026-09-18 MAP-02 live-decoder retention repair: live file-based decoders
  (`dump1090`, `rtl_433`, and AIS) now receive a restrictive-permission,
  OS-temporary IQ file even when a profile disables archival IQ; it is never
  linked to the event and is removed after the decoder exits. Explicit
  decoder-tagged channels now use finite-file dwell windows appropriate for
  frame acquisition (ADS-B 2 s; rtl_433/AIS/ACARS 3 s; DSD-FME/multimon-ng
  2.5 s), while generic analog scanning remains 450 ms and Mapper/range timing
  stays user-controlled. Local full tests/vet pass and Pi binary SHA-256
  `6ee854889b9f086ac6b0425fb59bfafc9f376991a68022c0d2575791983097a5`
  is installed. A real RTL ADS-B check exercised this no-retention path but
  did not obtain a valid Mode-S frame from the current antenna/location, so
  payload acceptance remains open. The direct `rtl_test` claim error during
  diagnosis was expected because GP-SDR's active persistent `rtl_tcp` child
  owns the USB interface; it is not evidence of a new RTL application fault.

- 2026-09-18 UI-05/DEC-02 tuner decoder-path repair: Tuner now batches complex
  IQ for the file-oriented `dump1090`, `rtl_433`, and AIS decoders, using the
  same private temporary-file lifecycle rather than silently offering a mode
  that cannot receive IQ. Only one decode batch may be in flight; additional
  live samples are discarded while busy, preventing an unbounded memory queue.
  Pi binary SHA-256
  `cd55ed93eb955bdcd9bd6ac49e3381941dab1ce60fa764fd116df5546071234a`
  was installed and an RTL 1090-MHz ADS-B Tuner session ran with no runtime
  error or audio-device access. No Mode-S frame was decoded in that short real
  RF window, so payload/audio acceptance remains open.

- 2026-09-18 UI-05 bounded cross-receiver Tuner evidence: receive-only WFM
  probes at 98.1 MHz delivered 5,275,648 live PCM bytes from RTL-SDR and
  532,480 bytes from PlutoSDR through GP-SDR's authenticated endpoint. Both
  tuner sessions started and stopped cleanly without local speaker/headphone
  access. This expands hardware stream-path coverage; it does not establish
  subjective station intelligibility, gain/squelch tuning, recording or
  reconnect acceptance.

- 2026-09-18 DEC-02/RF-11 isolated ADS-B check: with GP-SDR stopped solely to
  release its persistent RTL helper, the RTL-SDR captured 83,099,648 bytes of
  real 1090 MHz IQ at 2.4 MS/s for 18 seconds. `dump1090-fa` produced zero
  Mode-S frames from that capture; GP-SDR then restarted active and the Pi
  remained unthrottled with no new USB error. This separates the present
  no-frame result from the GP-SDR temporary-IQ or decoder invocation path. It
  is most consistent with no decodable aircraft signal at the current antenna/
  placement/window; keep ADS-B real-frame acceptance open and retest with a
  known-good 1090 MHz antenna/source before changing decoder logic again.

- 2026-09-18 DEC-02 isolated rtl_433 check: a separate 15-second RTL-SDR
  capture at 433.920 MHz (68,943,872 bytes at 2.4 MS/s) produced no rtl_433
  JSON frame. GP-SDR restarted active afterward; HackRF, RTL-SDR and PlutoSDR
  were all API-connected/available and the Pi remained unthrottled. This is a
  clean no-sensor observation for the current RF environment, not a decoder
  crash or a reason to label a candidate as a confirmed sensor signal.

- 2026-09-18 MAP-09/10 orphaned-snapshot recovery: inspection showed the Pi
  Data directory contained six unheld GP-SDR atomic-write temp files totaling
  391,206,174 bytes; a sampled 65 MB temp matched the canonical
  `mapper-records.json` checksum. Active writes were distinguished with
  `fuser` and allowed to complete before restart. Startup now removes only
  regular `.gpsdr-write-*.tmp` files in the Data directory before writers run,
  never follows symlinks, and has a regression proving canonical/nonregular
  files survive. Pi binary SHA-256
  `f1339d291a8964ad3e3c1c3d90e4f7712d94d465b77585e8fa7b01feea27d3ac`
  installed with rollback copy; orphan count became zero. Event history grew
  from 5,332 to 5,381 rows, Mapper results remained 5,000 records, and
  GP-SDR-owned storage fell from about 582 MB to 256 MB. The known Pluto P25
  profile was restored, locked 774.45625 MHz and received PCM. This validates
  orphan-temp cleanup only; it does not authorize deletion of results/history
  or close the full storage-policy/NAS acceptance items.

- 2026-09-18 RF-11 concurrent-radio soak failure: a receive-only RTL-SDR
  `rtl_test` soak at 2.4 MS/s ran alongside an already locked Pluto P25
  session, then ended with `librtlsdr` error `-5` and at least 180 lost bytes.
  At the same time the Pi kernel recorded an upstream USB over-current event
  and disconnected the hub plus attached SDRs. GP-SDR remained active and the
  Pi stayed unthrottled; the PiPower5 reported stable external input
  (14.812 V) and output (5.246 V) afterward. Devices re-enumerated and OP25
  safely changed from locked to searching rather than retaining a false lock.
  This strengthens the case for a hub/cable/device USB-power fault rather than
  an application decoder fault. Do not deploy/restart during recovery; inspect
  the physical USB power path before calling RF-11 accepted.

- 2026-09-18 RF-11 escalation: the same hub branch subsequently repeated the
  over-current/disconnect sequence and the Pi's USB inventory fell back to
  root hubs plus its card reader; all SDRs disappeared. GP-SDR then became
  inactive rather than displaying a false healthy receiver state. No service
  restart or additional RF test was attempted against an absent bus. This
  blocks hardware acceptance pending a physical hub/cable/power/device-path
  inspection and stable re-enumeration.

- 2026-09-18 UI-02 remote-receiver empty-state repair: a read-only installed
  API sweep found `/api/remote-receivers` returning JSON `null` when an older
  empty store contained `null`, unlike the array-shaped contract used by the
  Hardware UI. The store now normalizes legacy `null` input and always returns
  `[]` for no configured remote radios; a regression covers the persisted
  legacy case. Local application tests and vet pass. Pi deployment is pending
  a bounded concurrent RTL/Pluto soak so the active P25 receiver is not
  needlessly restarted.

- 2026-09-18 UI-02 remote-receiver deployment and recovery: after the user
  physically reconnected the SDR branch, the Pi again enumerated RTL-SDR,
  good HackRF and Pluto. Installed the ARM64 build with SHA-256
  `9d01bbcac57fe9613cca878967a1386eb6f047bd41be21269900670e3f6744ab`
  and retained a binary rollback copy. The service returned active, all three
  devices reported connected/available, and `/api/remote-receivers` returned
  an empty JSON array. The known receive-only Pluto profile was then started
  and relocked at 774.45625 MHz with OP25 PCM frames arriving. No browser or
  local audio playback was enabled. The physical hub fault remains open.

- 2026-09-18 RF-01/UI-02 browser and bounded live-audio continuation: after a
  fresh GP-SDR browser reload, the P25 workspace visually exposed the explicit
  user-gesture **Enable audio** control, a locked Pluto/OP25 session at
  774.45625 MHz, ten live talkgroups, and a rising OP25 PCM-frame count. A
  separate 15-second local no-playback probe advanced from 8,087 to 8,131
  received frames while the control channel remained locked; the stream was
  `receiving` during a burst and `idle` when it ended. GP-SDR remained active,
  the Pi reported `throttled=0x0`, and no fresh USB reset/disconnect or
  over-current event appeared. Browser audio was intentionally not enabled,
  so this remains delivery/UI evidence, not subjective speech-intelligibility
  or per-talkgroup-mixer acceptance.

- 2026-09-18 RF-02/UI-04 P25 per-device capture-rate repair: a shared P25
  profile stored at 2.4 MS/s was being written unchanged into a HackRF OP25
  configuration, even though that rate is not a supported HackRF P25 capture
  setting. OP25 generation now resolves a rate per assigned receiver: HackRF
  auto 10 MS/s (or explicit supported 5/8/10/20 MS/s), RTL-SDR auto 2.4 MS/s,
  and Pluto retains a valid saved rate. Unit/application/vet checks passed;
  the Pi good HackRF `a447` was started receive-only and the generated config
  plus live status both reported 10 MS/s. It searched for 18 seconds without
  control lock, then the known Pluto profile was restored and relocked at
  774.45625 MHz. This fixes an invalid configuration path, not HackRF RF
  acceptance or clear voice playback.

- 2026-09-18 RF-07/UI-02 P25 failure-lifecycle repair: OP25 could exit after
  a receiver/USB failure while the Runtime still claimed an active session,
  leaving the UI's hardware inventory stale and preventing a clean retry.
  The P25 monitor now stops the stale Runtime session and retains an explicit
  runtime error when OP25 exits; a regression simulates that exit. During the
  receive-only RTL/Pluto switching test, the Pi kernel reported a USB hub
  over-current event and temporarily disconnected all three SDRs. This was a
  physical bus/power event, not an application panic. After re-enumeration,
  GP-SDR again acquired the Pluto control lock at 774.45625 MHz. Long-term
  USB-power stability and P25 recovery under actual disconnect remain open.

- 2026-09-18 RF-11 bounded parallel-radio continuation: while Pluto remained
  P25-locked, the connected RTL-SDR/E4000 completed a direct 2.4 MS/s stream
  soak with no kernel USB reset/disconnect/error recorded, stayed API-available
  afterward, and Pi throttling remained `0x0`. This is a short direct-stream
  health check only; it does not close the prior long-duration/hub/cable dropout
  investigation.

- 2026-09-18 RF-01/UI-02 P25 status/audio continuation: Pi OP25 was started
  with the dedicated Pluto hardware-test profile and acquired EBRCS control
  lock at 774.45625 MHz. Its status reported 4,684 received 8 kHz PCM frames;
  a bounded local read of the authenticated live-audio endpoint accumulated
  417,792 bytes without routing sound to any local output. The P25 workspace
  now disables/relabels Start while the decoder is searching or locked and
  displays decoder PCM state/frame count. This is hardware lock and stream
  delivery evidence, not subjective browser playback/intelligibility
  acceptance; the user must refresh the page and enable/listen to browser audio
  during a real unencrypted call.

- 2026-09-18 PLAT-04 Pi-local display: identified the attached SHCHV/LCDWIKI
  2.4-inch 320x240 panel as ILI9341/XPT2046 from its published pin map and
  matched it to the Pi SPI wiring: CE0 LCD, CE1 touch, GPIO22 D/C, GPIO27
  reset. PiPower5 remained available on I2C address 0x5c; no GPIO collision was
  found. Installed a boot-time FBTFT framebuffer overlay and an enabled,
  read-only two-second GP-SDR dashboard service. It recovered after reboot,
  `/dev/fb0` changed across refreshes, and Pi `throttled=0x0`; GP-SDR did not
  receive controls or transmit requests from the display. Physical panel
  readability/orientation and unplug/recovery remain open. The reboot exposed
  an unrelated data-volume USB disconnect (`sda` offline with exFAT I/O
  errors); GP-SDR was stopped before further writes and the temporary fstab
  change was reverted. The user must reconnect/repair that volume before
  capture service can safely resume.

- 2026-09-18 PLAT-04 display interaction/telemetry continuation: changed the
  Pi-local display default to a one-second Overview refresh with manual page
  navigation and a 30-second automatic return to Overview. Added local CPU
  load/temperature, V3D clock, RAM, mounted data-volume capacity, IPv4 address,
  and RX/TX-rate reporting. The display still performs no GP-SDR receiver or TX
  control. Touch calibration and guarded reboot/shutdown actions remain open
  pending physical touch-coordinate acceptance; they must not be guessed from
  an uncalibrated resistive panel.

- 2026-09-17 UI-04 effective Tuner readout: live receiver telemetry now includes the effective generic gain and the Tuner status displays applied sample rate plus either HackRF LNA/VGA/amp or generic gain for the actual active device. Values originate after request normalization/calibration, not merely the untouched form. UI regression, JavaScript syntax/audio-queue, application unit, and vet passed. Pi test18 installed with SHA-256 `ced3f1c4e8ca8c6e98ec5ff282091c3c2fc15c633e04f30d5b440ff7af856123`, rollback `/home/sdr/gpsdr-before-pi-test18-MSVD6l`; Pluto OP25 relocked 774.45625 MHz, service active/no throttle. Driver readback and real browser visual acceptance remain open.

- 2026-09-17 UI-03 tuner capability serialization repair: while Tuner/Live hid HackRF-only controls for RTL/Pluto, the tuner request still serialized LNA/VGA/RF amp/bias settings for every receiver. Requests now force those fields to safe neutral values unless the selected device is HackRF; the existing visibility behavior plus request serialization are covered by regression. JavaScript syntax/audio queue, application unit, and vet passed. Pi test17 installed with SHA-256 `266e105032b9a3ee18f83459fa6b8e27f0f68709ffb0f0bc624c1bae47304dbd`, rollback `/home/sdr/gpsdr-before-pi-test17-BEAl1v`; Pluto OP25 relocked 774.45625 MHz and service/no-throttle checks passed. Cross-page browser click-through and actual applied-control telemetry remain open.

- 2026-09-17 RF-11 bounded RTL continuation: with Pluto P25 concurrently locked, direct RTL-SDR/E4000 receive at 98.1 MHz, 2.4 MS/s and 29 dB gain completed a 60-second bounded soak with normal timeout exit, no USB reset/error, GP-SDR service remaining active, and Pi `throttled=0x0`. Current Pi memory is healthy (6.2 GiB available); a historical kernel OOM record concerns a separate Codex process and is not evidence of an active GP-SDR fault. This rules out only an immediate transfer failure; direct/hub/cable comparison and long combined-radio soak remain open.

- 2026-09-17 RF-01/UI-02 P25 live-settings restart repair: changing capture rate or P25 gain/amplifier controls while a session ran could restart the profile using a stale receiver-dropdown value. The UI now passes the runtime `receiverDeviceIDs[0]` to the restart and has a regression requiring that behavior. Node syntax/audio-queue, application unit, and vet checks passed. Pi test16 installed with SHA-256 `1251368a469593c05a9914df85cea0e70b4a0c4bd0e862d15d95daa2f3852357`, rollback `/home/sdr/gpsdr-before-pi-test16-oMA3Py`; Pluto OP25 relocked 774.45625 MHz after install, service active/no throttling. The actual browser control click remains to be accepted; no user configuration was changed during this verification.

- 2026-09-17 RF-01/UI-02 P25 audio-status continuation: browser live-audio status now distinguishes a successful stream connection waiting for an enabled voice call from actual received PCM and displays the active PCM rate on first frame. The 8 kHz/160-sample queue-continuity regression passes. A 25-second direct no-playback Pluto OP25 stream capture during live traffic delivered 378 `p25-stream-0` frames / 60,480 samples / 7.56 seconds, RMS 690.3, peak 15351; this is robust delivery evidence but still not subjective listening acceptance. Pi test15 installed with SHA-256 `f8b04edc0c9766cc1f4196af0f2d77970baeb570801d26b681d961d79221ec2c`, rollback `/home/sdr/gpsdr-before-pi-test15-Odnzxy`, service active/no throttling and control lock reacquired at 774.45625 MHz.

- 2026-09-17 RF-01/UI-02 P25 runtime-receiver continuation: the P25 page previously had no authoritative live receiver field, so a stale dropdown selection could expose the wrong controls after a session started. `P25Status` now returns only connected/live assigned receiver IDs and the UI uses that runtime value for its receiver label, sample-rate options, and capability controls. Regression covers a connected Pluto plus unavailable RTL assignment. Pi test14 installed with binary SHA-256 `d2923c0aeb6458b084b74d49dbaa9c8d68523036ff83e9d97d41257335652e5c` and rollback `/home/sdr/gpsdr-before-pi-test14-g3Usf3`; service active, no throttling, authenticated OP25 returned Pluto ID and relocked 774.45625 MHz. A bounded no-playback stream read had non-silent PCM (one 4,000-sample frame, RMS 4160.4) while OP25 reported no enabled unencrypted call. This proves status assignment and limited PCM delivery, not clear voice, full UI click-through, or per-talkgroup audio routing.

- 2026-09-17 RF-01/UI-01/02 P25 repair: reproduced browser exception `profile.deviceAssignments is null`, which interrupted rendering and left stale P25 state. UI now normalizes all legacy profile arrays before rendering and guards profile-card counts. P25 Start initializes live audio during the user click before requesting receiver start. Pi test12 installed with rollback `/home/sdr/gpsdr-before-pi-test12-US4Jw1`; authenticated P25 API reacquired Pluto/OP25 control lock 774.45625 MHz. A bounded 15-second no-playback endpoint read parsed 191 valid frames / 30,560 samples, RMS 553.4. This confirms delivery, not audible intelligibility, complete GUI acceptance, or per-talkgroup routing. Damaged c5cb not used.

- 2026-09-17 MAP-05/09 group-retention continuation: moved deferred IQ finalization after successful combined-group analysis; only successfully processed files enter combination. Group-model failures mark affected files error and preserve IQ, adjusting completed/failed counters. Cancellation before cleanup restores processed files to pending. Regression simulates successful first model call followed by group HTTP 503 and verifies IQ/status/counters. Status-write failure handling and explicit failed-group retry controls remain open; no production deployment yet.

- 2026-09-17 MAP-05/09/AI-01/02 continuation: deferred decoder errors and ready/configured transcription errors now propagate before retention cleanup. Unconfigured optional transcription logs a skipped stage. Group-model errors appear in analysis LastError/log rather than a false success banner; disabled AI does not fail the optional stage. Added configured-missing-model IQ-preservation and group HTTP 503 regressions; local race suite/vet pass. Group retention/retry after individual-file cleanup, persistence-status errors and real decoder/speech acceptance remain open. Pi installed build remains test10 pending accumulated deployment.

- 2026-09-17 AI-07: user requested local lightweight first-pass analysis plus queued notable samples for a larger remote Ollama server. Requirements saved only; no implementation, server configuration, transfers or running-session changes.

- 2026-09-17 MAP-05/09/AI-02 analysis failure handling: enabled local-model errors now fail the capture before retention cleanup instead of being ignored; disabled AI remains optional. Corrupt saved WAV now returns an explicit read failure. Regression verifies HTTP 503 preserves IQ and corrupt audio fails. Local full race suite/vet passed. Deployment and Pi acceptance pending; transcription/decoder error propagation and group-level model failures remain open.

- 2026-09-17 MAP-05/09/UI-02 persistence continuation: Mapper Save/Create/Delete now persist proposed job state before committing in-memory changes, returning disk errors rather than phantom success. Background jobs/results snapshot failures populate Mapper error status. Local race suite/vet passed; deterministic write-failure regressions passed on actual Pi exFAT with disposable fixtures. Full offline-media recovery and GUI acceptance remain open; no user data removed.

- 2026-09-17 UI-03/PLAT-04/RF-02/11: Pi test7 installed. Native HackRF/RTL channel metadata corrected; Pluto remains driver-reported 1/1, not assumed dual-channel. Three fresh PiPower5 physical readings and watts conversion verified; signed battery current preserved, isolated Python import tested from shadowing directory. HAT failure/recovery UI regression passes, as do unit/race/vet and audio/Band Monitor regressions. c5cb still corrupt; a447 centered IQ; RTL 20-second transfer lost 28 bytes but stayed connected. Pluto restored/locked 774.45625 MHz. No TX/audio playback. Full battery-source switching, long RTL soak and GUI/listening acceptance remain OPEN. See Pi status/checklist for exact evidence.

- 2026-09-17 batch UI-02/DIST-01/03/TX-01: profile Unicode name/copy bounds and failed Save/Import/Duplicate/Delete state fixed; safe filename IDs block store escape; total WAV multipart body bounded. Unit/race/vet and JS syntax pass. VPN/node1 access restored, Pi test6 deployed with authenticated health check and rollback snapshot. Installed test5 long-profile save/duplicate/resave and artifact cleanup succeeded; test6 unsafe-ID API rejection succeeded. Broader GUI/physical acceptance remains open.

- 2026-09-17 UI-02/DIST-01: fixed bundled P25 name versus 80-byte validation mismatch using 160-character Unicode-aware backend/editor limit and bounded copy names. Profile Save/Import/Duplicate/Delete no longer publish in-memory success on disk failure. Regression added. Remote test5 deployment blocked by node1 SSH timeout; keep Pi acceptance open until actual install/lock/GUI checks succeed.

- 2026-09-16 TX-01/DIST-01 Pi test4 deployment: validate WAV before saving and reject truncated chunks/partial PCM frames; unit/race/vet and JS regressions passed. Accumulated lab/provenance/ownership fixes installed as `1.5.0-rc34-pi-test4`; rollback binary/Data/Profiles snapshot retained at `/home/sdr/gpsdr-before-pi-test4-H6JZiN`. Existing Pluto P25 profile restored, DSD-FME discovery ready, benchmark UI/API served. No RF TX. End-to-end GUI/listening/TX acceptance remains open.

- 2026-09-16 TX-01/AI-04 continuation: user deferred physical TX testing. Added Transmit benchmark manifest library/API and rejected parallel/busy waveform generation before allocation/file writes. Library and busy-path regression tests added. No production receiver interrupted and no RF emitted. Physical RF acceptance, paired orchestration and training remain open.

- 2026-09-16 TX-01/AI-04 lab foundation: fixture generation now avoids full ideal-I/Q float arrays, rejects nonfinite parameters/duration, avoids output collisions and stores schema/generator/seed/synthetic eligibility. Local unit/race/vet and audio/Band Monitor JS regressions pass. Isolated Pi ARM64 lab-test3 API generated all ten fixture families without RF, stored evidence on the mounted data disk, and exited without changing production. Paired TX/RX orchestration, independent decode/spectral comparison, dataset training and physical bench acceptance remain OPEN. See `FEATURE_STATUS_CHECKLIST.md` for mandatory categorized inventory.

- 2026-09-16 decoder evidence follow-up MAP-01/02: reject empty decoder text and messages with blank protocol before live/deferred verification. Regression exercises empty ADS-B/rtl_433/DSD/paging evidence while retaining a nonempty frame. This is false-positive filtering, not physical protocol acceptance.

- 2026-09-16 post-rc34 work: RF-01 OP25 two-byte DRAIN/DROP control packets are no longer published as single-sample PCM. Test sends drain/drop/malformed packets before valid PCM and requires only the real audio frame. Full DROP queue semantics and per-talkgroup routing remain OPEN. MAP-01/AI-06 authoritative identification source/reason now survive subsequent unverified model/band guesses instead of allowing new provenance to inherit an old verification badge; regression added. No findings or media deleted.

- 2026-09-16 next-release start: Band Monitor automatic receiver/control updates now require its own active profile, including a second ownership check after debounce; Stop is disabled for unrelated sessions. Non-HackRF requests no longer send RF amp/LNA/VGA values. Receiver rate controls enforce reported minimum as well as maximum, and missing limits no longer disable all explicit rates. Regression script `Scripts/test_band_controls.cjs` covers ownership and control serialization; full physical GUI acceptance remains OPEN.

- 2026-09-16 transition follow-up, RF-01/UI-05: real RTL WFM→Pluto P25 test exposed stopped tuner audio bleed due to persistent-reader close semantics. Explicit cancellation fixes installed on Pi; fresh 12-second capture contains only OP25 8 kHz stream (211 frames, 3.803 seconds intermittent voice), control lock 774.45625 MHz, no stopped WFM audio. Unit/race/vet pass; broader all-mode listening acceptance remains OPEN. Evidence `/tmp/gpsdr-p25-clean-transition.bin`; implementation revision `b1ed5b8`.

- 2026-09-16 audio priority: UI-05/07 continuous analog DSP state retained across tuner/Band Monitor blocks; AM/NFM/WFM chunk equivalence and VFO-reset vectors pass. RF-01/UI-05 browser scheduled-source backlog/Stop/disconnect cleanup corrected with mock-context regression tests. Go unit/vet/race and JS syntax checks passed; Pi rc33-test updated. Live Pluto 98.1 MHz WFM delivered 701 complete 48 kHz PCM frames (17.525 seconds in startup-inclusive 20-second capture); no subjective intelligibility claim. UI-05/07 and RF-01/03/04 remain OPEN. Evidence and limitations in Pi status document.

- 2026-09-16 continuation: RF-02 raw captures on Pi reproduced c5cb pinned-sign corruption (100% negative I/Q); a447 returned centered samples but its isolated P25 test still searched. Evidence `/tmp/gpsdr-{c5cb,a447}-current.iq` and companion logs; no HackRF acceptance claimed. RF-04: corrected OP25 active-call metadata source, stale/control/encrypted filtering and discovered mixer entries; unit/vet/race checks passed, Pi rc33-test installed, real active TG 2436 visible in API. Receiver-stream PCM is not yet per-talkgroup routing; RF-01/02/04 remain OPEN. See Pi status document for numeric results and remaining gates.

- 2026-09-16: Pi made primary platform; PLAT-01/02 mobile implementation on hold. MAP-09/DIST-01: installed mount/writable-directory startup guard without replacing private base service. RF-02/03/UI-04: explicit Pi-only OP25 backend configuration, forwarded HackRF gain/amp and saved PPM calibration, 100 kHz HackRF DC-avoidance offset; software regression tests pass. Isolated OP25 HackRF initially still searched before offset change. Full hardware/audio/long-soak gates remain OPEN. See `PI_REMEDIATION_STATUS.md`.

- 2026-09-16: Backlog saved on explicit user request. No item marked complete merely by saving it. Current Pi-only test build includes OP25 audio/site/targeting changes and decoder refresh/selection/audio-port cleanup. Entire app is not yet accepted.

For each update append: date; item IDs; revision/build/host; commands or evidence paths (no secrets); software vs hardware results; failure/blocker; next acceptance step. Retain the full list across releases.

- 2026-09-18 MAP-09/10 storage-policy continuation: added a separately
  configurable 128 MB default cap for `Data/capture-intervals.jsonl`, the raw
  per-capture timing diagnostic journal. Compaction is serialized with active
  capture appends, retains newest complete JSONL records, atomically replaces
  the journal, and is disabled by a zero cap. It cannot target
  `mapper-records.json`, event history, transcripts, profiles, calibration, or
  channel data. Existing policies migrate to the safe 128 MB default. Local
  full Go suite, vet, and diff checks passed. Pi binary SHA-256
  `af53619468d742f71a190a7745f822b55803947fe5cf3def895ff390016af3c0`
  installed with a rollback binary; service active, warning journal empty.
  Existing capture journal remained 108,393,477 bytes, mapper records
  65,201,029 bytes and events 3,692,748 bytes because no cleanup was invoked.
  GUI/manual-cleanup acceptance remains open.

- 2026-09-18 MAP-09/10 UI clarity follow-up: Storage now labels the manual
  action “Clean eligible data” and its confirmation explicitly names the three
  eligible categories (recordings, IQ evidence, capture diagnostics) and the
  protected categories (results, event history, transcripts, profiles,
  calibration, channel data). Completion reports trimmed journal-record count
  when applicable. Local full Go suite/vet/diff checks passed. Updated Pi
  binary SHA-256 `f7511e183e9cf763f99cde7ba21b87a3634391ae77acff0f4927f7e22942e568`
  installed with a rollback binary; service active and warning journal empty.

- 2026-09-18 MAP-09/10 torn-write hardening: capture-journal compaction now
  excludes an unterminated final line rather than preserving a possible partial
  JSON record after interrupted storage. Regression proves only complete newest
  records remain. Local storage-policy tests/vet/diff checks passed; Pi binary
  SHA-256 `50d5056d6e1e63c36f73d7c88bf4933aa36206b732d2ffc130329845854d058b`
  installed with a rollback binary, service active and warning journal empty.

- 2026-09-18 MAP-09/10 post-deploy preservation check: after the two bounded
  service restarts, the mounted Pi data volume still contained 89,601 Mapper
  records, 5,593 event rows, 11 profiles, and the unchanged 108,393,477-byte
  capture journal. Service was active and Pi throttling remained `0x0`. This
  confirms this deployment did not clear user findings; it is not a full
  long-run storage/NAS acceptance test.

- 2026-09-19 UI-01/UI-02 Live-workspace restructuring: Live now summarizes
  every active receiver session, Mapper job, and deferred-analysis run in one
  combined task board, with an explicit Open action and a scoped Stop action
  for each running task. The duplicate Live receiver controls are retained
  only as a collapsed quick-control section while a one-frequency Live session
  is actually running, preserving existing bindings without presenting Live as
  a second Tuner. P25, Digital voice, and Signal data workspaces now have
  direct sidebar entries; RF relay is a clearly non-operational placeholder
  for the future lab box. Go suite/vet/diff and an embedded-web regression
  passed. Pi deployment and rendered-UI acceptance are recorded separately.

- 2026-09-19 UI-04/UI-05 Band Monitor state clarity: the page now renders a
  distinct live applied-state line from receiver telemetry, rather than
  presenting only requested form values. A temporary receive-only GMRS band
  monitor on good HackRF `a447` started through the installed Pi API with an
  explicit 10 MS/s, LNA 8 dB, VGA 0 dB, amp-off request. Five seconds later
  telemetry reported the same device and applied state with no raw-I/Q warning.
  It was stopped immediately; no live-audio endpoint or output device was
  opened. This is hardware-control propagation evidence, not user-listened
  Band Monitor audio, CTCSS, mixer, or full browser interaction acceptance.

- 2026-09-19 UI-04/UI-05/RF-11 bounded Band Monitor receiver matrix: the
  exact built-in `GMRS · Whole Band` profile handed off from good HackRF `a447`
  (10 MS/s) to Pluto (24 MS/s); after the handoff, live telemetry reported
  Pluto rather than the previous receiver. Attempting the same whole-band
  profile on RTL-SDR was correctly rejected as too wide and left the live
  Pluto monitor intact. The earlier apparent RTL success came from testing the
  separate California repeater profile, which is a sequential profile rather
  than the whole-band channel bank. All test sessions were receive-only and
  stopped; this does not establish Band Monitor audio or long USB stability.

- 2026-09-19 RF-01/RF-02/RF-11 bounded P25 receiver matrix: on `Pluto P25
  hardware test`, Pluto locked for 8/12 observations and produced 210 OP25
  PCM frames without opening audio playback. The same profile did not lock on
  good HackRF at 5 MS/s during this pass, including a temporary amp-on/LNA-8/
  VGA-0 test profile, which was deleted afterward. RTL-SDR with a known EBRCS
  profile exited its SDRTrunk process after startup (`exit status 1`) before
  reporting an assigned receiver. This leaves HackRF and RTL P25 acceptance
  open; the next diagnosis must capture the current per-run SDRTrunk launch
  diagnostic rather than relying on stale logs.

- 2026-09-19 RF-01/RF-02 P25 RTL follow-up: the earlier RTL finding was
  corrected with a fresh receive-only installed-Pi run using the right
  `profileID` API field. OP25 attached to `rtl=0` at its applied 2.4 MS/s,
  initialized the E4000 tuner and completed multiple configured EBRCS control
  channel timeouts without a process crash. It did not lock during that
  bounded rotation window, so this is decoder/device-start evidence, not P25
  reception or voice acceptance. OP25 exit status now retains up to three
  concise log-tail lines in P25 status rather than discarding the root cause.
  The local focused test suite and vet passed; Pi arm64 binary SHA-256
  `4212c32d73d536bf549e409aecc4b844323b34fed0c58e0b7a1008cc8b777570`
  was installed only while idle with a timestamped rollback copy. Service
  restarted active and `throttled=0x0`; no user data was modified.

- 2026-09-19 RF-02 short installed-profile comparison: receive-only eight
  second runs of `Pluto P25 hardware test` used the currently connected good
  HackRF `a447` and Pluto without changing saved profiles. Pluto reported a
  decoded control lock in 5/8 observations; HackRF stayed running but had no
  control lock or PCM frames. This confirms the current user-visible P25 gap
  is HackRF lock/reception rather than a general application launch failure.
  It is not subjective voice acceptance and leaves RF-02 open.

- 2026-09-19 UI-04 P25 applied-state visibility: P25 status and workspace now
  expose actual session front-end state, including HackRF LNA/VGA/RF-amp or
  RTL/Pluto gain. An installed Pi receive-only P25 probe returned `Pluto PGA
  45 dB` at 2.4 MS/s and stopped afterward. Local full Go suite/vet/diff
  passed; arm64 binary SHA-256
  `ff5ae3a5f80e8eec14e11d8d0715b00a54577cdd993d578f574f126d860ce902`
  was installed while idle with a timestamped rollback. This verifies status
  reporting, not browser click or subjective media acceptance.

- 2026-09-19 RF-02 diagnostic boundary: receive-only 774.456250 MHz NFM
  tuner probes measured good HackRF at approximately 1 dB above its local
  noise estimate with amp-on/LNA-8/VGA-0, while Pluto's generic tuner path
  reported its floor even though OP25 locks that same system. Therefore those
  generic tuner levels are not a valid cross-device P25 sensitivity benchmark.
  No saved gain, PPM, antenna or profile setting was changed from this
  inconclusive comparison; use a controlled common-feed or known calibrated
  signal before assigning a receiver fault.

- 2026-09-19 UI-01 P25 responsive metric repair: the fixed five-column P25
  metric strip now uses an adaptive minimum-width grid and wraps applied
  front-end text, avoiding clipping with the added gain/amp status and on
  narrow clients. Local full Go suite/vet/diff passed; Pi arm64 binary
  SHA-256 `2a172362766aabfa9820a07a01f37647caa12cb01f32f1b43995d4c7f8b8b4be`
  was installed while idle with rollback retained. Service came back active
  and unthrottled. Rendered multi-size browser acceptance remains open.

- 2026-09-19 UI-04 multi-receiver activity attribution: Hardware cards no
  longer label every connected SDR as `Streaming` whenever any runtime is
  active. Mapper ownership, P25 live receiver IDs, and receiver telemetry now
  determine the specific active card. A web regression prevents reintroducing
  the broad runtime-only condition. Full Go suite/vet/diff passed; Pi arm64
  SHA-256 `12701b042b74f24585db86acf606d07292010373da70ef54817c37d0da5736cb`
  installed while idle with rollback retained, service active/unthrottled.
  Rendered multi-radio browser acceptance remains open.

- 2026-09-19 RF-04 UI truthfulness: P25 mixer rows no longer claim that
  talkgroup audio is routed to a system output. They now state that OP25
  receiver audio streams through GP-SDR, while mute/solo affects eligibility
  and isolated per-talkgroup audio routing remains unavailable. Local full Go
  suite/vet/diff passed; Pi SHA-256
  `9a9ea80df4647188a5c4d1784361d7fbba5a5852e711325a00394bac1b14382c`
  installed while idle with rollback retained, service active/unthrottled.
  This is not RF-04 implementation or media acceptance.

- 2026-09-19 RF-01/RF-03 bounded installed-Pi audio path: a receive-only
  Pluto P25 session obtained control lock, OP25 produced 326 PCM frames, and
  the authenticated GP-SDR live-audio endpoint delivered 91,656 bytes during
  a ten-second no-playback read. It was stopped afterward and Pi throttling
  remained `0x0`. This confirms decoder-to-app stream delivery, not browser
  playback, intelligibility, call continuity, or per-talkgroup mixing.

- 2026-09-19 DEC-01/02 inventory: installed Pi reports connected/available
  HackRF, RTL-SDR and PlutoSDR plus ready Analog, P25, DSD-FME, rtl_433,
  dump1090, multimon-ng, acarsdec and AIS components. This is dependency and
  discovery evidence only; each advertised decoder remains open for real-frame
  GUI/live/deferred acceptance.
