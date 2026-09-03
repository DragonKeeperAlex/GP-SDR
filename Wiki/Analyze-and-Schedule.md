# Analyze stored captures and schedule mapping phases

**Added through 1.5.0-rc9 (release candidate).** These controls are under **Mapping → Analyze** and **Mapping → Schedule**.

## Collect now, compute later

1. Configure a Mapper job on Discovery or Identify and choose its receiver/range or eligible channels.
2. Open **Compute & schedule → Analysis timing**. Choose **Live · analyze while scanning**, **After job stops**, or **Manual · compute later**.
3. For faster collection, use Manual. Changing the workflow to Discovery applies manual analysis and a short dwell by default; recheck settings after switching workflows or loading a saved job.
4. Save/start the job and confirm hits and queued capture evidence. This defers decoder, transcription, and model work; detection and bounded evidence collection still run.
5. Stop collection when ready. Manual work waits for you; After job stops attempts to start analysis when that job finishes without an error.
6. Open Analyze and inspect Files queued before pressing Start analysis.

This processes event records already queued by GP-SDR. It is not an arbitrary folder importer for unknown IQ files, nor a promise to recover every historical capture. If another analysis is already running, finish/stop it and start pending work explicitly.

## Run and monitor stored analysis

1. Install the optional decoder/transcription/model components you intend to use. Analysis can run without an attached SDR.
2. Choose **Parallel frequencies**: Auto, 1, 2, 4, 6, 8, 12, or 16. Auto derives a worker count from CPU capacity, capped at eight, then limits it to available groups.
3. Press **Start analysis**. Read Files queued, Frequency groups, Completed/failed, Elapsed, ETA, Current frequency, Stage, and the list of active groups.
4. Follow the **Live analysis log**. New entries appear at the bottom; the visible page refreshes about every five seconds. **Clear view** clears the displayed log view, not recordings or results.
5. Inspect updated evidence under Mapping → Results and Activity. A completed file does not mean every optional decoder or model succeeded, or that the signal was identified. Missing/unsuccessful optional stages can yield no evidence.
6. Press **Stop** to request cancellation. Allow the state to settle; unprocessed records remain queued. Review any error before restarting. Stop analysis before installing an update or removing capture storage.

Groups combine the same frequency within an approximate receive area (coordinates rounded to a tenth of a degree). Unknown-location captures share an unknown-area group. This is approximate grouping, not proof that two recordings are the same transmitter. Each event is processed, then available group evidence can be correlated by the local model.

**Parallel frequencies is different from Mapper Channels at once.** It controls offline worker groups, not RF capture width. Ollama model inference is serialized to limit memory use even with multiple workers; decoder and other stages can overlap. Raising workers will not create parallel local-model inference.

## Timed Discovery → Identify

1. Open Mapping → Schedule and configure the job’s range, step, receiver, Identify eligibility, and listening time.
2. In Compute & schedule, enable **Timed Discovery → Identify**.
3. Enter Discovery duration and Identify duration using minutes, hours, or days. Both phases must be between five seconds and seven days at the service level; use the offered UI units/limits.
4. Choose Analysis timing. For deferred post-collection work, select After job stops. If Repeat phases until stopped is enabled, the job continues until stopped, so after-job analysis waits for that end.
5. Enable **Repeat phases until stopped** only if you want repeated cycles. Otherwise one Discovery/Identify cycle finishes the job.
6. Save/start, watch the active phase and progress in Overview, and use the job’s Stop control when needed.

This is relative timing within a running GP-SDR process. It is not an at-a-clock-time schedule, a system service, or a guaranteed resume-after-reboot feature. Keep the host awake and storage available.

## Full auto

Overview’s **Full auto** starts the current range across available receivers using Map and automatic RF settings, with Analysis timing set to After job stops. Check range, step, storage, and receiver availability first. It does not automatically partition the range or mean all of each radio’s frequency range will be surveyed. Stop the collection to trigger after-job work, then check Analyze.

## Retention before running analysis

Identify’s **Rejected IQ** offers **Delete after analysis** (default) or **Keep briefly in quarantine**. Deletion removes low-value IQ and its sidecar after finalization; the quarantine recovery timer does not protect a file selected for immediate deletion. Choose quarantine before collection when you need a review window, and back up important evidence outside managed storage.

Read [Activity and Storage](Activity-Recordings-and-Storage) and [Local Intelligence](Local-Intelligence) before a large unattended run.

Source: [deferred analysis](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/deferred_analysis.go), [job phases](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/live_survey.go), [controls](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).

## Archive analysis and retry in rc11

Original-IQ jobs can defer waveform work and audio creation until analysis. Pending work is no longer limited to 25,000 events. Startup resets interrupted/canceled/timed-out work to Pending. Missing referenced files fail explicitly instead of being marked complete. Original shared IQ is immutable through analysis and retention. See [storage and recovery](Activity-Recordings-and-Storage).
