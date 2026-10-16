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
