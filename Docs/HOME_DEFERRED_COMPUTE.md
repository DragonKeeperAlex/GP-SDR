# At-home deferred evidence analysis

The Pi keeps receiver control, RF capture, protocol decoding, recording and the
initial DSP work. A configured home Ollama server can process queued evidence
summaries. This is **metadata analysis**, not remote raw-IQ decoding or remote
speech transcription. Audio and IQ files remain on the Pi.

The model receives the bounded DSP/decoder metadata, available transcript and
location/reference context used by the existing evidence analyst. Findings need
real decoded evidence before being treated as confirmed protocols.

## Operation

- Keep Ollama available at a private-network IP endpoint reachable from the Pi.
  The current endpoint validator requires an IP rather than a hostname; keep
  that address stable in the home network.
- Preserve the Pi's local model endpoint. Configure the separate remote endpoint
  and model in GP-SDR's LocalAI configuration.
- October 16 bounded checks used the already-installed `qwen3.5:9b` model with a
  2,048-token context. Larger models remain a separate performance decision.
- The live service polls queued work every 30 seconds and processes one job at a
  time. Failed jobs retain their metadata and retry with increasing delay, up to
  five minutes. Interrupted running jobs become eligible again after restart.
- The queue holds at most 1,000 jobs, pruning retained completed/discarded entries
  when capacity is needed. Existing failed/pending jobs are not silently dropped.
- A remote request has its own semaphore so it cannot occupy the Pi's first-pass
  analysis slot. Completed results are applied once rather than rewritten on
  every poll. The queue retains the submitted evidence and remote result.
- Successful P25 transcripts can enqueue their evidence directly for the home
  server. Existing live/deferred analysis fallback paths also use this queue.
  This does not automatically resubmit the entire historical archive.

The Local Intelligence status text shows the configured home model and queue
counts/errors. Saving the Pi's local-model settings preserves remote settings.
Configuration backups and the actual home address remain private on the Pi.

## Bounded verification

`TestConfiguredRemoteEvidenceAnalysis` is opt-in through
`GPSDR_REMOTE_AI_TEST_ENDPOINT` and `GPSDR_REMOTE_AI_TEST_MODEL`. It sends only
synthetic metadata, has a 45-second deadline, and does not create a live result.
Default test runs skip it. Queue regressions cover interruption recovery,
corrupt/failed persistence, exclusive remote processing, retry backoff,
independence from the first-pass semaphore and applying results only once.

A responding Ollama API does not establish Windows login/startup persistence or
an unattended power-cycle test. These require a separate host-level check.

October 16 live evidence: the installed Pi passed the opt-in synthetic structured
request in 5.75 seconds. One previously recorded call's metadata was queued,
processed automatically on the home server and applied once. Its audio remained
on the Pi. Queue summary refreshes every five seconds while Settings is visible.
No Windows host configuration was changed; SSH login/startup verification remains
separate from this working Ollama connection.

October 16 host follow-up: recovered the prior SSH login from the original setup
chat and verified key-based access. The Windows host reports an RTX 5080 with
16 GB VRAM, 128 GB RAM, and its existing LocalAI-Ollama task enabled with a boot
trigger and currently running. The Ollama firewall rule permits the private LAN.
No Windows settings changed; an unattended reboot was not exercised.


### October 16 — P25 stream boundaries, home batch and integration audit

Native OP25 DRAIN/DROP markers now reach recorder-only subscribers in order.
DRAIN closes the current segment, DROP revokes buffered/checkpointed audio,
and sequence gaps persist a partial-recording warning. Metadata without PCM no
longer creates an empty voice recording. Stale/encrypted/unassigned PCM is rejected.
Raw OP25 PCM still lacks packet-native call/talkgroup identity: this does not close
sample-level attribution/encryption-transition or real overlapping-RF acceptance.

Eight copied historical IQ captures (40 MB total) passed SHA-256 checks, completed
native Windows DSP and local-model analysis without failure, and were imported
into the Pi event history. Original paths/media/retention were preserved; journal
backup is private. No new digital-frame or speech-accuracy acceptance is claimed.
A separate resumable metadata-only home review covers 256 groups; it was still
running at the last checkpoint (191 complete, 65 remaining, no reported error).

The developed-feature audit integrated reviewed RTL stream recovery, Pluto
transport selection, Mapper identity reconciliation, home-analysis fallback,
FM discriminator limiting, tool detection and missing Pi helper source. See
DEVELOPED_FEATURE_INTEGRATION_AUDIT.md for decisions and deferred prototypes.
USB recovery recognizes one usable HackRF plus RTL, permits Ethernet Pluto,
and defers bus probes while the app runs or its status is unknown. Pi status
screen selects its framebuffer by panel identity and reports boot/data capacity.

Validation: full Go suite and exact staged-source suite; focused race/vet;
ARM64 and Windows builds; recorder/control/spool fixtures on actual Pi exFAT
and Windows; UI contract checks; USB presence fixtures; sudoers syntax checks.
These are bounded software checks, not RF overlap, listening, power-cut or
physical-display acceptance. P25 is running with the saved three-radio silent
profile; latest reception was searching. HackRF remains at least 8 MS/s.
Deployed ARM64 SHA-256:
`fa9c042672ff2fa277d6cd7b68924d9d7ad74fb3049ceedf00cc7af1ed5e2ab5`.
Group 4 remains open for packet-native attribution and remaining acceptance gates.
Credentials, recordings, transcripts and private batch manifests are excluded.
