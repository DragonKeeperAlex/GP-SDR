# GP-SDR

**General Purpose Software Designed Radio** is an app-first, receive-only SDR
workstation for scanning, tuning, logging, P25 trunk following, wideband channel
mixing, and later review. The native macOS app contains the complete interface
and local receiver service. The same interface can also run headlessly in a web
browser and is designed for phones and tablets.

> GP-SDR 1.0 is a public release. On-air performance still
> depends on the receiver, antenna, local signals, USB link, and gain settings.

## What is included

- Native macOS app for Apple Silicon, Intel, and Universal Macs
- Windows 10/11 package and Debian/Ubuntu packages for amd64 and arm64
- Bundled GopherTrunk v0.9.8 P25 Phase 1/2 trunk-following stack
- Native HackRF and RTL-SDR discovery and live IQ input
- Concurrent receivers with control, voice, survey, tuner, and channel-bank roles
- Built-in AM, narrow FM, and broadcast FM DSP, squelch, recording, and live audio
- Real tuner spectrum and scrolling waterfall
- Unified per-channel/per-talkgroup mute, solo, volume, and activity controls
- Unattended activity logging, signal grouping, recordings, and later review
- Dedicated pages for Analog, P25, DSD-FME, rtl_433, dump1090, multimon-ng,
  acarsdec, and AIS-catcher
- Built-in GMRS, NOAA Weather Radio, MURS, CB, broadcast FM/AM, civil air,
  marine VHF, 2 m, 70 cm, public-safety discovery, and custom-range profiles
- Bundled San Ramon/East Bay conventional and P25 profiles, all 84 California
  GMRS repeaters from the local archive, and sanitized handheld/travel banks
- Profile export plus bulk CHIRP CSV/TSV import; select multiple programming
  files and turn every file into a complete channel bank in one step
- RadioReference ZIP/location import with 5/10/25/50/100-mile and custom
  1–100-mile ranges
- Optional local whisper.cpp transcription
- Included Universal macOS SoapySDR bridge for other installed SDRs and remote sources
- Authenticated, responsive web console for headless and mobile use

Encrypted P25 calls are identified and skipped. GP-SDR does not transmit and
does not attempt to defeat encryption.

## Install and run

### macOS

Download the Universal package for the easiest choice, unzip it, and open
**GP-SDR.app**. Separate arm64 and x86_64 packages are also available. These
preview bundles are ad-hoc signed rather than Apple-notarized, so macOS may
require Control-click → **Open** the first time.

The complete P25 engine is already inside the app. For the built-in analog tuner
and scanner, open **Hardware** and use each component's **Install** or **How to**
button. On macOS, HackRF and RTL-SDR use user-space host tools rather than a
kernel extension.

### Windows 10/11

Extract the ZIP and run `GP-SDR.exe -open`. The included P25 engine can open
HackRF and RTL-SDR devices. Windows may still need the correct WinUSB device
driver; the Hardware page explains the device-specific setup. Other vendor
drivers must come from their vendor.

### Debian or Ubuntu

```bash
sudo apt install ./gp-sdr_1.0.0_amd64.deb
sudo systemctl enable --now gp-sdr
```

Use the `arm64` package on 64-bit ARM. Open `http://127.0.0.1:8073/` locally.

### Test without a radio

```bash
gp-sdr -demo -open
```

Demo activity is clearly labeled and never presented as received RF.

## App workflow

1. Connect one or more SDRs.
2. Open **Hardware**, press **Refresh**, and follow any shown setup action.
3. Use **Tuner** for direct listening and the real spectrum/waterfall.
4. Choose a built-in profile or create/import one under **Profiles**.
5. Assign receiver roles. For P25, use one wideband radio or separate control and
   voice radios. For GMRS or similar banks, use the channel-bank role.
6. Press **Start** on **Live**.
7. Control every active channel or P25 talkgroup from the mixer; inspect saved
   transmissions under **Activity**.

Bundled regional data, public sources, local-file sanitization, and accepted
bulk-import columns are documented in [Docs/CHANNEL_DATA.md](Docs/CHANNEL_DATA.md).

GP-SDR chooses the standard user configuration directory and migrates legacy
Signal Harbor data automatically when found. `-data /path` selects another
location.

