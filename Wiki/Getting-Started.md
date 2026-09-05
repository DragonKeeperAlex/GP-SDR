# Getting Started

In **1.5.0-rc9**, the navigation item **Channels** opens Channels & profiles (called **Profiles** in 1.4.1). Instructions referring to Profiles use this same workspace.

## Choose a package

### macOS 13 or newer

Use the Universal DMG unless you need the smaller Apple Silicon (`arm64`) or
Intel (`x86_64`) ZIP. Drag **GP-SDR.app** to Applications. Public builds are
ad-hoc signed rather than Apple-notarized, so the first launch may require
Control-clicking the app and selecting **Open**.

The native app launches its own receiver service and prevents idle system
sleep while the app is open. Display sleep is still allowed; quitting releases
the assertion. It serves a token-protected LAN console on a dynamically chosen
port. See [Remote and Settings](Remote-and-Settings) for the distinction from
the command-line server’s loopback default.

### Windows 10 or 11

Extract the complete x86_64 ZIP and keep its files together. Run
`GP-SDR.exe -open`. A compatible WinUSB driver may still be required for the
exact SDR interface; follow the Hardware page rather than replacing drivers on
unrelated USB devices.

For an always-on Windows receiver, follow [Linux and Windows Server
Setup](Server-Setup-Linux-and-Windows). It covers Task Scheduler, a persistent
token, private-network firewall access, updates, logs, and backups.

### Debian or Ubuntu

Install the package matching the computer:

```bash
sudo apt install ./gp-sdr_1.5.0-rc13_amd64.deb
# 64-bit ARM:
sudo apt install ./gp-sdr_1.5.0-rc13_arm64.deb
sudo systemctl enable --now gp-sdr
```

Open `http://127.0.0.1:8073/`. The service binds to this computer only by
default.

Continue with [Linux and Windows Server
Setup](Server-Setup-Linux-and-Windows) before enabling LAN access. Optional
decoders and transcription are covered separately in [Optional
Components](Optional-Components).

## First-run checklist

1. Connect one receiver directly, or through a powered USB hub.
2. Put a PortaPack in **HackRF USB mode**.
3. Open **Hardware** and press **Refresh**.
4. Select **Install** or **How to** on anything marked missing.
5. Open **Tuner** and select the receiver.
6. Choose a strong local broadcast FM frequency and WFM mode.
7. Begin with RF amplifier off, moderate gain, DC removal on, and squelch open.
8. Press **Start** and verify a station is intelligible—not merely visible.
9. Close squelch gradually and adjust gain only as needed.
10. Save calibration only after comparing against a known frequency.

## Demo mode

To inspect the interface without hardware:

```bash
gp-sdr -demo -open
```

Demo data is labeled and is never reported as received RF.

## Beginner and Advanced controls

Choose the interface mode under **Settings → Controls**.

- **Beginner** favors automatic sample rates, saved calibration, and a smaller
  set of controls.
- **Advanced** exposes manual RF gain, LNA/VGA, amplifier, antenna power,
  sample rate, PPM, DC and I/Q correction, and other tuning controls.

Changing modes does not erase saved profiles or calibration.

## Safe first test

A strong broadcast FM station is the best initial test because it separates
basic hardware/audio problems from trunking or decoder configuration. If it is
not intelligible in GP-SDR, fix receiver ownership, antenna, gain, sample rate,
squelch, and audio before attempting P25.

## Confirm the version and choose the next task

Download packages from [GitHub Releases](https://github.com/DragonKeeperAlex/GP-SDR/releases) and check **Settings → Build → Version** after installation. Package examples target 1.5.0-rc13. Keep the previous application and a stopped-service backup of your data when updating.

Once one known signal works, stop the Tuner before giving that same radio to a profile, Mapper, or P25. Continue with [Profiles](Profiles-and-Mixer), [Mapper](Mapper-Discovery-and-Identify), or [P25](P25-and-Decoders). On a phone, scroll the bottom navigation to reach all workspaces.

---

1.4.1 baseline source: [GPSDRApp.swift](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/macos/GPSDRApp.swift), [INSTALL.md](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Docs/INSTALL.md).

## Installing the current release candidate

The repository now also publishes **1.5.0-rc9**, with [Band monitor](Band-Monitor), [Analyze/Schedule](Analyze-and-Schedule), and [Local intelligence](Local-Intelligence). Use the explicitly labeled rc9 assets from Releases rather than assuming the latest-stable link selects them. In the package commands above, substitute the actual downloaded version’s filename. Stop analysis and capture before replacing the app, retain the data backup, and verify Version after restarting.

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
