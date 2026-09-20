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
- [x] Bounded Pi P25 hardware stream evidence: the dedicated Pluto profile
  locked 774.45625 MHz and OP25 delivered 4,684 8 kHz PCM frames to the
  authenticated live-audio endpoint without a local sound device. Browser
  playback/intelligibility remains open and is not implied by this check.
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
- [ ] IMG-01 NOAA APT/weather-image receive workflow and live image acceptance remain unimplemented; scope is separate from packet decoders.
- [ ] IMG-02 Digital SSTV receive workflow and live image acceptance remain unimplemented; one mode must be selected before implementation.
- [ ] PLAT-01 Android preview is not an accepted functioning RC; on hold.
- [ ] PLAT-02 iPad remote/signing/USB feasibility and actual acceptance remain; on hold.
- [ ] PLAT-03 Mobile web end-to-end, reconnect/accessibility acceptance remains.
- [ ] PLAT-04 Power-HAT telemetry and the optional Pi-local TFT status display
  need physical acceptance. Driver/framebuffer/service startup is verified;
  actual panel readability/orientation and unplug/recovery are not yet.
- [ ] DIST-01 Clean-machine install/update/rollback acceptance across desktop targets remains.
- [ ] DIST-02 Updater lacks complete secure manifest/rollback/replay protection acceptance.
- [ ] DIST-03 Cross-platform credential storage/migration/security acceptance remains.
- [ ] DIST-04 Clean dependency bundles need rebuild and artifact acceptance.
- [ ] DIST-05 Wiki/setup/license/source-offer information needs alignment with shipped evidence.

## Broken or previously reproduced faults requiring closure

User-deferred hardware: HackRF `c5cb` is excluded from tests/acceptance until the
user confirms independent retesting and returns it. Retain the fault record
below but do not treat that unit as an active application-remediation blocker.

- [ ] RF-11 RTL USB disappearance/dropouts: prior -110/-62, direct/hub longevity remains unresolved.
- [x] Bounded RTL stream health check: 2.4 MS/s direct streaming alongside a
  locked Pluto P25 session completed without new kernel USB errors or Pi
  throttling; the RTL remained available. Long-duration/direct-vs-hub/cable
  acceptance remains open.
- [ ] RF-02 Hardware prerequisite: c5cb produced pinned-negative/corrupt IQ; exclude from RF acceptance until repaired and retested. This is not proof of a GP-SDR decoder bug.
- [ ] AI-01 Prior non-speech hallucinations: suppression changes exist, but latest batch must demonstrate closure.
- [ ] UI-01/02 Prior clipped controls and unresponsive/click-target reports: full page-by-page GUI retest remains.

Fixed in this work tranche (bounded software scope only):

- [x] Deferred IQ retention waits for combined-group model success; failed combination preserves captures and marks files error instead of complete. Cancellation before cleanup restores pending files. Regression checks first-pass success→group HTTP 503, preserved IQ and failure counters. Explicit error-group retry and status-persistence failures remain open; production deployment pending.

- [x] Deferred decoder/configured transcription failures propagate before IQ cleanup; unconfigured transcription is explicitly skipped. Group-model failures populate error log/status instead of unconditional combined-evidence success. Configured missing-model retention and group HTTP 503 regression pass locally. Full group retention/retry and physical speech/decoder acceptance remain open.

- [x] Deferred local-model request failures no longer mark capture complete and proceed to IQ retention cleanup. Invalid saved WAV fails explicitly rather than being silently skipped. Regression checks HTTP 503 retains IQ. Local race suite/vet pass; Pi deployment pending. This does not close all decoder/transcription/group-combination failure handling.

- [x] Mapper job disk failures: Save/Create/Delete return persistence errors and retain original in-memory/disk state; background results/job snapshot failures appear in Mapper error status. Race suite/vet and Pi exFAT fixture regressions pass. This is not complete NAS/offline recovery acceptance.

- [x] Native channel metadata repaired: HackRF 1 RX/1 TX half-duplex; RTL 1 RX/0 TX. Pluto remains reported 1/1, not fictional dual-channel support.
- [x] HAT read errors no longer silently remove the Hardware card; escaped error feedback and automatic recovery tested in JS.
- [x] PiPower5 isolated subprocess import prevents CWD/PYTHONPATH shadowing; installed SDK read verified from conflicting home directory.

