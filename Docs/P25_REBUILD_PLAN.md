# P25 rebuild

Priority reset requested October 15, 2026. This is the next development priority,
ahead of further P25 patches and unrelated feature expansion. Status: milestone
1 in progress; the session evidence contract is implemented. The full replacement
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

Implement in milestone order. Complete reversible development, short meaningful
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
