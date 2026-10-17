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

## Windows desktop priority guard — October 17, 2026

The deployed Pi's remote evidence endpoint now uses the home PC's guarded
Ollama endpoint on port 11435. The existing Ollama service remains on 11434.
The guard runs as the interactive Windows user through the
`GP-SDR-Home-Compute-Guard` logon task, with limited privilege and no password.
It is a small native Go process; it does not poll GPU tools or run another model.

Default behavior:

- Pause background model requests during mouse/keyboard use; resume after ten
  idle minutes. Recent XInput controller use also holds a ten-minute pause.
- Keep background work paused while a foreground application covers its monitor
  (including fullscreen games/video). This deliberately favors foreground use.
  Borderless/maximized behavior depends on window/monitor geometry; this is not
  an exhaustive game/process detector. Non-XInput controllers may not be seen.
- Reject new requests with a retryable HTTP 503 and interrupt an active guarded
  request within the one-second activity polling interval. Ask Ollama to unload
  that request's model; VRAM release may take a few seconds. Every guarded
  inference also has `keep_alive: 0`, preventing idle model residency.
- GP-SDR's durable remote evidence queue retains interrupted jobs and retries
  automatically with its existing backoff (up to five minutes). Resumption may
  therefore occur after the idle delay plus retry backoff. Generation restarts;
  it does not resume from a partially generated token.
- Unknown desktop state fails closed. When logged out or the guard is stopped,
  its endpoint is unavailable and the Pi queue waits/retries. The task starts
  at login, not in Session 0; Windows input-idle information is session-specific.

Three shortcuts are installed on the Windows Desktop:

1. **Pause GP-SDR AI** — persistent manual pause, including across login/restart.
2. **GP-SDR AI - Automatic** — remove manual pause and restore the idle policy.
3. **GP-SDR AI Status** — show the current pause/processing state.

Automatic mode never overrides current desktop activity. No keyboard contents,
window titles, game names or prompts are logged by the guard. A local status
snapshot holds only the pause reason, idle duration and active-request flag.

### Reproduce and verify

Build from `server/`:

```
GOOS=windows GOARCH=amd64 go build -ldflags '-H=windowsgui' -o gp-sdr-home-guard.exe ./cmd/home-compute-guard
```

Copy the executable and `Scripts/windows/{Install-HomeComputeGuard.ps1,
HomeComputeControl.ps1}` together. Run the installer elevated, supplying the Pi's
private LAN IPv4 address with `-PiAddress`. It creates a firewall allow rule for
that exact executable, TCP port and Pi source address. It supports a LAN Windows
has categorized Public without changing the network category. It disables only
conflicting Windows-generated Query User blocks for this executable. No public
Internet port forwarding is required. Keep both hosts' LAN addresses stable.

Preserve the Pi LocalAI configuration, change only `remoteEndpoint` to the PC's
port 11435 and retain the existing remote model/local endpoint. Future raw-worker
or other background clients must explicitly use 11435 to receive this policy;
clients connecting directly to Ollama 11434 bypass the guard. The completed
one-off raw-capture batch is not a recurring worker.

`GET /guard/status` reports the pause state; allowed inference APIs are
non-streaming `/api/generate` and `/api/chat`. Model download/delete and remote
pause overrides are not exposed. Read-only Ollama tags/version/process queries
remain available while paused. One inference runs at a time.

Verification on October 17: guard running in desktop Session 1; startup task
registered with Interactive logon; actual recent input produced paused status;
Pi inference received 503/Retry-After 30 without loading a model; manual pause
marker and restoration to Automatic verified. All three shortcuts exist and
PowerShell scripts parse. Policy, forwarding, cancellation/unload, manual resume,
and unsupported-API regressions passed locally (race/vet) and on Windows.
No game was launched or desktop input injected. Actual gameplay latency and an
unattended login/reboot remain untested. No saved RF/media or password changed.
The preceding historical metadata batch finished **256/256, no reported errors**.

Private rollback configuration is retained on the Pi. To roll back, restore the
previous remote endpoint, stop/unregister the guard task and remove only its
named firewall rule. Leave the original Ollama task/configuration untouched.

API references: [Microsoft input-idle session semantics](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getlastinputinfo),
[Ollama model residency control](https://docs.ollama.com/api/generate).