- [x] Pi-local status-display software setup: identified the SHCHV/LCDWIKI
  2.4-inch panel as ILI9341 SPI (GPIO27 reset, GPIO22 D/C, CE0 LCD, CE1 touch),
  verified it does not collide with PiPower5 I2C telemetry, and installed a
  read-only `/dev/fb0` dashboard service. It uses only authenticated local GET
  requests and displays service/P25/receiver/PiPower/thermal/storage status;
  it has no touch or RF controls. After a reboot the framebuffer and service
  recovered, framebuffer contents changed on the two-second refresh, and the
  Pi remained unthrottled. Physical viewing/orientation remains unaccepted.

- [x] Pi-local display behavior: Overview is the boot/default page and refreshes
  once per second. A touch advances through Power, Radios, and System; a
  non-overview page returns to Overview after 30 seconds without a touch. The
  System page reads host CPU load/temperature, V3D clock, RAM, GP-SDR data
  storage, IPv4 address, and RX/TX rate locally. Physical touch navigation and
  panel readability still require on-device acceptance before touch calibration
  or guarded power controls are enabled.

- [x] September 17 profile-name error: bundled P25 names exceeded the old 80-byte limit. Editor/backend now allow 160 Unicode characters; duplicate names fit that bound. Regression includes long EBRCS names, multibyte names, duplicate/resave and blank/over-limit rejection.
- [x] Profile disk failures: Save/Import/Duplicate commit in-memory state only after persistence; failed deletion keeps the profile visible. No phantom success or duplicate after failed write.
- [x] September 17 VPN restored; test5 deployed and a 127-character bundled P25 profile saved/duplicated/resaved through the installed API. Two expendable copies deleted afterward; original profiles and Mapper records compare unchanged against backup. Pluto P25 reacquired 774.45625 MHz lock.
- [x] Profile filename confinement: Save/Import/persist reject path traversal, slash/backslash, NUL, absolute paths and dot entries. Unit/race tests cover unsafe IDs; installed API returns 400.
- [x] Transmit upload total-body limit: capped at 51 MiB including multipart overhead, alongside the existing 50 MiB audio-file cap.

- [x] Malformed WAV uploads are rejected before persistent storage; truncated chunks/incomplete PCM frames are rejected. Shared parser regression passes. Multipart temporary upload cleanup is explicit.

- [x] Concurrent/repeated transmit preparation is rejected before allocating IQ or saving files; active-job rejection also moved before generation. This does not establish cross-workflow atomic receiver reservations.
- [x] Saved benchmark library added to Transmit: bounded metadata listing, invalid-manifest counts and synthetic eligibility/paths/checksums. Listing never starts RF, loads IQ, promotes verification or trains a model. Label review/export/replay remain open.

- [x] TX fixture memory amplification: removed two full float64 ideal-sample arrays; 60 seconds at 2 MS/s now requires ~240 MB output rather than ~2.16 GB waveform buffers. Waveform math preserved.
- [x] TX same-time fixture file collision: unique output filenames prevent overwriting an earlier benchmark.
- [x] TX nonfinite duration/fixture parameters: reject NaN/Infinity before generation.
- [x] Dataset ambiguity: manifest records generator/schema/seed and marks synthetic-only eligibility; no automatic training or receiver verification claimed.

## Missing implementations

- [ ] AI-07 Local-first / remote deep-analysis queue: configurable promising-sample selection, second Ollama endpoint/model, availability-aware persistent retry queue, protected IQ retention, bounded resources, opt-in transfers and separate first-pass/deep provenance. Current metadata-only LLM input is not direct IQ decoding. Deferred behind existing reliability work; requirements recorded, not implemented.
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

- RF-02 current HackRF acceptance progress: with a temporary receive-only
  profile at 5 MS/s, RF amp on, LNA 8 dB and VGA 0 dB, good HackRF `a447`
  held EBRCS 774.456250 MHz control lock through 18 of 19 one-second status
  samples and produced 178 OP25 PCM frames. The temporary profile was deleted
  and the runtime stopped. This confirms current control-lock/PCM operation,
  but not human-listened voice continuity/intelligibility or talkgroup mixing.

