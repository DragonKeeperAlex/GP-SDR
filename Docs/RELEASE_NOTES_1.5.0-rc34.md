# GP-SDR 1.5.0-rc34 — Pi audio and receiver maintenance

This release candidate prioritizes Raspberry Pi/Linux ARM64. It is not an all-features acceptance release. Mac/mobile installs are unchanged.

## Fixes

- Preserve continuous AM/FM DSP state across tuner and Band Monitor frames; reset on channel configuration changes.
- Stop persistent RTL tuner publication when switching modes; live RTL→Pluto P25 transition no longer contains old tuner audio.
- Stop queued browser sources during backlog recovery, disconnect and Stop; disconnect finished audio nodes.
- Read actual OP25 call metadata, exclude control/encrypted/stale entries, discover talkgroups, clear inactive mixer rows, and attribute events to the actual backend.
- Honor Pi OP25 engine selection, HackRF gain/amp/calibration settings, and correct Pluto device arguments.
- Prevent Band Monitor setting changes from replacing unrelated sessions; omit unsupported HackRF controls and enforce receiver rate bounds.
- Reject channel banks that cannot fit the selected receiver instead of presenting sequential scanning as simultaneous monitoring.

## Verification

Go unit, vet and race checks; JavaScript syntax; audio queue and Band Monitor regression scripts. Real Pi Pluto/RTL 98.1 MHz WFM endpoint captures, Pluto OP25 CCCO Central control lock and PCM, and a clean RTL WFM→Pluto P25 transition. See `Docs/PI_REMEDIATION_STATUS.md` for exact evidence. PCM delivery is not subjective intelligibility acceptance.

## Package and upgrade

The Linux ARM64 tarball contains the server and documentation. Existing Soapy helper, radio drivers, OP25, Java/SDRTrunk and optional decoders are **not bundled**. On the configured development Pi those are separate installed dependencies. Do not treat this tarball as a clean-machine, plug-and-play installer. Preserve the existing service, token, data directory and mount settings. Stop GP-SDR, back up its executable, replace only the executable, then restart. Never delete the data directory during upgrade.

`Scripts/pi/gp-sdr-storage.conf` and `gp-sdr-p25.conf` are site-specific examples, not universal defaults. Do not apply the storage guard unless `/mnt/gp-sdr-data` is actually the intended mounted data drive.

## Still open

Per-talkgroup OP25 PCM routing/mute/solo, clear audio listening acceptance across all modes, live SDRTrunk headless audio, HackRF P25 acceptance, second Pluto RX/TX channels, DMR/other decoder real-frame acceptance, Mapper accuracy/storage/multi-radio soak, UI completion, and clean cross-platform dependency packages. c5cb currently returns corrupted raw IQ; a447 has centered samples but P25 lock is not verified. Mobile ports remain paused. No old dependency bundle is republished.
