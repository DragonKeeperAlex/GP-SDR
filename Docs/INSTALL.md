# Installing GP-SDR

GP-SDR is receive-only. The release packages contain the GP-SDR interface,
analog DSP, SDRTrunk P25 Phase 1/2 engine, and JMBE Creator. Hardware access is
provided by user-space receiver tools or the operating system's USB driver.

## Before installing

1. Download the package for your operating system and `SHA256SUMS.txt` from the
   same release.
2. Verify the package checksum before opening it.
3. Connect one receiver at a time for first setup. Avoid unpowered USB hubs.
4. A PortaPack must be placed in **HackRF USB mode**. Seeing a PortaPack menu or
   mass-storage device does not mean the Mac or PC can stream IQ samples.

## macOS 13 or newer

Use the current `GP-SDR-*-macos-universal.dmg` unless you specifically need the smaller
Apple Silicon (`arm64`) or Intel (`x86_64`) package.

1. Open the DMG and drag **GP-SDR.app** to Applications.
2. Control-click the app, choose **Open**, then confirm **Open**. The public
   package is ad-hoc signed but is not Apple-notarized.
3. Open **Hardware** and press **Refresh**.
4. If a card says **Driver needed**, use **Install** or **How to**. GP-SDR uses
   user-space tools; it does not install a macOS kernel extension.
5. For a PortaPack, enter HackRF USB mode and verify that `hackrf_info` reports
   one board. For RTL-SDR, verify that `rtl_test -t` can claim the device.
6. Return to GP-SDR, press **Refresh**, select the receiver, and try the
   Broadcast FM profile with conservative gain and the amplifier off.

If macOS says the app is damaged after downloading, first download it again and
verify its checksum. Do not remove quarantine attributes from an unverified file.

### P25 on macOS

SDRTrunk is already inside the app. Open **Decoders → SDRTrunk** or a P25
profile. Use the JMBE action once if unencrypted P25 voice is silent. JMBE is
created locally because of its license. Control-channel lock, system/NAC data,
and grants are the acceptance signals; RF energy alone is not decoded P25.

### Always-on Mac setup

For a Mac that will keep the receivers connected permanently:

1. Put GP-SDR in Applications and add it under **System Settings → General →
   Login Items**.
2. Enable LAN access in GP-SDR only on a trusted home network. Bookmark the
   tokenized Web UI link shown by the app; do not expose its port directly to
   the public internet.
3. GP-SDR prevents idle system sleep while a receiver, Mapper job, or P25
   session is running. The display may still turn off normally.
4. Connect each dongle directly or through a powered USB hub and assign stable,
   named receiver roles in **Hardware**.
5. Back up the GP-SDR data folder shown in **Settings**. Recordings and IQ files
   can be large, so choose a retention period instead of leaving it unlimited
   unless the storage is monitored.

## Windows 10 or 11

1. Extract the complete ZIP to a normal folder. Keep `GP-SDR.exe`, `sdrtrunk`,
   and `jmbe-creator` together.
2. Run `GP-SDR.exe -open`.
3. Open **Hardware**, connect one receiver, and press **Refresh**.
4. If the receiver cannot be claimed, follow the in-app **How to** guide. HackRF
   and many RTL-SDR units use WinUSB. With Zadig, replace the driver only for
   the exact SDR interface—never an unrelated USB device.
5. Unplug/reconnect the receiver and refresh GP-SDR.

Windows Firewall permission is needed only if you intentionally enable access
from another device. Local use does not require a public-network firewall rule.

For an unattended Windows host, use Task Scheduler to start GP-SDR at boot or
sign-in with an explicit data directory and access token. Do not expose port
8073 directly to the public internet. The complete procedure, private-network
firewall rule, updates, logs, and backup steps are in the [Linux and Windows
server guide](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Server-Setup-Linux-and-Windows).

## Debian and Ubuntu

Choose the package matching the computer:

```bash
sudo apt install ./gp-sdr_1.4.1_amd64.deb
# or, on 64-bit ARM:
sudo apt install ./gp-sdr_1.4.1_arm64.deb
sudo systemctl enable --now gp-sdr
```

Open `http://127.0.0.1:8073/`. The system service binds to localhost by default.
Install the distribution's `hackrf` or `rtl-sdr` package if the Hardware page
reports missing host tools, then confirm your user has the distribution's SDR
udev permissions and reconnect the receiver.

To expose the interface on a trusted LAN, override the service with a listen
address of `0.0.0.0`. Preserve the generated access token and use a VPN or an
authenticated HTTPS reverse proxy outside a trusted LAN.

The packaged service uses a restricted dynamic account and stores data under
`/var/lib/gp-sdr/GP-SDR`. Some distributions grant USB SDR access through the
`plugdev` group; if so, add that group through a systemd override. Exact
commands for the override, token, firewall, USB permissions, logs, updates, and
backups are in the [Linux and Windows server
guide](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Server-Setup-Linux-and-Windows).

## First signal checklist

1. In **Hardware**, confirm the receiver is **Connected**, not merely listed.
2. In **Tuner**, select the correct receiver and start with the amplifier off.
3. Choose a known strong local broadcast FM station, WFM, and the Broadcast FM
   sample rate. Increase LNA/VGA gain gradually.
4. Confirm the signal rises above the noise floor, the signal indicator changes,
   squelch opens, and audio is audible.
