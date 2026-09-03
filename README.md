# GP-SDR

**General Purpose Software Designed Radio** is an app-first, receive-only SDR
workstation for scanning, tuning, logging, P25 trunk following, wideband channel
mixing, and later review. The native macOS app contains the complete interface
and local receiver service. The same interface can also run headlessly in a web
browser and is designed for phones and tablets.

> GP-SDR 1.5.0-rc11 is the current release candidate. On-air performance still
> depends on the receiver, antenna, local signals, USB link, and gain settings.

## What is included

- Native macOS app for Apple Silicon, Intel, and Universal Macs
- Windows 10/11 package and Debian/Ubuntu packages for amd64 and arm64
- Bundled SDRTrunk v0.6.1 P25 Phase 1/2 trunk-following stack and JMBE Creator
- Native HackRF and RTL-SDR discovery and live IQ input
- Persistent, serialized local RTL-SDR sessions for scanning, Mapper, Tuner,
  and calibration, avoiding repeated USB teardown between short captures
- Concurrent receivers with control, voice, survey, tuner, and channel-bank roles
- Built-in AM, narrow FM, and broadcast FM DSP, squelch, recording, and live audio
- Real tuner spectrum and scrolling waterfall with peak hold, channel markers,
  cursor frequency/power, click-to-tune, recent frequencies, and estimated SNR
- Separate hardware center and software Listen VFO controls; GP-SDR offsets a
  new VFO away from receiver DC by default and can move it inside a locked
  HackRF/RTL-SDR passband without restarting the device
- Unified channel controls plus per-talkgroup mute, solo, identity, and activity
- Unattended activity logging, signal grouping, recordings, and later review
- Indexed Timeline search across transcripts, callsigns, protocols, labels,
  systems, and frequencies; configurable recording/IQ retention
- Independent Mapper jobs per receiver: Map combines discovery, local
  classification, available decoders, and transcription in a repeating range
  pass. Discovery prioritizes collection and defaults to deferred/off-air
  analysis; Identify revisits one channel at a time by default for maximum
  accuracy. Each job has live frequency/progress/ETA, 0.1-second-to-7-day per-channel observation, and
  receiver/job provenance in the combined result table and CSV export. Nearby
  targets share one IQ capture with a configurable 1–1,024 software-VFO limit.
  Auto uses 512 for Discovery, 64 for Map, and one for Identify; usable receiver
  bandwidth and channel spacing can reduce the actual number in each capture
- Mapper shows the current workflow, receiver, batch, every software VFO being
  checked, and the latest receiver spectrum/waterfall; results can be collapsed,
  searched across decoded evidence, filtered to repeated activity, and sorted.
  Identify eligibility can require a chosen hit count, 10–100% successful-check
  rate, recent activity, and a bounded/prioritized channel list
- Automatic Mapper RF tuning starts RTL-SDRs at a conservative manual tuner
  gain and steps through common supported gains from measured headroom; this
  avoids driver-AGC clipping on strong local signals. HackRF uses a bounded
  controller for LNA, VGA, and the roughly 11 dB RF amplifier.
  Current gain, clipping, threshold, and overload decisions remain visible;
  manual and saved-calibration controls are available in Advanced mode
- Mapper measures every software VFO through one shared spectrum pass before
  doing per-hit DSP. Manual analysis queues compact evidence for the
  **Analyze stored captures** action, which runs decoders, transcription, and
  local-model correlation later without opening an SDR. Useful IQ is retained;
  rejected IQ enters a short-lived recoverable quarantine
- DMR and conventional digital voice can be selected directly in Tuner, Live,
  custom channel/range profiles, decoder workspaces, and Mapper. DSD-FME output
  is normalized into protocol, slot, color code, talkgroup, source, encryption,
  and decoded-audio events when those fields are present
- Every Mapper job can use automatic decoder routing or force DSD-FME/DMR,
  rtl_433, dump1090, multimon-ng, acarsdec, or AIS-catcher for its targets
- Parallel receiver and antenna characterization across full nominal, antenna,
  or custom ranges with live frequency, ETA, response/noise plots, overload
  detection, saved-calibration/manual-gain modes, recommendations, and CSV export
- Peak activity hours, expandable identification evidence, optional
  location/transcription, filters, saved job controls, and Sheets upload
