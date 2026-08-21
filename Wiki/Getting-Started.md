# Getting Started

## Choose a package

### macOS 13 or newer

Use the Universal DMG unless you need the smaller Apple Silicon (`arm64`) or
Intel (`x86_64`) ZIP. Drag **GP-SDR.app** to Applications. Public builds are
ad-hoc signed rather than Apple-notarized, so the first launch may require
Control-clicking the app and selecting **Open**.

The native app launches its own local receiver service and prevents idle system
sleep while a receiver, Mapper job, or P25 session is active. Display sleep is
still allowed.

### Windows 10 or 11

Extract the complete x86_64 ZIP and keep its files together. Run
`GP-SDR.exe -open`. A compatible WinUSB driver may still be required for the
exact SDR interface; follow the Hardware page rather than replacing drivers on
unrelated USB devices.

### Debian or Ubuntu

Install the package matching the computer:

```bash
sudo apt install ./gp-sdr_1.1.1_amd64.deb
# 64-bit ARM:
sudo apt install ./gp-sdr_1.1.1_arm64.deb
sudo systemctl enable --now gp-sdr
```

Open `http://127.0.0.1:8073/`. The service binds to this computer only by
default.

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
