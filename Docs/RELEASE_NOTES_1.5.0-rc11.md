# GP-SDR 1.5.0-rc11

Fixes mapping data preservation and makes recording limitations visible.

- Removes the 25,000-event retention cap. Original events and append-only updates survive analysis and restart; UI pagination no longer limits stored history.
- Adds Original IQ archive mode: exact receiver bytes, original rate/format, shared across channel events, checksummed and saved before correction. Quiet capture intervals are retained too.
- Removes the two-second channel IQ cutoff and 30-second recording cooldown. Filtered channel mode uses a windowed-sinc FIR before decimation.
- Checks completed captures using overlapping FFT windows across the entire buffer, including bursts the old sparse windows missed.
- Records capture intervals, receiver settings, target lists, failed requests and sample-derived durations.
- Recovers recognizable orphan media without fabricating identities, flags missing links, preserves interrupted journal tails, and requeues interrupted analysis.
- Protects original archives and pending IQ from automatic deletion. Archive capture stops at its storage budget or a write failure. Existing audio/completed-channel retention policies remain applicable.
- Defers per-channel audio creation for manual/after-job archives until Analyze runs. Existing settings remain compatible; choose Capture quality in job settings.

## Validation and limits

Regression tests cover history beyond 25,000 and restart, journal interruption recovery, burst detection between former sample windows, alias rejection/passband preservation, exact original-byte retention, shared archive safety, capacity stopping, and idempotent media recovery. Full Go/race/static checks and JavaScript checks are required before packaging.

Scanning still leaves tuning, capture-start and processing gaps. Host timing is not hardware timestamping; this release does not promise gap-free capture, calibrated sensitivity, reliable decoding of every protocol, or accurate transcription of arbitrary radio audio. Original IQ at 20 MS/s consumes about 144 GB per captured hour. The full event index currently grows in RAM with history.

Back up Data and Profiles before upgrading; keep both events.jsonl and event-updates.jsonl. Startup recovery preserves existing media and cannot reconstruct files already missing. Stop the old app before installation. See the updated storage and mapping Wiki pages for archive settings and the offline -repair-media command.
