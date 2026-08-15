# Third-party acknowledgements

GP-SDR's original source is MIT-licensed. The release packages also contain the
P25 component identified below. Every other integration remains a separate,
user-installed program or service and retains its own license and terms.

## Bundled P25 stack

| Component | Use | Version and source | License | Packages |
|---|---|---|---|---|
| [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) by Matt Cheramie and its contributors | P25 Phase 1/2 control-channel decoding, trunk following, talkgroup audio, and HackRF/RTL-SDR source handling | [v0.9.8](https://github.com/MattCheramie/GopherTrunk/releases/tag/v0.9.8), source commit `20f39dc116c1a29a39992bbdbddd2094e4147107` | [Apache License 2.0](https://github.com/MattCheramie/GopherTrunk/blob/v0.9.8/LICENSE) plus its own third-party notices | macOS arm64/x86_64/universal, Linux arm64/amd64, Windows amd64 |

Each binary package includes `GopherTrunk-LICENSE` and
`GopherTrunk-THIRD_PARTY_LICENSES.md`. `Scripts/fetch_p25_stack.sh` downloads the
official release assets and rejects them unless their SHA-256 matches:

| Official asset | SHA-256 |
|---|---|
| `gophertrunk-v0.9.8-darwin-amd64.tar.gz` | `413220e6591ad606ac6f9cee702b367cb4f19cbd7a1d334ea7883f041d390014` |
| `gophertrunk-v0.9.8-darwin-arm64.tar.gz` | `0ea913ab233e86573b260d5a087eae6458e8327727ad341405ef1c152b2f4ec8` |
| `gophertrunk-v0.9.8-linux-amd64.tar.gz` | `e63bc3c7f6e22be02c8cbce36d26c5beedb2456bbc2be00ceda9d870200b4673` |
| `gophertrunk-v0.9.8-linux-arm64.tar.gz` | `bdefc7dc0bdca9134021b7d7639da8fb5cd80b2a537f73a57a7e9431671b68fb` |
| `gophertrunk-v0.9.8-windows-amd64.zip` | `b8e5e0cf7799b76ad4645c49eca417cbc2bf97ca87f10292ab931ad52df04db7` |

GopherTrunk displays an upstream notice about the AMBE+2 voice codec's patent
status. Inclusion of the decoder is not legal advice or permission to monitor,
record, or disclose communications. GP-SDR identifies and skips encrypted
talkgroups; it does not attempt to defeat encryption.

## Separate runtime integrations

| Project or service | How GP-SDR uses it | Upstream license or terms | Bundled |
|---|---|---|---|
| [SoapySDR](https://github.com/pothosware/SoapySDR) | Device discovery and streaming through the original, dynamically loaded `gpsdr-soapy` bridge in this repository | [Boost Software License 1.0](https://github.com/pothosware/SoapySDR/blob/master/LICENSE_1_0.txt) | The original GP-SDR bridge is bundled on macOS; upstream SoapySDR and device modules are not |
| [OP25, boatbod fork](https://github.com/boatbod/op25) | Legacy fallback for P25 when a user explicitly supplies `multi_rx.py` | GPL-3.0-or-later; see upstream headers | No |
| [whisper.cpp](https://github.com/ggml-org/whisper.cpp) | Offline transcription through `whisper-cli` | [MIT](https://github.com/ggml-org/whisper.cpp/blob/master/LICENSE) | No; model weights are also not included |
| [HackRF host software](https://github.com/greatscottgadgets/hackrf) | Analog IQ discovery and capture through `hackrf_info` and `hackrf_transfer` | [GPL-2.0](https://github.com/greatscottgadgets/hackrf/blob/main/COPYING), with component notices upstream | No |
| [rtl-sdr](https://github.com/osmocom/rtl-sdr) | Analog IQ discovery and capture through `rtl_test`, `rtl_eeprom`, and `rtl_sdr` | [GPL-2.0](https://github.com/osmocom/rtl-sdr/blob/master/COPYING), with component notices upstream | No |
| [RadioReference](https://www.radioreference.com/) | Authorized subscriber location/range searches and selected imports | RadioReference account, subscription, API-key, and service terms | No code or database content |

Optional decoder executables detected by GP-SDR remain independent programs:

- [DSD-FME](https://github.com/lwvmobile/dsd-fme) for supported digital voice modes
- [rtl_433](https://github.com/merbanan/rtl_433) for supported ISM sensors
- [dump1090](https://github.com/flightaware/dump1090) for ADS-B/Mode S
- [multimon-ng](https://github.com/EliasOenal/multimon-ng) for pager and signaling protocols
- [acarsdec](https://github.com/TLeconte/acarsdec) for ACARS
- [AIS-catcher](https://github.com/jvde-github/AIS-catcher) for marine AIS

## Original interface work

The GP-SDR adapters, configuration generators, DSP, web/mobile interface, macOS
shell, installer guidance, tests, and packaging scripts in this repository are
original project code. They communicate with third-party executables through
published command-line, streaming, and HTTP interfaces. No OP25, whisper.cpp,
HackRF, rtl-sdr, RadioReference database, or optional-decoder source is copied
into GP-SDR.

Missing or incorrect attribution is a release-blocking bug. Please report it.