- UI-04/UI-05 Band Monitor: automatic HackRF channel-bank capture retains a
  10 MS/s preference, but a valid explicit lower rate is no longer overridden.
  A temporary 98.1/98.3 MHz bank applied and reported 5 MS/s with its selected
  LNA/VGA/amp controls, and Band Monitor now publishes live receiver telemetry
  to Hardware. Full analog audio/control acceptance remains open.

- RF-02/UI-04 receiver integrity: GP-SDR now detects a quadrature ADC path
  pinned at a rail before DC/IQ correction, emits a hardware-specific warning,
  and keeps it distinct from ordinary gain overload.  The implementation has
  signed-I/Q pinned-path and normal-noise regressions plus full/race/vet
  coverage and is installed on the Pi. An initial direct HackRF Q-rail reading
  did not reproduce in controlled repeats or the later GP-SDR Tuner probe, so
  it is recorded as transient rather than a receiver-failure diagnosis.

- 2026-09-18 current Pi deployment: the full current source suite, vet and
  race run passed before deployment.  The cross-compiled arm64 binary
  `8be6e6fcd11b380fa5e7b004674873429e330a9bbc619e473e69debe9947b901`
  replaced the older Pi binary only while GP-SDR was idle; the former binary
  was retained as a timestamped rollback copy.  The restarted service is
  active and the Pi reported `throttled=0x0`.  No profiles, results, captures,
  or calibrations were cleared.

- RF-02 current receiver comparison: the good HackRF `a447` ran the known
  receive-only P25 profile at the applied 5 MS/s and remained searching after
  eleven seconds.  The same profile on the connected PlutoSDR immediately
  afterward locked 774.456250 MHz at 2.4 MS/s and reported 284 OP25 PCM
  frames.  Both jobs stopped cleanly; GP-SDR remained active and the Pi was
  unthrottled.  This narrows the open issue to the HackRF RF/frontend or
  device-specific configuration path rather than a global P25 decoder outage.
  It does not establish voice intelligibility.

- RF-02/UI-04 HackRF P25: automatic HackRF P25 capture now defaults to 5 MS/s
  on Pi, while explicit supported higher rates remain available. A pre-deploy
  isolated 5 MS/s run on good unit `a447` locked the known 774.45625 MHz
  control and produced OP25 PCM. Fresh deployed 15/35-second runs with the
  same profile (including after the antenna change) searched without locking;
  the amp-on measurement clipped about 12.3% of IQ samples, so RF amp remains
  off for this source. Pluto concurrently relocked and delivered 910 PCM
  frames. HackRF voice/lock remains unaccepted.

- RF-02 continued: alternate 8 MS/s and high receive gain (LNA 40/VGA 32,
  amp off) both started cleanly but did not lock during a full configured
  control-list cycle. This rules out the obvious rate/gain configuration
  failure; it does not prove the HackRF is defective or establish voice audio.

- RF-02 current positive evidence: low-gain amp-on HackRF (LNA 8/VGA 0,
  5 MS/s) locked 774.45625 MHz and received 1,336 OP25 PCM frames. Preserve
  this as a profile-level safe starting point, not a global amp-on default.

- RF-02 refreshed deployed evidence: good HackRF `a447` again locked
  774.456250 MHz at 5 MS/s with amp on/LNA 8/VGA 0; OP25 reported 122 live
  8 kHz PCM frames. The short receive-only window had no enabled unencrypted
  call to judge for intelligibility, so RF-01/RF-02 voice acceptance remains
  open.

- RF-03 bounded bridge evidence: the locked low-gain HackRF P25 session
  delivered 15,048 bytes through GP-SDR's authenticated live PCM endpoint.
  End-user voice listening and talkgroup routing remain open.

- UI-05 Band Monitor: a temporary one-channel 98.1 MHz WFM receive-only
  profile run through the installed Band Monitor API on good HackRF produced
  351,392 bytes of authenticated live PCM in eight seconds, with no local
  playback and no retained test profile/media. The repair uses a nonpersistent
  runtime open-monitor flag and updates receiver telemetry. It does not yet
  establish subjective sound quality or AM/NFM/full-control coverage.