5. If every frequency appears active, disable the amplifier, reduce gain, run DC
   correction, and raise squelch. An antenna overload can look like activity
   across the entire display.
6. Save a device calibration only after verifying it against a known reference.

## P25 checklist

1. Import or create a profile with verified control-channel frequencies.
2. Assign a wideband receiver, or separate control and voice receivers.
3. Start the profile and wait for **Control channel locked**.
4. Confirm NAC/WACN/system identifiers and channel grants appear.
5. Create JMBE if unencrypted calls have no audio.
6. Use the unified talkgroup mixer to mute, solo, or change each talkgroup level.

GP-SDR forces the HackRF RF amplifier off when starting its isolated P25
receiver. Excess front-end gain can preserve control-channel lock while making
voice frames sound robotic or intermittent; raise LNA/VGA gain gradually before
enabling an external amplifier.

Encrypted calls can be identified and logged but are not decrypted. A quiet
system, wrong control channel, unsuitable antenna, overload, or weak signal can
all produce no calls even when the software is operating normally.

## Remote and headless receivers

Add an `rtl_tcp` source from **Hardware → Remote receiver**. A remote source has
reduced hardware-control support because gain, bias tee, calibration, and sample
rates depend on the remote server. Do not expose unauthenticated `rtl_tcp`
directly to the internet.

For GP-SDR's mobile web interface, enable LAN listening in Settings or start:

```bash
gp-sdr -listen 0.0.0.0 -port 8073
```

Open the displayed tokenized URL from a phone on the same trusted network.

## Optional components

- **Transcription:** press **Install** on the Transcription card. GP-SDR installs
  `whisper.cpp` and downloads the pinned English base model after you confirm.
  Audio and text remain local unless Mapper upload is enabled.
- **SoapySDR:** install SoapySDR and the module for the exact receiver. The macOS
  app already contains GP-SDR's Universal stream bridge.
- **RadioReference:** use approved API credentials, or download official CSVs
  while signed in and select their folder under **Settings → Local database**.
- **Other decoders:** cards show **Install** or **How to** only when GP-SDR can
  safely automate or explain the platform-specific setup.

The Wiki's [Optional Components
guide](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Optional-Components)
contains a platform matrix, installation paths, executable names, verification
commands, Mapper routing behavior, and troubleshooting for every supported
decoder. Linux and Windows intentionally show **How to** when no safe,
maintained automatic installer exists for that platform.

### Optional decoders on macOS

In the packaged app, open **Hardware** and press **Install** on any missing
DSD-FME, dump1090, multimon-ng, acarsdec, or AIS-catcher card. One confirmed
action installs the complete revision-pinned optional decoder suite, including
rtl_433, into the active Homebrew prefix. Progress and errors remain visible in
GP-SDR. Press **Refresh** afterward.

GP-SDR detects decoder executables in both Apple Silicon and Intel Homebrew
prefixes. The following helper installs the maintained upstream builds of
DSD-FME, dump1090, multimon-ng, acarsdec, and AIS-catcher, plus the dependencies
needed for GP-SDR's file/audio bridges:

```bash
chmod +x Scripts/install_optional_decoders_macos.sh
Scripts/install_optional_decoders_macos.sh
```

The helper builds reviewed, pinned upstream revisions in a temporary directory
and installs into the active Homebrew prefix. It does not start background services or change USB/network
security settings. Return to **Hardware** and press **Refresh** when it finishes.
The source projects and their licenses remain independent and are credited in
`THIRD_PARTY.md`.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| HackRF not listed | Re-enter HackRF USB mode, reconnect the data cable directly, run `hackrf_info`, then Refresh. |
| RTL-SDR busy | Close SDR++, SDRTrunk, `rtl_tcp`, or any other process claiming it, then reconnect. |
| Static but no station | Verify antenna band, WFM mode for broadcast FM, gain, sample rate, squelch, and audio output. |
| Overloaded with antenna removed | Turn amplifier off, reset gains, enable DC removal, restart the receiver, and recalibrate only after the baseline is normal. |
| P25 energy but no lock | Verify the current control channel, system profile, sample rate, antenna, and frequency calibration. |
| P25 lock but no voice | Check encryption status, JMBE creation, talkgroup mute/solo, and audio output. |
| Interface unavailable on phone | Enable LAN binding, use the displayed tokenized URL, allow the private-network firewall rule, and stay on the same LAN/VPN. |

## Verify a download

On macOS or Linux:

```bash
shasum -a 256 -c SHA256SUMS.txt
```

On Windows PowerShell, compare this result with the matching line in
`SHA256SUMS.txt`:

```powershell
Get-FileHash .\GP-SDR-1.4.1-windows-x86_64.zip -Algorithm SHA256
```

## Build from source

Install Go 1.24 or newer. macOS packaging also needs Xcode command-line tools.

```bash
cd server
go test ./...
go build -o gp-sdr .
./gp-sdr -demo -open
```

To create all release packages on macOS:

```bash
chmod +x Scripts/build_release.sh Scripts/fetch_p25_stack.sh
Scripts/build_release.sh 1.4.1
```

Do not build a source checkout from a cloud-synced folder while it is resolving
conflicts. The release script stops if duplicate conflict copies are detected.
All third-party code and licenses are listed in `THIRD_PARTY.md`.
Before publishing a release, follow `Docs/RELEASE_CHECKLIST.md`; updating the
versioned Wiki source and live GitHub Wiki is part of the release gate.
