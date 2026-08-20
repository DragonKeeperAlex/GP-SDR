# Third-party acknowledgements

GP-SDR's original source is MIT-licensed. The release packages also contain the
P25 component identified below. Every other integration remains a separate,
user-installed program or service and retains its own license and terms.

## Bundled P25 stack

| Component | Use | Version and source | License | Packages |
|---|---|---|---|---|
| [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) by Matt Cheramie and its contributors | P25 Phase 1/2 control-channel decoding, trunk following, talkgroup audio, and HackRF/RTL-SDR source handling | [v0.9.9](https://github.com/MattCheramie/GopherTrunk/releases/tag/v0.9.9), pinned source commit `09b00149d289d9db2867a6d166bc4d71ea912503` with the documented GP-SDR compatibility patch | [Apache License 2.0](https://github.com/MattCheramie/GopherTrunk/blob/v0.9.9/LICENSE) plus its own third-party notices | macOS arm64/x86_64/universal, Linux arm64/amd64, Windows amd64 |

Each binary package includes `GopherTrunk-LICENSE` and
`GopherTrunk-THIRD_PARTY_LICENSES.md`. `Scripts/fetch_p25_stack.sh` clones the
exact pinned upstream commit, verifies it, applies
`third_party/patches/gophertrunk-opensourcesdrlab-offset-binary.patch`, runs the
upstream HackRF tests, and cross-compiles each bundled platform. The small
GP-SDR patch detects the offset-binary IQ stream produced by the tested
OpenSourceSDRLab HackRF One/PortaPack H2 reproduction; standard signed HackRF
streams remain unchanged. The patch source and test ship in this repository.

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