- MAP-09 storage warning: `/mnt/gp-sdr-data` is presently mounted again, but
  its backing exFAT disk has recorded device-offline I/O failures, a forced
  read-only remount and an unclean-remount warning. Offline fsck requires a
  controlled service stop/unmount and must not be attempted during capture.

- MAP-02 live file-decoder handoff: decoders that require IQ now receive an
  ephemeral, private temporary capture when archival IQ is disabled, then
  remove it on exit. This prevents a no-retention preference from silently
  disabling live dump1090/rtl_433/AIS decoding. The installed Pi build also
  gives decoder-tagged channels a longer default observation interval. A
  1090 MHz RTL live check had RF-energy events but no decoded Mode-S frame,
  therefore real ADS-B payload acceptance remains open.

- UI-05/DEC-02 Tuner now has a bounded IQ batch path for dump1090, rtl_433
  and AIS instead of presenting file-IQ decoder modes with no input. The
  installed Pi check started and stopped an RTL ADS-B session without a runtime
  error or an output audio device. Real decoded-frame acceptance remains open.

- UI-05 bounded Tuner hardware stream evidence: 98.1-MHz WFM delivered PCM
  through the authenticated endpoint on both RTL-SDR and PlutoSDR without
  opening a sound device. This is transport evidence, not subjective audio or
  full applied-control acceptance.

- DEC-02 ADS-B isolation result: a direct bounded RTL capture outside GP-SDR
  also produced no Mode-S frame through dump1090-fa. The current ADS-B
  no-frame result is therefore not attributed to the GP-SDR decoder bridge;
  it needs a known-good 1090-MHz RF source/antenna acceptance check.

- DEC-02 rtl_433 isolation result: a bounded direct 433.920-MHz RTL capture
  had no rtl_433 JSON frame. This is an environmental no-sensor observation,
  not an application decoder failure; real rtl_433 acceptance needs a known
  transmitting sensor or a recorded known-good fixture.

- UI-05 bounded analog stream evidence: good HackRF AM and NFM tuner paths
  each delivered approximately 267 KB of authenticated PCM during an
  eight-second no-playback receive-only probe; WFM Band Monitor delivery is
  separately recorded above. This is not user-listened audio acceptance.

- UI-03/UI-04 bounded applied-control evidence: installed Band Monitor now
  honors and reports its requested HackRF 5/10 MS/s, LNA 16, VGA 8 and amp-off
  settings. A 5 MS/s live monitor emitted 495,242 bytes of PCM. This is not
  complete per-page/capability acceptance.

- UI-09 bounded analyzer evidence: good HackRF swept 98–100 MHz at 5 MS/s,
  generated 104 slices/1,024 bins, cleared successfully and stopped cleanly
  without Mapper work or Pi throttling. Advanced UI and multi-radio acceptance
  remain open.

- MAP-09/10 orphaned atomic snapshot cleanup: guarded startup recovery now
  removes only unheld regular `.gpsdr-write-*.tmp` files, never canonical data
  or symlinks. On the Pi it cleared six stale copies (391,206,174 bytes); event
  count grew from 5,332 to 5,381, Mapper records stayed at 5,000, and total
  GP-SDR storage fell from about 582 MB to 256 MB. The deployed binary is
  `f1339d291a8964ad3e3c1c3d90e4f7712d94d465b77585e8fa7b01feea27d3ac`.
  This does not delete results/history and does not close broader storage/NAS
  recovery acceptance.

- RF-11 concurrent-radio soak: an RTL-SDR 2.4 MS/s receive-only soak ended
  with librtlsdr error `-5` and at least 180 lost bytes during a kernel USB
  over-current event that disconnected the upstream hub and SDRs. GP-SDR and
  Pi power remained healthy (external 14.812 V input, 5.246 V output, no
  throttling); radios re-enumerated and OP25 truthfully changed from locked to
  searching. This is physical USB/hub/cable/device-power evidence, not an app
  or P25 decoder acceptance failure. Keep RF-11 open.

