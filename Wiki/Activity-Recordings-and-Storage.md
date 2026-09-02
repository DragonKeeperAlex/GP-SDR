# Activity, recordings, transcription, and storage

**rc9 retention change:** Identify can select **Delete after analysis**, which removes rejected IQ without a quarantine window. Choose **Keep briefly in quarantine** before collecting if you need the recovery timer below. [Analyze and Schedule](Analyze-and-Schedule) covers queued processing.

## Find and replay a transmission

1. In a custom profile, enable audio recording before starting reception. Enable transcription too if the local component is ready.
2. Open **Activity → Signals** for repeated activity grouped by frequency. Filter by frequency, identity, or type. The share button exports a frequency as a profile.
3. Switch to **Timeline** for individual events. Search by a frequency, label, callsign, protocol, system, talkgroup, or words in a transcript.
4. Inspect the time, frequency, type, signal level, duration, transcript, and any displayed talkgroup/source or CTCSS information.
5. Press the recording’s **▶** button to play or pause when audio is available. Check master mute and volume if playback is silent.

The timeline is a bounded view of indexed history, not a complete export of every stored event. Search narrows the returned events. An event can exist without an audio file: recording may have been disabled, a decoder may have supplied metadata only, or capture retention may have removed the file. Demo events are explicitly marked.

## Enable local transcription

1. Open the Transcription integration card and install whisper.cpp plus its model; see [Optional Components](Optional-Components) for platform setup.
2. Restart/refresh as required until the component is Ready.
3. Enable transcription in the recording profile, or **Dictate voice** for a Mapper workflow that exposes it.
4. Receive intelligible, unencrypted speech and allow local analysis to finish.
5. Read the transcript in Timeline or expanded Mapper evidence, then compare it with the recording before treating names or callsigns as confirmed.

Transcription runs on the GP-SDR host. A weak, noisy, encrypted, or incorrectly demodulated signal does not become reliable text just because transcription is enabled. Large models and several simultaneous channels can delay analysis.

## Set bounded capture storage

Open **Settings → Data**:

1. Set **Recording cap · GB** and **IQ cap · GB**; zero disables that cap.
2. Choose **Keep captures** for the age limit; No age limit disables age-based removal.
3. Enable **Automatic cleanup** if you want the saved policy enforced automatically. It is off by default.
4. Press **Save limits**.
5. Use **Clean now** only after reviewing the limits and confirming removal.

Cleanup targets old files in GP-SDR’s Recordings and IQ directories and protects files modified in the last ten minutes. It does not remove profiles, Mapper result history, calibration, or imported channel databases. Saving limits is distinct from enabling automatic cleanup. Profile event retention and capture-file limits are separate controls.

## Understand managed Mapper IQ

| State | Meaning |
| --- | --- |
| Analyzing | Waiting for or running local classification, available decoding, and enabled transcription. |
| Retained | Evidence kept after analysis. It is still subject to any applicable general capture limits. |
| Rejected | Low-value evidence held temporarily when the job uses quarantine; rc9’s Delete after analysis policy removes it without this window. |

Mapper channelizes wide captures before archiving narrowband evidence to reduce storage. Manual/after-job analysis leaves evidence pending until processing. In rc9, the following recovery controls apply to quarantine, not immediate deletion. **Remove rejected IQ** is a separate setting, enabled by default. Choose **Rejected IQ recovery** from one hour through seven days; the default is 24 hours. Disabling general Automatic cleanup does not disable this rejected-IQ timer.

The recovery period means the rejected files remain on disk temporarily; it is not a documented one-click restore interface. If evidence matters, stop the relevant job, disable its impending cleanup as needed, and back up the data before its recovery period expires. Keep a copy outside the managed capture directories.

## Back up and restore

Stop capture jobs and quit/stop the service before copying its data directory. The native macOS data is normally under `~/Library/Application Support/GP-SDR`; the packaged Linux service uses `/var/lib/gp-sdr/GP-SDR`. A server started with `-data` uses that chosen location. Preserve the entire directory for profiles, histories, calibration, and linked media.

Restore with the service stopped, preserve the existing directory as a recovery copy, and use the same data path. Check profile names and a known recording after restarting. [Server Setup](Server-Setup-Linux-and-Windows) includes platform update instructions.

---

1.4.1 baseline source: [app.js](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/app.js), [storage_policy.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/storage_policy.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
