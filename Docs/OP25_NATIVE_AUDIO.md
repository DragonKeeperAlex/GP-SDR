# OP25 decoder-tagged audio adapter

This is a receive-only GP-SDR adapter for the inspected boatbod OP25 source
`71abcd0ead32f86f51615ea6cc8a6a4dba4c949a`. It does not implement a new P25 decoder.
Upstream/source reference: https://github.com/boatbod/op25 . The patch and new
header use GPL-3.0-or-later, consistent with the containing decoder. Preserve
upstream notices and supply the matching patch/header with an adapted build.

## Contract

Set `GPSDR_P25_TAGGED_AUDIO=1` only with the matching decoder and GP-SDR build.
The environment is inherited by GP-SDR's OP25 child. Without the opt-in, the
upstream UDP PCM/control wire format is retained. With it, GP-SDR refuses bare
PCM rather than silently substituting a polled talkgroup. Diagnostics reports
received native headers, suppressed packets, or an adapter mismatch.

Each UDP datagram contains a 32-byte header and optional signed 16-bit
little-endian mono PCM at 8 kHz. UDP port still identifies the receiver stream.
Integers in the header are little-endian.

| Offset | Value |
| --- | --- |
| 0–7 | `GPSDP25` plus NUL |
| 8 | Version 1 |
| 9 | Kind: 0 PCM, 1 DRAIN, 2 DROP |
| 10–11 | Flags: bit 0 identity present, bit 1 crypto state known, bit 2 encrypted |
| 12–15 | Per-emitter packet sequence |
| 16–19 | Per-emitter call generation |
| 20–23 | Decoded talkgroup |
| 24–27 | Decoded source radio, 0 when unknown |
| 28–29 | NAC (may be configured/wildcard for Phase 2) |
| 30 | Signed slot: -1 FDMA, 0 or 1 TDMA |
| 31 | Observed algorithm; 0x80 is clear, 0xFF is an encrypted service-option hint |

The packet encoder reuses a bounded-size vector for ordinary 20 ms PCM blocks.
A generation changes on a decoded identity/crypto transition, an explicit HDU/
MAC_PTT start, and DRAIN/DROP. Explicit starts prevent merging two successive
calls with identical talkgroup/source when the preceding end was missed; repeated
starts can conservatively split a recording. A Go-generated receive-session ID
prefix makes saved native call IDs unique across decoder/session restarts.

Phase 1 takes group/crypto from corrected HDU and source/group from supported
unprotected standard Group Voice Channel User link control. Phase 2 takes source/
group/crypto from validated MAC_PTT. Encrypted LC/ESS hints apply to packet tagging
before the affected voice block without changing upstream decryption/key logic.
A call with no usable header/identity is deliberately omitted until the decoder
has sufficient evidence. Reset, termination, idle and timeout invalidate stale
native attribution. This can omit audio rather than invent an identity.

GP-SDR suppresses PCM with unknown identity/crypto, encrypted flags or a non-clear
algorithm before playback/recording/transcription. Tagged PCM must also agree
with recent polled group/source/slot metadata before archiving. Polled metadata
still supplies frequency, configured system/site and human labels; those fields
are not new packet-native evidence. Delayed/mismatching metadata can lose early
or trailing audio. Decoder errors, missed headers/terminators and real encryption
transitions still require RF/replay acceptance. This adapter does not by itself
prove complete call capture, real-RF per-talkgroup mute/solo/volume, multi-system routing,
real concurrent capacity or clear speech. It is a bounded software identity fix.

## Build, test and deploy

1. Preserve the installed library, GP-SDR binary, source and service configuration.
2. Run `python3 Scripts/op25/prepare_native_audio.py --source <OP25 source>`.
   It checks normalized hashes of all six touched upstream files before applying
   the patch, refuses unknown/partial modifications, and verifies the resulting
   hashes. The new helper header must also match. No radio settings are changed.
3. Build the existing configured tree's `gnuradio-op25_repeater` target, Release,
   with two compilation jobs. Do not run a broad install that replaces apps or
   unrelated libraries.
4. Compile/run `test_native_audio.cpp` for the independent header encoder. Link
   `test_native_emitter.cpp` with the build's `op25_audio.cc.o` (the class is hidden
   inside the shared library), then run `test_native_emitter.py <executable>`.
   These are socket-only synthetic fixtures; no SDR or speaker is opened.
5. Build GP-SDR and its application tests. With live P25 stopped to release its
   ports, set `GPSDR_TEST_NATIVE_EMITTER` to that emitter executable and run
   `TestOP25NativeEmitterTransport`. It sends three independent synthetic streams
   through GP-SDR's actual UDP listeners and hub, and verifies encrypted PCM does
   not reach ordinary audio subscribers. Native recorder and checkpoint recovery
   tests verify boundaries, metadata mismatch rejection and retained identity.
6. Atomically install only the reviewed library at the actual resolved installed
   path and the matching GP-SDR binary. Add the explicit environment opt-in to
   the GP-SDR service, restart, and restore the saved silent receiver profile.
   Keep runtime/library hashes and rollback privately with the matching source.
7. Check Diagnostics for tagged packets. Validate real calls separately; do not
   insert synthetic fixtures into the live journal to create a success claim.

Rollback: stop the GP-SDR service, restore the preserved library and binary,
remove only the native-audio environment override, reload service definitions
and restart/restore the saved profile. Retain original media/settings/calibration.
Source changes can be reversed with the matching patch; do not reset an upstream
checkout containing unrelated user edits.

## Browser talkgroup controls

For one-system profiles using the opt-in tagged adapter, live browser frames use
an identity that includes both the decoded talkgroup and native call generation.
The browser shares mute/solo/volume/pan controls per talkgroup while retaining
independent timing queues per call. Even two overlapping calls on the same
talkgroup retain separate scheduling clocks. Finished call clocks are removed;
unknown/unregistered groups start silent. Recorder and radio-relay stream IDs
remain unchanged. Browser listening controls do not mute the decoder or stop
recording; relay routing has its own controls. Solo no longer erases a different
talkgroup's manual mute in this mode.

Multi-system profiles retain the legacy receiver-stream path and do not expose
these volume/pan controls: the existing talkgroup mixer is keyed by numeric TG
rather than system identity, so claiming independent cross-system routing would
be incorrect. Independent multi-system talkgroup controls remain software work.
A stale frame already buffered by a browser subscription is rejected at the
HTTP writer after Stop/session replacement, not merely at the UDP reader.
These behaviors have mock-browser/routing/queue tests; real audio listening and
concurrent RF acceptance remain separate.
