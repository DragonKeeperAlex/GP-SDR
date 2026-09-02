# Optional Components

GP-SDR's analog DSP and complete SDRTrunk P25 engine are included in release
packages. Other protocol decoders, hardware abstraction modules, transcription
models, and account-based data sources remain separate because their upstream
licenses, platform packaging, or credentials differ.

Open **Hardware** or **Settings → Integrations** after installation. A card must
show **Ready** before GP-SDR can launch that component. Press **Refresh** after
installing anything outside the app.

## What is included

| Component | Package state | What remains |
| --- | --- | --- |
| AM, NFM, WFM DSP | Included | Connect a supported receiver. |
| SDRTrunk P25 Phase 1/2 | Included in complete packages | Create JMBE once for unencrypted voice. |
| JMBE Creator | Included | Run the in-app creation action and accept its upstream notice. |
| Native HackRF/RTL capture | GP-SDR bridge included | Install user-space tools/USB driver when the OS requires them. |
| SoapySDR bridge | Included where built | Install SoapySDR and the module for the exact radio. |
| DSD-FME and protocol decoders | Integrated, not bundled | Install the matching executable. |
| whisper.cpp transcription | Integrated, model not bundled | Install `whisper-cli` and an approved local model. |
| RadioReference | Integrated, credentials/data not bundled | Add approved credentials or an official local export. |

## Supported decoder executable names

| GP-SDR decoder | Executable GP-SDR searches for | Mapper use |
| --- | --- | --- |
| DSD-FME | `dsd-fme` | DMR, conventional P25, NXDN, D-STAR, YSF, and M17 candidate decoding |
| rtl_433 | `rtl_433` | ISM sensors, weather sensors, and TPMS |
| dump1090 | `dump1090` or `dump1090-fa` | ADS-B and Mode S |
| multimon-ng | `multimon-ng` | POCSAG, FLEX, MDC1200, and DTMF |
| acarsdec | `acarsdec` | ACARS |
| AIS-catcher | `AIS-catcher` or `ais-catcher` | Marine AIS |
| whisper.cpp | `whisper-cli` or `main` | Voice transcription after demodulation/decoding |

Matching a decoder's normal band or modulation is only a candidate. Mapper
marks a protocol identified only after valid decoder output, or another
authoritative verification source, is received.

## macOS automatic setup

With Homebrew installed, press **Install** on a missing receiver,
transcription, or decoder card. The packaged optional-decoder action builds the
revision-pinned DSD-FME, dump1090, multimon-ng, acarsdec, AIS-catcher, and
rtl_433 suite from their maintained upstream sources. GP-SDR shows progress and
errors, then refreshes component status.

From a source checkout, the same reviewed decoder-suite helper is:

```bash
chmod +x Scripts/install_optional_decoders_macos.sh
Scripts/install_optional_decoders_macos.sh
```

GP-SDR searches both `/opt/homebrew` and `/usr/local`, so a native Apple Silicon
installation and an older Intel/Rosetta prefix are both discoverable.

## Debian and Ubuntu

Start with distribution packages where available:

```bash
sudo apt update
sudo apt install hackrf rtl-sdr soapysdr-tools
sudo apt install rtl-433 multimon-ng
```

Package availability varies by Debian/Ubuntu release. Use each project's
maintained build instructions when a package is absent:

- [DSD-FME](https://github.com/lwvmobile/dsd-fme) — digital voice decoder; its
  upstream project is primarily Linux-focused.
- [rtl_433](https://github.com/merbanan/rtl_433) — upstream provides Debian and
  Ubuntu package guidance and Linux release archives.
- [dump1090-fa](https://github.com/flightaware/dump1090) — FlightAware's
  maintained ADS-B/Mode S decoder and Debian build instructions.
- [multimon-ng](https://github.com/EliasOenal/multimon-ng) — pager and signaling
  decoder.
- [acarsdec](https://github.com/f00b4r0/acarsdec) — ACARS decoder.
- [AIS-catcher](https://github.com/jvde-github/AIS-catcher) — AIS decoder with
  RTL-SDR, HackRF, and Soapy-capable builds.

Install custom builds into `/usr/local/bin`, or place them together and set
`GPSDR_HELPERS` in the GP-SDR systemd override. Restart the service and press
**Refresh**.

## Windows 10 or 11

Use native x86_64 executables. Keep every decoder's required DLLs beside its
executable or on `PATH`.

- `rtl_433` publishes native Windows release archives from its [official
  releases](https://github.com/merbanan/rtl_433/releases).
- Install [PothosSDR](https://downloads.myriadrf.org/builds/PothosSDR/) when a
  decoder or receiver needs RTL-SDR/Soapy libraries, and choose the matching
  hardware module.
- AIS-catcher, multimon-ng, acarsdec, dump1090, and DSD-FME should be installed
  only from their maintained upstream instructions or release assets. If no
  maintained native binary exists for the selected version, run that decoder
  with GP-SDR on Linux rather than downloading an unverified third-party build.

Add the decoder directory to `PATH` or set `GPSDR_HELPERS` for the account
running GP-SDR:

```powershell
[Environment]::SetEnvironmentVariable(
  "GPSDR_HELPERS", "C:\GP-SDR\Helpers", "User")
```

Restart the app or scheduled task afterward.

## Offline transcription

GP-SDR requires both a whisper.cpp executable and a GGML model. The macOS
Install action downloads a checksum-pinned English base model. On Linux or
Windows, install [whisper.cpp](https://github.com/ggml-org/whisper.cpp), then set
both environment variables for the GP-SDR process:

```text
GPSDR_WHISPER_EXECUTABLE=/full/path/to/whisper-cli
GPSDR_WHISPER_MODEL=/full/path/to/ggml-base.en.bin
```

On Linux, put these in the systemd override as `Environment=` entries. On
Windows, set them in the scheduled task account's user environment with
`[Environment]::SetEnvironmentVariable(...)`. Restart GP-SDR; the Transcription
card should show **Ready**. Larger models use more memory and can delay Mapper
analysis.

## SoapySDR and additional radios

Install the SoapySDR runtime plus the module for the exact receiver. Confirm the
module outside GP-SDR first:

```bash
SoapySDRUtil --info
SoapySDRUtil --find
```

Native HackRF and RTL-SDR paths do not require SoapySDR. Do not run a native and
Soapy instance of the same physical receiver simultaneously.

## RadioReference and local databases

RadioReference API use requires a Premium account and an approved application
key. GP-SDR also accepts official local CSV exports through **Settings → Local
database**. Do not redistribute subscription-only database downloads. Mapper
uses location filtering before treating a reference match as authoritative.

## Verify readiness

Check the executables from the same account that runs GP-SDR:

```bash
dsd-fme -h
rtl_433 -V
dump1090 --help
multimon-ng -h
acarsdec -h
AIS-catcher -h
whisper-cli --help
```

Some tools return a nonzero status after printing help; the important check is
that the intended executable launches and its required libraries are found.
Then restart GP-SDR, press **Refresh**, and confirm each card says **Ready**.

## Mapper configuration

Create or edit a Mapper job and choose **Auto** or one decoder. Auto uses signal
classification and known target ranges to choose an available backend. A forced
decoder is useful for a known band or imported channel list. Identification can
monitor multiple channels inside one receiver capture, subject to the job's
simultaneous-channel limit and the SDR sample width.

The result details record decoder output, identifiers, transcripts, and the
evidence used. Unsupported, missing, timed-out, encrypted, or invalid frames are
logged without being promoted to a successful identification.

## Common setup problems

| State or error | Fix |
| --- | --- |
| Card remains Setup | Restart GP-SDR, press Refresh, and verify the executable from the same account/service. |
| Executable works in a terminal only | Add its directory to `GPSDR_HELPERS` or the service/task environment. |
| Missing DLL/shared library | Install the decoder's matching runtime dependencies; do not mix architectures. |
| Decoder starts but produces no messages | Verify frequency, mode, bandwidth, signal quality, and supported protocol. |
| Digital voice has no audio | Check encryption, JMBE/DSD-FME readiness, audio routing, and dropped samples. |
| Mapper labels only Candidate | Valid decoder frames or authoritative nearby reference evidence have not been received yet. |

---

1.4.1 baseline source: [integrations.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/integrations.go), [install_optional_decoders_macos.sh](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Scripts/install_optional_decoders_macos.sh).

The rc9 [Local Intelligence](Local-Intelligence) guide adds Ollama model setup. Ollama and whisper.cpp serve different stages: the former correlates evidence; the latter transcribes audio. Neither replaces a missing protocol decoder. Android’s preview has separate [platform limits](Android-Preview).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