- RF-11 escalation: the hub repeated the over-current/disconnect sequence;
  the Pi subsequently listed only root hubs and its card reader, with all SDRs
  absent and GP-SDR inactive. No automatic service restart or further RF test
  was attempted while the physical bus was absent. This requires physical USB
  power-path inspection before resuming receiver acceptance.

- UI-02 remote-receiver empty state: legacy persisted JSON `null` is now
  normalized to an empty array for the remote-receiver API, preventing a
  no-configured-radio state from looking like an invalid API response. Local
  application tests and vet pass; Pi deployment is pending the current
  bounded RTL/Pluto stability check.

- UI-02 deployed remote-receiver repair: after physical USB reconnection, the
  Pi build `9d01bbcac57fe9613cca878967a1386eb6f047bd41be21269900670e3f6744ab`
  was installed with a rollback binary. The API now returns an empty array for
  no remote radios; the service sees RTL-SDR, HackRF and Pluto. Receive-only
  Pluto P25 reacquired 774.45625 MHz and received PCM. USB-hub longevity and
  subjective audio acceptance remain open.

- RF-01/UI-02 browser continuation: a fresh browser accessibility inspection
  of the live Pi P25 workspace showed the explicit user-gesture audio button,
  locked Pluto OP25 control channel at 774.45625 MHz, ten talkgroups and a
  rising PCM-frame counter. A 15-second non-playback probe advanced from
  8,087 to 8,131 frames with no new USB fault, Pi throttling, or service loss.
  It did not enable browser audio and therefore does not establish audible
  speech clarity or mixer acceptance.

- RF-02/UI-04 continuation: the installed Pi P25 configuration now selects
  compatible capture rates per receiver instead of passing a shared Pluto/RTL
  setting into HackRF. A good `a447` run verified applied/generator rate
  10 MS/s, but did not obtain a control lock in 18 seconds. Pluto was restored
  and relocked at 774.45625 MHz. HackRF lock and audible voice remain open.

- RF-07/UI-02 continuation: an exited OP25 process now clears GP-SDR's stale
  active Runtime state and surfaces the failure for retry/refresh instead of
  retaining an unusable running session. A real hub over-current event during
  receiver switching disconnected the SDRs; after devices re-enumerated, the
  installed Pluto profile relocked. This is not a long-run USB-power or
  automatic-recovery acceptance result.

- Pi UI-04 test18: Tuner status now reports effective rate/front-end state from live receiver telemetry instead of only requested fields. Unit/UI regression and source checks pass; Pluto P25 relocked after install. This is not hardware-driver readback or visual GUI acceptance.

- Pi UI-03 test17: non-HackRF Tuner/Live requests now neutralize HackRF LNA/VGA/RF-amp/bias fields rather than merely hiding them. Regression and app checks pass; Pluto P25 relocked after install. Browser click-through and applied-value telemetry remain open.

- RF-11 bounded Pi test: 60 seconds direct RTL receive at 2.4 MS/s completed normally alongside locked Pluto P25, with no immediate USB fault or throttle. This is too short to close the reported long-duration RTL dropout.

- Pi P25 test16: live P25 rate/gain/amp changes now preserve the actual runtime receiver rather than a stale UI choice during the supervised restart. Source regression, syntax, audio-queue, unit and vet checks pass; Pluto relocked after install. Browser click-through and audible output acceptance remain open.

- Pi P25 test15: live audio status now reports connected/waiting until a frame is actually received, then reports the PCM rate. The 8 kHz P25 scheduling regression passes. A 25-second no-playback live Pluto stream captured 378 frames / 7.56 seconds of non-silent 8 kHz PCM while OP25 stayed locked. This strengthens delivery evidence only; clear user-listened speech, mixer routing, and full GUI acceptance remain open.

- Pi P25 test14: status now exposes the actual assigned live receiver to the P25 UI, preventing a stale receiver choice from showing the wrong sample-rate/capability controls. Pluto assignment and OP25 lock at 774.45625 MHz verified through the installed authenticated API. Unit/vet and JavaScript syntax pass. A one-frame non-silent no-playback PCM capture is delivery evidence only; real voice intelligibility and complete GUI control acceptance remain open.