- Dedicated pages for Analog, P25, DSD-FME, rtl_433, dump1090, multimon-ng,
  acarsdec, and AIS-catcher
- Normalized live decoder bridges for digital voice, sensors, ADS-B/Mode S,
  paging/signaling, ACARS, and AIS; analog CTCSS detection
- Built-in decoder scan profiles for conventional digital voice, common
  315/345/433.92/868/915 MHz sensors, 1090 MHz ADS-B/Mode S, paging/signaling,
  North American ACARS, and both marine AIS channels
- Built-in GMRS, NOAA Weather Radio, MURS, CB, broadcast FM/AM, civil air,
  marine VHF, 2 m, 70 cm, public-safety discovery, and custom-range profiles
- Bundled San Ramon/East Bay conventional and P25 profiles, all 84 California
  GMRS repeaters from the local archive, and sanitized handheld/travel banks
- Profile export plus bulk CHIRP CSV/TSV import; select multiple programming
  files and turn every file into a complete channel bank in one step
- Automatic Google Sheets range sync with read-only built-in profiles, manual
  refresh, scheduled updates, validation, and an offline cache
- RadioReference ZIP/location import with 5/10/25/50/100-mile and custom
  1–100-mile ranges
- Optional local whisper.cpp transcription
- Optional localhost-only signal intelligence with a lightweight Ollama model,
  conservative confidence gating, user-confirmed examples, and JSONL training export
- Included Universal macOS SoapySDR bridge for other installed SDRs and remote sources
- Authenticated, responsive web console for headless and mobile use

Encrypted P25 calls are identified and skipped. GP-SDR does not transmit and
does not attempt to defeat encryption.

Mapper keeps RF evidence separate from protocol proof. Activity on a known
decoder target is labeled as a candidate and records the matching decoder and
its installation state; a specific protocol is not claimed merely because a
frequency falls inside a known band.

## Install and run

