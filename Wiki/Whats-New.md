# What’s new through GP-SDR 1.5.0-rc15

## 1.5.0-rc15: live cleanup reconciliation

Automatic and manual storage cleanup now update the event journal immediately. Removed payloads no longer remain in the Analyze queue until GP-SDR is restarted.

## 1.5.0-rc14: bounded IQ storage

The configured IQ cap now includes wideband Archive and Pending captures. This closes the cap bypass that allowed long Mapper collection runs to consume hundreds of gigabytes. Low-value per-channel IQ is deleted after analysis when **delete junk** is selected; useful evidence remains compacted and retained.

## 1.5.0-rc13: decoder evidence precedence

Confirmed decoder frames can no longer be downgraded by a later heuristic or local-model candidate. Startup reconciliation restores valid historical frames from the event journal before Mapper counts are shown, and retired source files no longer remain in the analysis retry queue.

## 1.5.0-rc12: identification integrity

Mapper counts now cover the complete stored frequency set and distinguish analyzed candidates from strictly verified identities. False DSD-FME startup-banner “decodes,” shared analysis queue timeouts, contradictory local-model summaries, and additional standalone noise captions are corrected. Known affected evidence is safely requeued at startup; source recordings are not deleted. See the [rc12 release notes](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/RELEASE_NOTES_1.5.0-rc12.md).

## 1.5.0-rc11: capture preservation and recovery

- Original receiver IQ archive with shared files, SHA-256, and capture timing log.
- Full-duration filtered channel evidence with stronger filtering; removed recording cooldown.
- Overlapping detection windows across the full buffer.
- Event history beyond 25,000 with append-only updates; orphan-media recovery and visible missing-file warnings.
- Interrupted analysis becomes retryable; archives and pending IQ are protected from automatic deletion.

See [storage, capture quality and recovery](Activity-Recordings-and-Storage) for setup, disk requirements, and remaining capture gaps.


## 1.5.0-rc10 release candidate

- **Explore:** recorded-activity heatmaps, daily timeline, offline collection map,
  frequency/area groups, filters, bookmarks, tags, saved views, and capture details.
  Nearby reference areas are candidates, not confirmed transmitter locations.
  [Instructions and limitations](Explore).
- **Transcription:** conservative silence/static checks and standalone non-speech
  caption cleanup. Existing transcripts are preserved; hallucinations remain possible.
- **Analyze:** invalid worker counts cannot leave a run marked active; stopped runs
  clear active-stage indicators and canceled captures remain pending for retry.

Software and UI checks are not new live-RF acceptance evidence.

## 1.5.0-rc9 release candidate

