# P25 rebuild

Priority reset requested October 15, 2026. This is the next development priority,
ahead of further P25 patches and unrelated feature expansion. Status: milestone
1 in progress; input transport is verified on all three radios and milestone 3
interface work has started. The full replacement
and independent three-radio RF acceptance are not complete.

Rebuild GP-SDR's P25 session management, receiver coordination, status, audio
identity, and operating interface. Use established P25 decoding components where
they meet the requirements; writing a new protocol decoder is not a prerequisite.
Select and document the engine after a bounded capability check on the Pi.

## Problems established by the source review

- `server/internal/app/p25.go` generates one decoder channel per selected radio.
  Cooperative control/voice allocation and distinct concurrent calls need direct
  proof; the number of checked radios is not a verified call-capacity figure.
- Session capture width is taken from the first configured device. Front-end
  status similarly describes the first assigned device, despite different rates
  and gain controls on the others. An 8 MS/s session label does not establish
  that the RTL-SDR is sampling at that rate.
- The existing backlog keeps isolated HackRF reception, true talkgroup audio
  routing, simultaneous-call capacity, and recovery acceptance open. These are
  foundational requirements for the replacement.
- The current page combines profile selection, receiver planning, archive
  settings, technical metrics, and mixer controls. The replacement needs a clear
  everyday operating view and an accessible setup/diagnostics view.

## Milestone 1 — Prove each receiver and define the session contract

Target: PlutoSDR, working HackRF a447, and RTL-SDR independently receive through
the chosen engine on the Pi 5 with 8 GB RAM. Keep c5cb excluded.

1. Inventory the actual deployed engine/version, drivers, configuration,
   recordings and receiver identities; establish a rollback snapshot.
2. Compare required engine capabilities on the installed Pi: Phase 1/2,
   simulcast, device-specific tuning, control following, call identity, isolated
   audio, and multi-call scheduling. Choose a documented architecture based on
   this evidence, rather than current UI assumptions.
3. Introduce per-receiver capabilities and applied settings: transport, sample
   rate, usable bandwidth, gain, frequency correction, ownership and errors.
4. Define session states and time-stamped evidence for searching, control lock,
   grants, voice decoding, encrypted traffic, degraded operation and failure.
5. Run short isolated receiver checks against a verified active control channel;
   use replayable IQ fixtures for deterministic regression checks. Keep live RF
   evidence distinct from replay evidence and subjective listening acceptance.

Done when every supported receiver has independently attributed control decode
and an unencrypted voice call, per-device settings match the applied config,
stop releases the radio, and failures identify a cause/action. If live RF is
unavailable, finish reversible implementation and replay checks, but keep that
receiver's live acceptance explicitly open.

## Milestone 2 — Coordinate receivers and preserve call identity

1. Build explicit control/voice or shared wideband roles according to the chosen
   engine's proven capabilities. Do not assume one receiver always equals one
   call, or wider capture automatically adds decoders.
2. Allocate different eligible calls across available capacity, retain control
   coverage, prevent duplicate recordings and report queued/missed calls.
3. Key calls by system, site, talkgroup, call identity, frequency and TDMA slot;
   carry that identity through audio, history and recordings.
4. Verify real per-call/talkgroup mute, solo and volume without affecting another
   call. Support separate systems through explicit receiver assignments.
5. Measure one, two and three receivers on the Pi: useful call capacity, CPU,
   memory, sample loss and temperature. Use per-device rates and bounded queues.

Done when distinct overlapping calls are demonstrably captured and attributed,
resource pressure is visible, and losing one radio does not corrupt other calls.
Publish measured capacity and limitations rather than an assumed maximum.

## Milestone 3 — Replace the field operating interface

1. Main view: system/site, Listen or Silent record, Start/Stop, control reception,
   active calls and an obvious recordings entry point.
2. Setup view: site/control candidates with provenance, receiver roles, supported
   gain/rate/PPM controls, talkgroup filters and recording preferences.
