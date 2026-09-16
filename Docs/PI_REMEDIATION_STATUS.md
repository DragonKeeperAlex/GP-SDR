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