The current repository adds the following workflows. Check the actual version under Settings; release-candidate downloads are listed on [Releases](https://github.com/DragonKeeperAlex/GP-SDR/releases).

- **Mapping navigation:** Overview, Discovery, Identify, Analyze, Schedule, and Results now have separate navigation entries. **Channels** opens the page formerly labeled Profiles.
- **Analyze:** process queued captures without opening an SDR; group by frequency and approximate receive area; choose up to 16 parallel frequency groups. Progress, stages, logs, and ETA refresh every five seconds while visible. [Instructions](Analyze-and-Schedule).
- **Collection throughput:** Channels at once now supports 1–1,024. Auto uses 512 for Discovery, 64 for Map, and one for Identify, bounded by usable bandwidth and spacing. Discovery defaults to manual deferred analysis; Identify defaults to live analysis and one channel.
- **Timed phases:** alternate Discovery and Identify for chosen durations, optionally repeating; Analysis timing determines when heavy processing runs. This is an in-process sequence, not a calendar scheduler. [Instructions](Analyze-and-Schedule).
- **Band monitor:** choose a channel bank and receiver; hear fitting channels from one capture with independent mute/solo/volume and CTCSS evidence. [Instructions](Band-Monitor).
- **Local intelligence:** optional localhost Ollama evidence summaries, confidence gating, manually confirmed examples, and JSONL export. [Instructions](Local-Intelligence).
- **Retention:** Identify can delete rejected IQ after analysis instead of quarantining it; choose the policy deliberately. [Instructions](Activity-Recordings-and-Storage).
- **Tuner:** click above/below a frequency digit to step it, use the reorganized receiver/RF/signal controls, and save the current frequency as a profile. Audio level auto is the clearer label for audio AGC.
- **Native Mac:** standard visible title bar/window controls and a native file-picker delegate for file inputs.
- **Android source preview:** standalone shared engine, HackRF USB receive, RTL-TCP, removable-storage selection, and rendering modes. Direct RTL USB and native P25 are not included. [Instructions](Android-Preview).

The [rc9 release notes](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) report software checks, packaging, and installed UI inspection; they do not establish every protocol’s live RF performance or complete capture-recovery behavior. Stop analysis before replacing the app; reopen Analyze to continue pending work.

## 1.4.1 baseline and earlier additions


Use **Settings → Build → Version** to check your installation. The following historical section describes the 1.4.1 baseline; rc9 changes above supersede its defaults. [Download releases](https://github.com/DragonKeeperAlex/GP-SDR/releases).

## 1.4.1: correct P25 receiver selection

P25 now matches the preferred receiver using the hardware serial expected by SDRTrunk. USB bus/port identifiers still serve configuration and ownership purposes. After updating, refresh Hardware, select the intended receiver on the P25 page, start it, and verify the selected hardware and decoded control identity. See [P25 setup](P25-and-Decoders) and [Multiple Receivers](Multiple-Receivers).

## 1.4.0: compact console and receiver fixes

- **Mapper:** select a workflow, receiver, and Range preset, then use the visible Start/Save buttons below the workflow controls. Job setup and receiver monitoring are separate; progress continues while editing. See [Mapper](Mapper-Discovery-and-Identify).
- **P25:** choose System profile and Receiver directly, press Start P25, search talkgroups, and use Active only. Saved HackRF gains can be retained or explicitly overridden; changes restart active P25.
- **Receiver diagnostics:** an enumerated HackRF with a diagnostic warning remains available for receive testing and dry runs. Actual RF transmission stays blocked until the warning is resolved.
- **RTL-SDR:** persistent capture sessions reduce repeated USB teardown. macOS discovery avoids claiming the tuner just to refresh a menu. A USB dropout still requires a physical connection check.
- **Transmit:** signed IQ, complete-WAV processing, and AM carrier-offset corrections improve generated waveforms. See [Transmit](Transmit) for the required format and dry-run workflow.
- **Mobile layout:** compact controls and corrected navigation scrolling keep every workspace reachable.

## Features added across the recent releases

| Feature | How to use it |
| --- | --- |
| Default Map workflow | Run a repeating range pass that combines discovery, local classification, available decoding, and enabled transcription. |
| Independent Mapper jobs | Assign one job per idle radio; Use all connected receivers copies a template to available radios. |
| Software VFO batching | In 1.4.1, set Channels at once from 1–32; rc9 increases the limit as described above. |
| Selective Identify | Set minimum hits, Discovery-only hit history, successful-check percentage, recency, maximum channels, and ordering. |
| Evidence-based identification | Expand results and distinguish Successfully identified from candidate labels; CSV retains the verification reason. |
| Decoder routing | Choose Auto or a specific decoder per Mapper job; choose digital modes in Tuner or profiles. |
| Automatic RF tuning | Leave Mapper gain/sensitivity/amplifier on Auto, monitor clipping and the current decision; use Advanced for manual control. |
| Managed IQ | Inspect Analyzing/Retained/Rejected usage and set the rejected-IQ recovery period under Settings. |
| Receiver/antenna comparison | Select radios, bounds, points, and dwell in the Hardware lab, run, then export CSV. |
| Indexed activity search | Search Timeline by transcript, callsign, protocol, system, or frequency and replay available recordings. |
| Local database and bulk import | Import multiple channel files or scan an official-export folder; review source and receive-area coverage. |
| Sheets integration | Sync read-only ranges or send reviewed Mapper observations to an Additions Queue. |
| Remote/headless operation | Use the host’s tokenized address; configure a service/task and capture limits for unattended use. |

Each task is linked from [Home](Home); the sidebar covers the full manual.

## What has and has not been verified

The 1.4.1 release notes report a physical HackRF P25 control decode with the intended receiver and no preferred-tuner fallback. The 1.4.0 notes also report HackRF IQ captures and voice recordings. These are bounded observations, not a guarantee for every radio, system, frequency, or OS.

Long-duration RTL-SDR reliability and simultaneous two-radio acceptance remain unresolved in those release notes. Optional protocols were not all exercised on-air. No RF transmission was performed for the 1.4 release validation. Microphone input, digital-voice transmission, DCS decoding, and a trained RF classification model were not added.

Primary release records: [1.4.1](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/RELEASE_NOTES_1.4.1.md), [1.4.0](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/RELEASE_NOTES_1.4.0.md), [1.3.0](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/RELEASE_NOTES_1.3.0.md), [1.2.0](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/RELEASE_NOTES_1.2.0.md). Older feature-status documents describe their named versions and are not the current support matrix.