- P25 UI repair installed as Pi test12: legacy null profile arrays no longer throw and block rendering; P25 Start now establishes browser audio from the initiating click. Pluto relocked 774.45625 MHz after install; a non-playback live-audio read parsed 191 frames / 30,560 samples. Clear audible playback, UI click-through acceptance and individual-talkgroup audio remain open.
- 2026-09-18 MAP-09/10: raw capture-timing journal retention is now a bounded,
  separately configured policy (128 MB default; zero disables). It retains
  newest complete records with an atomic write and does not target mapper
  results, event history, transcripts, profiles, calibration, or channel data.
  Local full Go tests/vet passed. Installed Pi SHA-256
  `af53619468d742f71a190a7745f822b55803947fe5cf3def895ff390016af3c0`
  remains active; preexisting 108,393,477-byte journal was not compacted
  because no manual/automatic cleanup was requested. GUI/manual cleanup and
  long-run capture append acceptance remain open.
- 2026-09-18 MAP-09/10 UI wording was corrected so manual cleanup names its
  eligible data classes and excludes Mapper results/history explicitly. Pi
  SHA-256 `f7511e183e9cf763f99cde7ba21b87a3634391ae77acff0f4927f7e22942e568`
  installed; service active with no warning journal entries.
- 2026-09-18 MAP-09/10: journal compaction now discards an unterminated final
  row rather than retaining a possibly torn capture record. Regression and vet
  passed; Pi SHA-256
  `50d5056d6e1e63c36f73d7c88bf4933aa36206b732d2ffc130329845854d058b`
  installed and active.
- 2026-09-18 MAP-09/10 post-deploy data check: 89,601 Mapper records, 5,593
  events, 11 profiles, and the unchanged 108,393,477-byte journal remained on
  the mounted Pi data volume; service active and Pi `throttled=0x0`.
- Global PiPower5 status added beside master volume: battery percentage, source and output watts; expandable input/output/battery readings; telemetry error state; hidden on hosts without a HAT. Uses existing cached telemetry, no additional hardware polling. JavaScript syntax and application tests pass; visual acceptance remains pending.
- September 17 persistence follow-up installed as `1.5.0-rc34-pi-test8`, SHA-256 `77a83ab8d96e782b7d3fade40e1856e97d6e82e788d6215f1bf48b1ec24b8318`; rollback `/home/sdr/gpsdr-before-pi-test8-nDmAcr`. Mapper job/results snapshots now serialize writers and use synced same-directory atomic replacement instead of truncating the previous file. Shared JSON writes use unique temporary names. Concurrent replacement and failed marshal preservation tests pass locally and on Pi exFAT. Persistence error reporting and filesystem crash durability remain open.
- 26 installed status GET endpoints returned valid JSON. Twelve isolated storage/retention tests passed on the Pi data drive, including media cleanup preserving results/event history. No user captures/results deleted. RTL transferred for 120 seconds at 2.4 MS/s alongside Pluto P25 with 80 bytes lost and no disconnect; long-run dropout acceptance remains open.
- Ten combined status/HAT samples showed fresh HAT readings and Pluto control lock. A 20-second live-audio capture contained 158 complete OP25 frames / 3.16 seconds intermittent voice PCM, no trailing bytes. This is delivery evidence, not subjective intelligibility or continuous-call loss measurement. Damaged c5cb was not retested in this follow-up.

- September 17 batch installed as `1.5.0-rc34-pi-test7`, SHA-256 `9adb0c9ea31eaf8211b0d04f188d4e30a4b5df45c4268c2506159a97da1cbf00`; rollback snapshot `/home/sdr/gpsdr-before-pi-test7-2tAJvK`. Three fresh physical HAT samples (~15 V external, 5.29–5.31 V output, 97–98% battery), power arithmetic and radio channel metadata verified through installed API. Pluto regained 774.45625 MHz control lock. Battery-source switching and exhaustive GUI acceptance remain open.
- Real RX hardware: c5cb 200,000 complex samples at 10 MS/s/98.1 MHz, mean I/Q -125.85/-124.27, 100% negative, 8 values; a447 same test mean 0.17/1.34, 175 values. Evidence `/tmp/gpsdr-{c5cb,a447}-sept17.cs8` and logs on Pi. RTL log `/tmp/gpsdr-rtl-sept17.log`: 20 seconds at 2.4 MS/s, 28 bytes lost, no disconnection; not long-run acceptance.

