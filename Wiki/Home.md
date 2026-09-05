# GP-SDR user guide

**Current coverage: 1.5.0-rc13 release candidate, including the earlier baseline.** Use [What’s New](Whats-New) to compare versions. The release candidate is not a final 1.5 release.

This wiki was checked against the source through **1.5.0-rc13** on September 4, 2026. GP-SDR brings tuning, scanning, P25 trunk following, optional digital decoders, multi-receiver mapping, recordings, and offline transcription into one console. It also includes a guarded, local-only HackRF WAV transmit workspace; receive workflows do not transmit.

[Download GP-SDR and release candidates](https://github.com/DragonKeeperAlex/GP-SDR/releases) · [What changed and how to use it](Whats-New) · [Troubleshooting](Troubleshooting-and-Limits)

## First successful session

1. [Install and complete first run](Getting-Started).
2. [Connect your radio and check its drivers](Hardware-and-Drivers).
3. [Tune a known signal and hear clear audio](Tuner-and-Live).
4. [Create or import a profile and start scanning](Profiles-and-Mixer).
5. [Review recordings and search the timeline](Activity-Recordings-and-Storage).

## Choose a task

| I want to… | Guide |
| --- | --- |
| Listen to one frequency or use the spectrum/waterfall | [Tuner and Live](Tuner-and-Live) |
| Scan a channel bank or import CHIRP files | [Profiles and Mixer](Profiles-and-Mixer) |
| Follow a P25 system or decode DMR, sensors, aircraft, paging, or AIS | [P25 and Decoders](P25-and-Decoders) |
| Find and identify activity across a band | [Mapper: Map, Discovery, and Identify](Mapper-Discovery-and-Identify) |
| Run separate jobs on several radios | [Multiple Receivers](Multiple-Receivers) |
| Compare receivers or antennas | [Receiver and Antenna Lab](Receiver-and-Antenna-Lab) |
| Find an earlier call, transcribe it, or limit capture storage | [Activity, Recordings, and Storage](Activity-Recordings-and-Storage) |
| Import local databases, sync ranges, or queue results to Sheets | [Data and Integrations](Data-and-Integrations) |
| Install optional decoders or whisper.cpp | [Optional Components](Optional-Components) |
| Operate from a phone or another computer | [Remote and Settings](Remote-and-Settings) |
| Set up an always-on Linux or Windows host | [Server Setup](Server-Setup-Linux-and-Windows) |
| Prepare HackRF audio-file transmission | [Transmit](Transmit) |
| Monitor a whole fitting channel bank with independent audio | [Band Monitor](Band-Monitor) |
| Collect now, analyze later, or alternate timed phases | [Analyze and Schedule](Analyze-and-Schedule) |
| Configure a local model and confirmed examples | [Local Intelligence](Local-Intelligence) |
| Try standalone Android HackRF USB reception | [Android Preview](Android-Preview) |

## Know what a result proves

A spectrum peak proves RF energy. A decoder candidate suggests which tool to try. Valid frames provide protocol evidence; intelligible unencrypted audio provides voice evidence. An automatic label or transcript can be wrong. Encrypted traffic is identified and skipped, never decrypted.

Optional decoders need their executables and dependencies installed. Phone/tablet browser access operates the receiver attached to the host computer. The separate [Android preview](Android-Preview) has experimental HackRF USB support and is not included in the desktop release downloads. See [What’s New](Whats-New) for release-specific evidence and remaining limits.

The source repository includes a matching `Wiki/` copy for maintenance. [Source and credits](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/THIRD_PARTY.md).