## P25 setup

The packaged app contains GopherTrunk; no OP25 installation is required. Create
or import a profile containing the system's control channels and optional
talkgroup labels, then assign available receivers. A single wideband-capable
receiver can host the control channel and several voice/signaling taps when the
system span fits its usable bandwidth. Multiple radios can be assigned
independently when it does not.

The example [two-receiver P25 profile](examples/p25-two-receiver.gpsdr.json)
contains placeholder values, not a real radio system. RadioReference import can
build location-specific conventional and P25 profiles when authorized account
credentials are available.

## Optional components

The Hardware and Settings pages keep setup inside the app wherever an automatic
installation is safe. Where platform security or vendor licensing prevents it,
the **How to** action gives exact steps instead.

For source builds on macOS (the release app already includes the stream bridge):

```bash
brew install hackrf librtlsdr soapysdr cmake
Scripts/build_optional_components.sh build/helpers/darwin-arm64
```

For offline transcription, install whisper.cpp and set:

```bash
export GPSDR_WHISPER_EXECUTABLE=/path/to/whisper-cli
export GPSDR_WHISPER_MODEL=/path/to/ggml-small.en.bin
```

For authorized RadioReference import:

```bash
export GPSDR_RR_USERNAME='your account'
export GPSDR_RR_PASSWORD='your password'
export GPSDR_RR_APP_KEY='approved developer key'
```

Credentials are never included in shared profiles or event logs. A Premium
subscription and developer API access are separate RadioReference requirements.

## Headless and mobile use

Local-only mode is the default:

```bash
gp-sdr -listen 127.0.0.1 -port 8073
```

To use GP-SDR from a phone on a trusted LAN:

```bash
gp-sdr -listen 0.0.0.0 -port 8073
```

The service prints a URL with a random access token when listening beyond
localhost. Keep that token private. For use outside a trusted LAN, place GP-SDR
behind a VPN or authenticated HTTPS reverse proxy; the built-in service is HTTP.

## Build from source

Requirements: Go 1.24 or newer. macOS desktop packaging additionally needs the
Xcode command-line tools. Packaging downloads only the exact, checksum-pinned
GopherTrunk v0.9.8 release binaries documented in [THIRD_PARTY.md](THIRD_PARTY.md).

```bash
cd server
go test ./...
go build -o gp-sdr .
./gp-sdr -demo -open
```

Build all release packages on macOS:

```bash
chmod +x Scripts/build_release.sh Scripts/fetch_p25_stack.sh
Scripts/build_release.sh 0.3.0-preview.1
```

Outputs are written to `dist/` with `SHA256SUMS.txt`. The script creates macOS
arm64, x86_64, and Universal apps; Linux amd64/arm64 DEBs; and a Windows amd64
ZIP. Public notarization requires an Apple Developer ID and notarization
credentials, which are intentionally not stored in this repository.

## Command-line options

```text
-listen ADDRESS   Listen address (default 127.0.0.1)
-port PORT        Interface port (default 8073)
-data PATH        Profiles, recordings, and event data directory
-token TOKEN      Access token; generated automatically for LAN binds
-demo             Generate clearly marked simulated activity
-open             Open the interface in the default browser
```

## Verification scope

Automated tests cover profile validation, API authorization, tuner input,
spectrum generation, GopherTrunk configuration, and the web application. The
macOS hardware checks used live HackRF and RTL-SDR input for the tuner and
verified that the bundled P25 process opens a HackRF, exposes its local API,
creates wideband voice/signaling taps, and applies talkgroup mixer changes.
Actual P25 control-channel lock and decoded voice require an active local system
and remain location-dependent validation, not a simulated pass.

See [Architecture](Docs/ARCHITECTURE.md), [release notes](Docs/RELEASE_NOTES_0.3.0-preview.1.md),
and [third-party credits](THIRD_PARTY.md).

## Responsible use and license

Receive, record, transcribe, and share only communications you are legally
permitted to handle in your jurisdiction. GP-SDR's original code is released
under the MIT License. Bundled and optional components retain their own licenses;
all credits and redistribution details are recorded in [THIRD_PARTY.md](THIRD_PARTY.md).