For the complete first-run checklist, receiver-driver steps, P25 setup, LAN
access, and troubleshooting, see the [installation guide](Docs/INSTALL.md).
The [GP-SDR Wiki](https://github.com/DragonKeeperAlex/GP-SDR/wiki) provides a
detailed guide to every main page, receiver workflow, decoder, integration, and
setup option. Start with [Getting
Started](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Getting-Started), use
[Linux and Windows Server
Setup](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Server-Setup-Linux-and-Windows)
for an always-on or headless host, and see [Optional
Components](https://github.com/DragonKeeperAlex/GP-SDR/wiki/Optional-Components)
for platform-specific decoder and transcription setup.
The [1.1 feature status](Docs/FEATURE_STATUS_1.1.0.md) separates live-hardware
evidence from implemented features and external requirements.

### macOS

Download the Universal DMG for the easiest choice, drag **GP-SDR.app** to
Applications, and open it. Separate arm64 and x86_64 ZIP packages are also available. These
preview bundles are ad-hoc signed rather than Apple-notarized, so macOS may
require Control-click → **Open** the first time.

While the native app is open, GP-SDR prevents idle system sleep so unattended
scans and recordings continue. Display sleep remains available, and the sleep
assertion is released when the app quits.

The complete P25 trunking engine is already inside the app. The P25 page can
create the JMBE voice codec locally on first use. For the built-in analog tuner
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
sudo apt install ./gp-sdr_1.5.0-rc11_amd64.deb
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
Google Sheets setup and accepted range columns are documented in
[Docs/GOOGLE_SHEETS_SYNC.md](Docs/GOOGLE_SHEETS_SYNC.md). Mapper CSV exports
and reviewed writes to a master **Additions Queue** are documented in
[Docs/MAPPER_SHEET_WRITE.md](Docs/MAPPER_SHEET_WRITE.md).

GP-SDR chooses the standard user configuration directory. `-data /path`
selects another location.

## P25 setup

The packaged app contains SDRTrunk; no separate trunking app is required. Create
or import a profile containing the system's control channels and optional
talkgroup labels, then assign available receivers. A single wideband receiver
can host the control and fitting traffic channels. Multiple radios increase the
number of simultaneous traffic channels SDRTrunk can follow.

The example [two-receiver P25 profile](examples/p25-two-receiver.gpsdr.json)
contains placeholder values, not a real radio system. RadioReference import can
build location-specific conventional and P25 profiles when authorized account
credentials are available.

## Optional components

The Hardware and Settings pages keep setup inside the app wherever an automatic
installation is safe. Where platform security or vendor licensing prevents it,
the **How to** action gives exact steps instead.

On macOS, a decoder card's **Install** action can run the bundled, revision-pinned
decoder-suite installer for DSD-FME, rtl_433, dump1090, multimon-ng, acarsdec,
and AIS-catcher. Receiver host tools and offline transcription also have in-app
Homebrew actions. Windows driver replacement and Linux system packages continue
to use explicit **How to** steps because those require administrator control.

For source builds on macOS (the release app already includes the stream bridge):

```bash
brew install hackrf librtlsdr soapysdr cmake
Scripts/build_optional_components.sh build/helpers/darwin-arm64
```

For offline transcription, use **Install** on the in-app Transcription card.
GP-SDR installs whisper.cpp and downloads its checksum-pinned English base
model without an API key. Source builds can still override either path with:

```bash
export GPSDR_WHISPER_EXECUTABLE=/path/to/whisper-cli
export GPSDR_WHISPER_MODEL=/path/to/ggml-small.en.bin
```

For local signal intelligence, install or open Ollama, download the default
small model once with `ollama pull qwen2.5:1.5b`, then enable **Settings → Local
intelligence**. GP-SDR sends only bounded DSP/decoder/transcript metadata to
the localhost service. It does not send IQ or audio to a remote server. Use the
checkmark beside a real Activity event to add a verified example; simulated and
unconfirmed detections are never learned automatically. Confirmed examples are
used immediately for retrieval and can be exported as JSONL for later training.

For authorized RadioReference import:

```bash
export GPSDR_RR_USERNAME='your account'
export GPSDR_RR_PASSWORD='your password'
export GPSDR_RR_APP_KEY='approved developer key'
```

Credentials are never included in shared profiles or event logs. A Premium
subscription and developer API access are separate RadioReference requirements.

For an offline library without API access, open a RadioReference county or
state **Downloads** page while signed in and save its official CSV files into a
folder. In GP-SDR, open **Settings → Local database → Choose folder**. GP-SDR
recursively imports `.csv`, `.tsv`, and GP-SDR `.json` files; large statewide
CSVs are split into stable 4,000-channel banks so rescans update rather than
duplicate them. RadioReference provides an official **All Identified
Frequencies in California** CSV on the California Downloads page, but does not
document a single whole-US archive. GP-SDR does not scrape or mirror the
RadioReference database.

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
Xcode command-line tools. Packaging downloads official SDRTrunk v0.6.1 and JMBE
Creator v1.0.9 assets and includes their corresponding source archives as
documented in [THIRD_PARTY.md](THIRD_PARTY.md).

```bash
cd server
go test ./...
go build -o gp-sdr .
./gp-sdr -demo -open
```

Build all release packages on macOS:

```bash
chmod +x Scripts/build_release.sh Scripts/fetch_p25_stack.sh
Scripts/build_release.sh 1.5.0-rc11
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
spectrum generation, SDRTrunk playlist generation, concurrent Mapper jobs,
indexed event search, storage boundaries, optional-decoder handoffs, and the
web application. A live HackRF acceptance capture through GP-SDR decoded EBRCS
NAC `0x1F5`, WACN `0xBEE00`, system `0x1F1`, Phase 1/2 grants, traffic-channel
frequencies, and encrypted-call state with IMBE/AMBE loaded. Live RTL-SDR P25,
other systems, and other packaged operating systems remain separate hardware
acceptance checks; a passing build alone is not presented as RF proof.

See [Architecture](Docs/ARCHITECTURE.md), [release notes](Docs/RELEASE_NOTES_1.5.0-rc11.md),
and [third-party credits](THIRD_PARTY.md).

## Responsible use and license

Receive, record, transcribe, and share only communications you are legally
permitted to handle in your jurisdiction. GP-SDR's original code is released
under the MIT License. Bundled and optional components retain their own licenses;
all credits and redistribution details are recorded in [THIRD_PARTY.md](THIRD_PARTY.md).
