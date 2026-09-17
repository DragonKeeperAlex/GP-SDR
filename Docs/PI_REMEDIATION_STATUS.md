# Pi remediation status

Primary platform: Raspberry Pi/Linux ARM64. Mobile ports paused until main-app acceptance. Authoritative scope: [master backlog](MASTER_REMEDIATION_BACKLOG.md).

## Storage startup guard

The current Pi uses `/mnt/gp-sdr-data/GP-SDR` on an exFAT data drive. Without a mount dependency, the service could start against the underlying internal-card directory. The site-specific `Scripts/pi/gp-sdr-storage.conf` drop-in requires the mount and checks the data directory is writable before launch. It preserves the base service and its private access settings.

Install only on this configured Pi:

```
sudo install -d /etc/systemd/system/gp-sdr.service.d
sudo install -m 0644 Scripts/pi/gp-sdr-storage.conf /etc/systemd/system/gp-sdr.service.d/storage.conf
sudo systemctl daemon-reload
```

This does not restart GP-SDR. Test a subsequent normal restart before acceptance. Do not unmount the live data drive as a casual test. The guard prevents startup fallback; it does not establish hot-unplug, exFAT crash repair, or NAS recovery acceptance. Existing data is not moved or deleted.

## September 16, 2026 baseline

## Headless P25 backend selection

`Scripts/pi/gp-sdr-p25.conf` sets `GPSDR_P25_ENGINE=op25` on this Pi only. The application also accepts `auto` (default) or `sdrtrunk`; invalid values fail clearly, and explicitly selecting SDRTrunk for Pluto fails rather than silently choosing another engine. OP25 must already be installed. This configuration uses the existing direct UDP PCM bridge rather than nonexistent native sound hardware. It does not complete the separate SDRTrunk PCM bridge backlog item.

Install as `/etc/systemd/system/gp-sdr.service.d/p25.conf`, reload systemd and restart GP-SDR after preserving job settings. HackRF P25 LNA/VGA/amp overrides are now passed to OP25 instead of ignored. RF-02/03 and UI-04 remain open until their full acceptance gates pass.

- Service active; two HackRFs, Pluto and RTL enumerate on USB.
- Pi `get_throttled` is `0x0`.
- Root drive approximately 41 GB available; data drive approximately 477 GB available.
- Current isolated HackRF SDRTrunk P25 test remains searching; no new HackRF lock claim.
- Pluto previously independently locked EBRCS with OP25 and delivered decoded PCM. Subjective audio acceptance remains open.
- MAP-09, MAP-11 and DIST-01 remain open despite startup-guard work.

## Post-install verification

September 16: storage-guard service restart succeeded and API accepted the isolated HackRF `c5cb` OP25 run with 10 MS/s, gain overrides, calibration forwarding and 100 kHz offset. A subsequent 20-second silent live-audio check produced no audio file/frames; P25 remained searching. This is NOT a HackRF lock/audio pass. Unit suite, vet and race checks passed locally. Pluto/RTL control comparisons and raw IQ diagnosis are the next RF-02 steps. No mobile port, public release or Mac install was changed.

## Raw-IQ comparison and OP25 metadata correction

September 16 continuation, Pi `1.5.0-rc33-test`: captured 20 million complex signed-byte samples per HackRF at 774.55625 MHz, 10 MS/s, 8 MHz filter, LNA/VGA 24/24, amp off. `/tmp/gpsdr-c5cb-current.iq` had 100% negative I and Q (means -125.03/-123.32, only 10 unique values each). This reproduces the prior pinned-sign sample corruption; withdraw any assumption that c5cb is the known-good unit. `/tmp/gpsdr-a447-current.iq` was centered and varying (means 0.36/2.09, 48/47 unique values). Centered samples do not establish sensitivity or P25 acceptance. A subsequent isolated a447 OP25 test did not lock in 25 seconds; Pluto locked again under the same site selection.

Corrected the OP25 active-call reader, which previously read SDRTrunk CSV files even when OP25 was selected. It now parses OP25 channel updates, rejects control-channel/null talkgroup and encrypted entries, expires stale state, and adds discovered talkgroups to the mixer. Unit, vet and race checks passed; ARM64 build installed on this Pi only. Live `/api/mixer` returned active talkgroup 2436 on EBRCS CCCO Central after restart. This verifies live metadata propagation, NOT per-talkgroup PCM routing or intelligibility: audio still uses receiver-stream rows. RF-01/02/04 remain open. No data/results cleanup, mobile work, public release or Mac replacement occurred.

## Continuous analog audio and browser queue fixes

September 16 continuation: tuner and Band Monitor now preserve discriminator, oscillator, decimation remainder, AM DC filter and FM deemphasis state across frames. Previously every tuner 25 ms block restarted these states. Retuning/mode/rate/format changes reset state; inactive Band Monitor channels discard old state. Chunked-versus-uninterrupted regression vectors pass for AM/NFM/WFM, including non-aligned blocks and VFO reset. These vectors are software acceptance, not real AM/NFM reception proof.

Browser playback now tracks scheduled sources, stops old sources before recovering from an excessive backlog, clears them on disconnect/Stop, and releases finished nodes. `node Scripts/test_audio_queue.cjs` verifies continuity, channel-isolated recovery, completion and stop using a mock audio context; JavaScript syntax passes. All Go tests, vet and race tests passed before installing the Pi build.

Actual Pluto WFM at 98.1 MHz, 2.4 MS/s, gain 40, open monitor: initial 15-second endpoint capture contained 502 complete frames / 12.55 seconds PCM; after continuous DSP changes, a 20-second capture contained 701 frames / 17.525 seconds PCM, 48 kHz, RMS 8531.5, peak 30029. Capture starts during tuner startup, so these durations are not steady-state drop-rate measurements. Files `/tmp/gpsdr-pluto-wfm-audio.bin` and `/tmp/gpsdr-pluto-wfm-continuous.bin`; trailing partial packet is expected from forcibly stopping curl. No headphone playback performed. Clear subjective audio and other receiver/mode acceptance remain OPEN.

RTL WFM at 98.1 MHz, 2.4 MS/s, gain 30, open monitor after the final Pi install: 765 complete frames / 19.125 seconds PCM in a startup-inclusive 20-second capture; 48 kHz, RMS 7424.0, peak 29724. File `/tmp/gpsdr-rtl-wfm-continuous.bin`. Service active, throttling 0x0. This is delivery evidence only; other modes and intelligibility remain unaccepted. Pluto P25 profile restored after the tuner comparison.