3. Diagnostics view: per-radio applied settings, decoded-message age, signal
   health, grants/voice/error counts and actionable recovery messages.
4. Make one responsive interface first: large touch targets, readable labels,
   stable control positions, no hover-only actions, and no continual whole-form
   replacement while editing. Separate box/browser layouts only where testing
   shows a concrete benefit. Use the Pi's system keyboard.
5. Verify every visible action in Firefox and at the actual 10-inch display's
   resolution; track physical touch/visual acceptance separately when necessary.

Done when setup → start → understand reception → listen/record → stop → review
works through the UI, with useful empty/error states and no clipped controls.

## Milestone 4 — Complete silent recording and recovery

1. Store unencrypted call audio with system/site, talkgroup/name, date/time,
   duration and receiver metadata; provide searchable day/system/talkgroup views.
2. Keep silent monitoring independent of recording. Skip encrypted calls and
   prevent their audio from entering the archive/transcription path.
3. Transcribe saved calls through a persistent, bounded background queue. Show
   pending/complete/failed states and allow retry; prioritize live decoding over
   transcription on the Pi.
4. Exercise control rotation, signal loss, radio disconnect, decoder crash,
   restart, full/unavailable storage and interrupted recording/transcription.
5. Migrate existing profiles/settings while preserving recordings, credentials,
   calibration and rollback. Switch the live service after bounded checks and
   record the deployed revision; do not publish secrets or captured call media.

Done when audio/transcript identity is correct, silent operation remains silent,
failures are recoverable without duplicate/corrupt calls, and the deployed Pi
passes short end-to-end checks. Keep long soak and listening gates explicit.

## Execution rules

October 15 user adjustment: first confirm changing RF reaches the decoder from
each receiver, then advance milestone 3. Defer independent P25 reception tests
until the large antenna can be used on HackRF and RTL; only Pluto currently has
the large antenna. This does not waive those reception/call acceptance gates.

Complete reversible development, short meaningful
tests and documentation before reporting a milestone complete; identify any
remaining live RF or user-dependent gate explicitly. Preserve existing work and
data. Maintain RF-01 through RF-09 and UI-01 through UI-04 in the master backlog;
this plan changes priority, not historical acceptance evidence.

## October 15 implementation and bounded evidence

- Retain the installed OP25 engine for the initial replacement adapter. Direct
  inspection of its installed `tk_p25.py` confirms control assignment, a shared
  talkgroup scheduler, Phase 2 slots, and encryption exclusion hooks. This is
  source evidence of capabilities, not accepted concurrent-call performance.
  Installed `multi_rx.py` SHA-256:
  `f822465dee67947d02b79674a8b0b96ef24a0f96289b1042b89a1df9c4aed6c6`.
  Installed `tk_p25.py` SHA-256:
  `dc8091e23f8e2ddb91d10a8fa9c3b554308246d92be632deab1a2b5fadc362cf`.
  The installed files differ from the old upstream checkout; do not identify
  that checkout's commit as the exact installed engine version.
- Added `p25_session.go`: immutable per-receiver generated configuration plus
  coherent decoder telemetry, with receiver identity, rate, usable bandwidth,
  gain, correction, transport, role, tuned frequency, slot and control timestamps.
  Configuration is explicitly distinguished from hardware readback. Rejects
  stale, future, foreign-system and previous-session control evidence; idle
  receivers do not inherit another receiver's lock. Missing telemetry is unknown
  reception. Decoder HTTP polling no longer holds the manager lock needed by Stop.
- Added receiver status cards and an explicit preferred control receiver. Mixed
  rates are reported per receiver. Capture-setting restarts preserve multiple
  receiver assignments. This is an interim view of the new contract; the complete
  operating/setup/diagnostics interface remains milestone 3.
