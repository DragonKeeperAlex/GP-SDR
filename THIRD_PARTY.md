# Third-party acknowledgements

GP-SDR's original source is MIT-licensed. Complete binary packages aggregate
the separate GPL programs listed below. Those programs retain their own
copyright, license, source-code, patent, and usage terms.

## Bundled P25 components

| Component | Use | Version | License | Bundled material |
|---|---|---|---|---|
| [SDRTrunk](https://github.com/DSheirer/sdrtrunk) by Dennis Sheirer and contributors | P25 Phase 1/2 control decoding, trunk following, HackRF/RTL-SDR input, call events, and native audio playback | [v0.6.1](https://github.com/DSheirer/sdrtrunk/releases/tag/v0.6.1) | GPL-3.0 | Unmodified platform distribution, `SDRTrunk-LICENSE`, and complete `SDRTrunk-v0.6.1-source.tar.gz` |
| [JMBE Creator](https://github.com/DSheirer/jmbe) by Dennis Sheirer and contributors | Creates the optional IMBE/AMBE P25 voice library locally after the user starts setup | [v1.0.9](https://github.com/DSheirer/jmbe/releases/tag/v1.0.9) | GPL-3.0 with the upstream patent notice | Unmodified platform creator, `JMBE-LICENSE`, and complete `JMBE-v1.0.9-source.tar.gz`; a compiled codec is not redistributed |

`Scripts/fetch_p25_stack.sh` downloads the unmodified official release assets
and corresponding source archives. `COMPONENT_SHA256SUMS.txt` records every
downloaded binary archive. GP-SDR launches SDRTrunk as a separate process in
its supported headless mode and communicates through generated playlists and
SDRTrunk's event logs. GP-SDR does not incorporate or relink SDRTrunk code.

The JMBE project warns that compiled MBE codec objects may be covered by
patents. GP-SDR therefore includes the upstream creator rather than a compiled
codec. The in-app setup action runs that creator only when the user requests
it. This project does not provide legal advice. GP-SDR does not defeat radio
encryption and marks or excludes encrypted calls.

## Separate runtime integrations

| Project or service | How GP-SDR uses it | Upstream license or terms | Bundled |
|---|---|---|---|
| [SoapySDR](https://github.com/pothosware/SoapySDR) | Device discovery and streaming through the original, dynamically loaded `gpsdr-soapy` bridge | Boost Software License 1.0 | The GP-SDR bridge is bundled on macOS; upstream SoapySDR/modules are not |
| [OP25, boatbod fork](https://github.com/boatbod/op25) | Developer/source compatibility code; it is not the packaged or selected P25 backend | GPL-3.0-or-later | No |
| [whisper.cpp](https://github.com/ggml-org/whisper.cpp) | Offline transcription through `whisper-cli` | MIT | No; the in-app installer can install the executable and download a checksum-pinned English base model on request |
| [HackRF host software](https://github.com/greatscottgadgets/hackrf) | Analog IQ discovery and capture through `hackrf_info` and `hackrf_transfer` | GPL-2.0 | No |
| [rtl-sdr](https://github.com/osmocom/rtl-sdr) | Analog IQ discovery and capture through `rtl_test`, `rtl_eeprom`, and `rtl_sdr` | GPL-2.0 | No |
| [RadioReference](https://www.radioreference.com/) | Authorized subscriber location/range searches and selected imports | RadioReference account, subscription, API-key, and service terms | No code or database content |

Optional independent decoders detected by GP-SDR include DSD-FME, rtl_433,
dump1090, multimon-ng, acarsdec, and AIS-catcher. Each remains under its
upstream license and must be installed separately unless a package explicitly
states otherwise.

## Original GP-SDR work

The GP-SDR process adapter, isolated playlist generator, event normalization,
DSP, Mapper, web/mobile interface, native macOS shell, setup workflow, tests,
and packaging scripts are original project code. Missing or incorrect
attribution is a release-blocking bug; please report it.
