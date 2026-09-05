# GP-SDR 1.5.0-rc14

This release prevents wideband collection from silently bypassing configured storage limits.

- IQ caps now include Archive and Pending captures.
- Cleanup removes disposable/quarantined and oldest archive material before pending analysis and retained evidence.
- Recently written files remain protected while active.
- Per-channel IQ already follows the analysis lifecycle: low-value samples are deleted when the job selects **delete junk**, while useful decoded, transcript, callsign, protocol, strong-signal, or confident waveform evidence is retained in compact channel-rate form.

The local maintenance performed alongside this release preserved profiles, calibration, results, recordings, pending analysis, and retained evidence while removing an obsolete 184 GB raw archive and reproducible application/build backups.