- Browser validation reproduced and fixed recording-checkbox drafts being lost
  when setup controls rerendered. Preferences are retained per selected profile
  until saved at Start. Receiver-card styles explicitly override older global
  definition-list layout rules instead of inheriting their column arrangement.
- `p25_receiver_baseline.py` provides bounded receive-only isolated checks with
  separate configs/logs and PCM transport counts. It plays and saves no audio,
  leaves profiles unchanged, rejects the excluded c5cb device, and checks for an
  already occupied decoder control port before running.
- All three isolated ALCO East candidate scans ran for about 42 seconds each
  without a decoder exit, control messages or PCM. No site-lock claim follows.
- Pluto on the saved EBRCS CCCO Central configuration at 2.4 MS/s independently
  decoded control at 774.456250 MHz and delivered 270 PCM packets / 43,200 samples
  in an 18-second check. This establishes control/PCM transport for that setup;
  listening quality and call/audio identity acceptance remain open.
- RTL at 2.4 MS/s (LNA 36 and 49), and HackRF a447 at 8 MS/s (LNA 8/VGA 8/amp on)
  then 5 MS/s (LNA 8/VGA 0/amp off), did not decode control or PCM during bounded
  18-second Central checks. Antenna/feed equivalence was not established; these
  results do not distinguish RF conditions from receiver/driver configuration.
- Pi panel reports a preferred 1280×800 mode. Browser rendering at that size is
  a separate check from physical panel readability/touch acceptance.
- A rollback binary and private test evidence are under the Pi operations
  directory `p25-rebuild-20261015`. The old Pi source checkout is stale relative
  to the service; the pre-change local `app.js` and `app.css` hashes exactly
  matched the running embedded assets. Build from the matching local source and
  retain the candidate source archive with each installed binary.
- Clean-index verification uncovered prerequisite runtime source files omitted
  from earlier commits (queue/classifier, Pluto transport, health/power and relay
  settings/audio). Restored those dependencies and their required configuration
  members/styles/tests so this increment can build from tracked source. The
  staged source passes the full Go suite, targeted P25 race checks and `go vet`;
  this does not accept those other subsystems' outstanding physical gates.

Milestone 1 remains open for independent RTL/HackRF control and voice, receiver
capability/setup work, replayable RF vectors, and full lifecycle/error acceptance.
Milestones 2–4 are not marked complete by this foundation work.

## October 15 input-path gate and field interface increment

- Receive-only inline probes in the installed OP25 flowgraph checked 8,192
  finite, changing IQ samples per device and counted symbols forwarded directly
  into its real frame assembler. Each receiver ran independently for two 2-second
  observation windows using the saved Central site's device settings. In the
  second window: Pluto advanced 4,784,128 IQ samples and 9,599 decoder symbols;
  HackRF a447 advanced 9,612,538 and 9,244; RTL advanced 4,761,514 and 9,609.
  Hardware rates reported 2,399,998 / 5,000,000 / 2,400,000 samples/s respectively.
  All three passed this input-path gate; this is not a valid-frame or voice claim.
- A combined flowgraph also delivered changing IQ from all three devices.
  Unassigned voice receivers intentionally produced no decoder symbols while
  idle; independent runs activated each receiver's demodulator before requiring
  advancing assembler input. Do not interpret scheduler idleness as failed RF.
- `Scripts/pi/p25_decoder_input_check.py` is a bounded diagnostic, not persistent
  instrumentation. It preserves source configuration, rejects c5cb, keeps only
  a small sample in RAM, forwards the original stream, and creates no IQ/audio
  files. Stop P25 before using it; pass the exact installed runtime config and
  one device name. Private results/logs remain on the Pi, outside Git.
- Field console now separates Operate, Setup, Diagnostics and Recordings.
  Operation keeps system selection, start/stop, browser listen/mute, startup
  silence, reception, receiver count, call assignments and archive summary.
  Setup holds receiver planning, capture/gain controls, recording preferences,
  saved control candidates with honest source labels, and the system editor.
  Diagnostics retains per-radio settings and adds last decoded-control age.