- September 17 final deployment: `1.5.0-rc34-pi-test6`, binary SHA-256 `f377bdaec0f4ba8e11a77ef3efd2af5531dd3bd9a19c1d54046a0d611fa4e6df`. Rollback binary plus Data/Profiles snapshot `/home/sdr/gpsdr-before-pi-test6-QGzqfx`. Local unit/race/vet and JavaScript syntax pass. Initial health script omitted the token and rolled back on expected HTTP 401; corrected authenticated health check then passed on reinstall. No user data restored/replaced and no RF transmitted.

- Post-deploy `/api/p25/status`: OP25 `reception=locked`, control channel `774456250`, source decoded control messages. Saved Profiles directory and mapper-records.json compare unchanged against the rollback snapshot; service restart counter remains zero. This confirms control reacquisition, not fresh voice intelligibility.

- Pi deployment: `1.5.0-rc34-pi-test4`, SHA-256 `9ad24e45eefc264664cf8319087ebe7133d8bc0a346c05bc33d538e7ceba0c0e`. Rollback binary plus Data/Profiles snapshot: `/home/sdr/gpsdr-before-pi-test4-H6JZiN`. Existing service/configuration retained. Service active; DSD-FME ready; benchmark library endpoint and served UI present. Restored existing `pluto-p25-hardware-test` profile on its prior Pluto. No TX started. An empty production fixture library is expected: isolated test fixtures were not copied into user data or promoted to training samples.

- Local Go unit suite, race suite and vet passed after fixture changes.
- Audio scheduling and Band Monitor control/ownership JavaScript regression tests passed; app.js syntax passed.
- Running Pi service remained active with zero service restarts; device/decoder/integration/local-AI/transmit status GETs returned HTTP 200.
- Standalone analyzer's correct GET `/api/spectrum-analyzer` returned HTTP 200 (the nonexistent `/status` suffix returned 404, not an analyzer failure).
- Pi ARM64 `1.5.0-rc34-lab-test3` isolated HTTP test generated all ten fixture families (CW/AM/NFM/WFM/OOK/2FSK/GFSK/GMSK/BPSK/QPSK), each 200,000 complex samples, dry-run complete with checksummed provenance. Evidence: `/mnt/gp-sdr-data/gpsdr-lab-acceptance-ek6lyjx7`. Test server terminated afterward; production service/binary and user data unchanged. This is physical-host execution of synthetic fixtures, not a physical RF test.
- M3 Pro benchmark: 200,000-sample QPSK generation took ~6 ms and allocated 410,272 bytes; benchmark speed is not Pi throughput acceptance.
- No RF transmitted or test audio played. No attenuated wiring/input-limit evidence was available, so physical TX acceptance is blocked, not passed.
- These checks do not constitute exhaustive installed GUI or real-hardware protocol acceptance. The coordinated lab is not finished; no public release is claimed.
- 2026-09-18 UI acceptance pass: authenticated Pi browser showed three connected
  receivers, live PiPower5 telemetry, stored activity/event counts, and the
  complete main navigation. Each primary navigation target opened without a
  client-side exception; Tuner digit controls, receiver selector, mode/options,
  Band Monitor profile/channel mixer, Mapper navigation, Hardware, decoder,
  transmit, and Settings controls rendered with labels and tooltips. The pass
  found a timing defect where receiver-specific tuner controls could remain
  hidden after device data refreshed, and unsupported receiver rates remained
  visually present. The deployed UI now reapplies capability visibility after
  tuner rendering and hides unsupported rate choices. Go test/vet and diff
  checks pass. This is control/render acceptance only: no audio playback,
  transmit, destructive cleanup, or user-result modification was performed;
  UI-01/02/05/07 remain open for exhaustive interaction and media acceptance.