- Recordings shows recent loaded, unencrypted P25 audio with system, talkgroup,
  date/time, duration, receiver and transcript; search and playback reuse the
  existing archive paths. It is limited to the loaded activity history, with an
  explicit full-history entry. Full archive pagination, persistent transcript
  queue status/retry and guaranteed concurrent-call identity remain open.
- One responsive interface, minimum 44px action targets, larger checkboxes,
  wrapping tools and no built-in keyboard. Panel switches preserve form nodes;
  focused form controls remain protected from polling replacement.

### HackRF operating minimum and verification

- User requirement: HackRF operates at a minimum of 8 MS/s throughout GP-SDR.
  OP25 and SDRTrunk automatic/fallback P25 rates are now 8 MS/s; 5 MS/s is no
  longer offered for HackRF. Tuner, survey, Mapper, wideband, calibration and
  comparison, spectrum analyzer and FPV paths honor the floor for native and Soapy devices. The capture
  boundary rejects lower native/Soapy requests. HackRF transmit fixture/sample
  generation and native output also use 8 MS/s; no RF transmission was performed.
- A repeated isolated HackRF a447 input check at an actual 8,000,000 samples/s
  passed: 8,192 finite changing IQ samples, 15,955,296 additional IQ samples and
  9,599 symbols forwarded into the real OP25 assembler over two seconds. This
  supersedes 5 MS/s as the operating test configuration, not its historical data.
- Firefox desktop and 1280×800 responsive checks exercised panels, profile
  selection, recording draft retention, system editor entry, search/no-match,
  muted saved-audio playback and full-history navigation. Start/Stop/restart
  returned the saved three-radio session; browser listen/mute was exercised
  with master output muted. Physical touch and subjective audio remain open.
- Targeted P25/web/rate checks and ARM64 build passed; no raw test samples, call
  recordings, transcripts or credentials are included in the Git increment.

The full Go suite passed after updating the simulated analog fixture to use
the same 8 MS/s HackRF input as the runtime; the ARM64 build also passed.

## October 15 receiver setup and archive browsing increment

- Added optional profile settings keyed by physical receiver identity for OP25
  capture rate, gain and frequency correction. HackRF exposes separate LNA/VGA
  and amplifier controls; Pluto exposes PGA and RTL exposes LNA. Blank fields
  retain the shared profile or saved calibration. Unsupported controls/rates
  fail with an actionable error; HackRF remains at least 8 MS/s. Receiver tuning
  uses explicit Save/Use profile defaults actions while stopped. Unsaved tuning
  blocks Start instead of silently discarding changes. Other radios and global
  calibration remain unchanged. SDRTrunk explicitly rejects these OP25-only
  overrides rather than claiming to apply them.
- Corrected the remaining SDRTrunk conservative tuner-file fallback from 5 to
  8 MS/s. Valid wider saved rates are retained for wide systems; legacy rates
  below the floor are replaced. The earlier rate-floor increment did not cover
  this tuner-file branch.
- Profile editor now shows and preserves receiver-to-system targets and handles
  legacy numbered voice roles. Newly created P25 plans use canonical voice roles.
  Receiver setup lists saved role/system assignments; running configuration is
  shown separately from saved choices and is not described as hardware readback.
  Single-radio front-end summaries now also use generated configuration.
- Added authenticated `/api/p25/recordings` with filtering before limiting and
  stable older-call cursors. Confirmed unencrypted saved P25 calls are independent
  of survey volume and the general Activity search. The endpoint reads journal
  metadata in memory, not audio files. The view has loading, retry, empty, older
  history and last-refresh states. Recent calls refresh every five seconds only
  while Recordings is visible; paging older calls pauses automatic replacement.
- P25 screen refresh is capped at 1.5 seconds while form input stays protected
  and button focus is restored. No persistent diagnostic RF probes were added.
- Regression coverage verifies device identity after receiver reorder, explicit
  zero gain and PPM persistence, invalid cross-device settings, archive filtering
  past 175 unrelated events, cursor stability during incoming calls, stale-search
  rejection, paging/retry and profile editor assignment preservation.

The full field interface, RF reception and concurrency acceptance remain open.
This increment does not establish simultaneous independent call identity,
subjective voice quality, transcription queue recovery or physical touch feel.

Verification for this increment: full Go suite and Node workspace regression
checks passed. Firefox confirmed HackRF choices of 8/10/20 MS/s, saved 8 MS/s
and LNA 8 without changing other receiver settings, Start protection for unsaved
tuning, and archive search/no-match behavior. Desktop and 1280×800 layouts were
inspected; physical touchscreen acceptance remains separate. Final field spacing
was widened, and archive rendering is bounded to 500 loaded calls (search can
still reach older journal entries). Live archive cursor requests returned distinct
older IDs. Historical calls with missing receiver attribution or near-zero
duration metadata remain an archive-identity issue for later P25 acceptance.

## October 16 field-interface diagnostics and handoff

- System-level OP25 control-message totals, voice-frequency update counters and
  last control-decode age now remain distinct from each receiver's evidence.
  Valid stale system timestamps remain visible during reception loss; previous
  session/future timestamps are rejected. Frequency offset is labeled in Hz,
  not as an error count. Missing telemetry remains unavailable rather than zero.
- Diagnostics explains searching, scheduler-idle radios, incomplete telemetry,
  decoder exits, and the next recovery step. Preferred control is explicitly a
  startup preference, not a pinned role. Per-radio transport, configured capture
  width and demodulator offset are visible. No packet-error/grant/RSSI totals are
  invented when the installed adapter does not report them.
- Operate identifies the configured systems/sites and provides a direct Review
  recordings action. Diagnostics links back to receiver setup. Navigation updates
  the selected tab consistently. Calibration text distinguishes defaults from
  explicit receiver overrides. Receiver cards wrap at a wider minimum width;
  diagnostic labels are readable at the 1280×800 panel resolution.
- Verification: full Go suite, focused P25 race checks, vet and workspace Node
  regressions. Firefox at 1280×800 exercised the saved Central site's silent
  three-receiver Start/Stop, enabled/disabled setup states, diagnostics and its
  setup shortcut; prior October 15 archive search/playback checks still apply.
  Desktop and final deployed layout checks are recorded in the handoff below.
- Group 3 software field-interface scope is ready to hand off to group 4.
  Final desktop and 1280×800 Firefox rendering and both new shortcuts passed;
  the exact staged-source full suite also passed. This is not completion of milestones 1/2 or full UI/RF
  acceptance. Physical touch feel remains user acceptance; the user previously
  reported the touchscreen works well. Final HackRF/RTL antenna reception,
  overlapping call identity, intelligibility and recovery remain explicit gates.

Group 4 order: correct call identity/duration/receiver metadata; persistent,
bounded transcription state and retry; encrypted-call exclusion; decoder/radio/
storage interruption and restart recovery; then the requested full review with
GPT-6 Astra. Preserve all media, credentials, settings, calibration and rollback.

Final October 16 deployment: ARM64 SHA-256
`439915aad81807fab2958954c022bd2db5b3dc3b044c8ca6807a63cd55cb0092`.
Rollback and matching full candidate source are retained privately in the Pi's
`p25-field-20261016` operations directory; unrelated pre-existing local source
changes remain preserved. Three-radio Central silent collection was restored
at 2.4/8/2.4 MS/s. No credentials, recordings or transcripts are committed.
The live store audit also reports 182 missing media references across the
existing archive. Group 4 must distinguish unavailable media from playable
calls, preserve their findings, and test recovery without inventing missing data.