- 2026-09-18 Pi audio acceptance continuation: temporary receive-only 98.1 MHz
  WFM tuner runs on the good HackRF, RTL-SDR, and Pluto delivered respectively
  240/1,073/72 live PCM frames to ALSA's null sink. Their sample rates were
  48.78/48/48.78 kHz and non-silent sample counts 290,564/1,281,074/87,311;
  all streams stopped cleanly, the service remained active, and throttling was
  `0x0`. A Pluto P25 run likewise delivered 519 real 8 kHz frames (44,374
  non-silent samples) to the sink. The Pi's HDMI ALSA device rejects playback
  with error 524 because no physical output is present. This verifies RF to
  app to operating-system playback transport, not subjective audio quality.
- 2026-09-19 UI-01/UI-02 Live task-board repair: Live combines active runtime,
  Mapper, and deferred-compute status rather than rotating through one job or
  duplicating the Tuner. Scoped Open/Stop task actions, decoder sidebar links,
  and a non-operational RF relay placeholder are source-regression covered.
  Local unit/vet/diff passed. Browser click-target and long-running combined
  receiver acceptance remain open.
- 2026-09-19 UI-04/UI-05 Band Monitor applied-state continuation: installed
  good-HackRF receive-only GMRS run reported the explicitly requested 10 MS/s,
  LNA 8, VGA 0 and RF amp off through live receiver telemetry after five
  seconds. The UI now has a separate applied state line. No audio output was
  opened and no user data was changed; browser/audio/CTCSS acceptance remains
  open.
- 2026-09-19 Band Monitor/P25 continuation: Whole Band GMRS hardware handoff
  succeeded from HackRF to Pluto and rejected RTL-SDR without disrupting the
  Pluto session. Pluto P25 locked and emitted 210 frames in a short no-playback
  run; HackRF P25 did not lock in the current environmental/profile test, and
  RTL SDRTrunk exited after startup. These are test findings, not acceptance.

- 2026-09-19 RTL P25 diagnosis correction: the apparent decoder failure was
  caused by a malformed internal test request (`id` instead of `profileID`),
  not a reproducible RTL/OP25 launch failure. A fresh receive-only run opened
  RTL index 0 at 2.4 MS/s and rotated across EBRCS candidates without exiting;
  no control lock occurred in the bounded environmental window. OP25 now
  exposes a concise decoder-log tail when it genuinely exits. This preserves
  the open real RF/control-lock/audio acceptance requirements.

- 2026-09-19 P25 hardware comparison: with the unchanged known P25 profile,
  Pluto decoded control lock during 5 of 8 installed-Pi observations; the good
  HackRF session remained healthy but did not lock or receive PCM in its own
  8-second window. This is an honest unresolved HackRF RF lock condition, not
  a claim that P25 audio has passed or that the HackRF app path crashes.

- 2026-09-19 UI-04 P25 front-end state: the P25 workspace/API now reports
  applied session gain/front-end values rather than leaving operators to infer
  them from profile controls. Installed Pi verification returned the live
  Pluto PGA setting during a bounded receive-only run. Cross-page/browser
  click acceptance remains open.

- 2026-09-19 RF-02 tuner-level comparison was inconclusive: generic analog
  tuner telemetry does not agree with the established Pluto OP25 lock, so it
  must not be used as a cross-device P25 sensitivity verdict. A controlled
  common-feed/calibrated-signal test remains required.

- 2026-09-19 UI-01 P25 metric strip now adapts to the available width and
  wraps live front-end state rather than clipping it. Source checks and Pi
  deployment passed; rendered browser acceptance is still pending.

- 2026-09-19 UI-04 Hardware cards now attribute active streaming only to the
  assigned Mapper/P25/telemetry receiver, rather than every connected SDR.
  Regression and Pi deployment passed; live browser acceptance remains open.

- 2026-09-19 RF-04 P25 mixer disclosure corrected: the UI now distinguishes
  receiver-stream audio from the still-unimplemented isolated per-talkgroup
  routing. This preserves the open mixer/audio acceptance requirement.

- 2026-09-19 bounded P25 stream verification: Pluto locked, OP25 generated
  326 PCM frames, and GP-SDR live audio delivered 91,656 bytes without opening
  a sound device. Clear browser/listening quality and talkgroup isolation
  remain open.

- 2026-09-19 decoder inventory: all advertised Pi decoder components currently
  report ready alongside the three connected receivers. Real over-air frame
  acceptance is still required for every non-analog protocol.
