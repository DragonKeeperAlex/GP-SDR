# GP-SDR Wiki

GP-SDR (General Purpose Software Designed Radio) is an app-first, receive-only
SDR workstation. It combines direct tuning, spectrum and waterfall displays,
wide-range discovery, frequency identification, recording, analog audio,
P25 trunk following, optional protocol decoders, and later review in one
interface.

The native macOS app is the primary experience. The same local service also
provides an authenticated companion interface for a browser, phone, tablet, or
headless computer.

## Start here

1. [Install and complete first run](Getting-Started).
2. [Connect and configure a receiver](Hardware-and-Drivers).
3. [Tune a known signal and verify audio](Tuner-and-Live).
4. [Choose or import a scan profile](Profiles-and-Mixer).
5. Use [Mapper](Mapper-Discovery-and-Identify) for unattended discovery.
6. Use [P25 and decoder pages](P25-and-Decoders) for digital systems.
7. Use [Server Setup](Server-Setup-Linux-and-Windows) for an always-on Linux or
   Windows host.
8. Use [Optional Components](Optional-Components) to enable additional protocol
   decoders and offline transcription.

## Main pages

| Page | Purpose |
| --- | --- |
| Live | Current receiver state, RF controls, waterfall, active channels, and the mixer |
| Tuner | Direct tuning, spectrum, waterfall, software VFO, audio, and calibration |
| Activity | Searchable transmission timeline, signal groups, recordings, and transcripts |
| Mapper | Independent long-running Discovery and Identify jobs for one or more SDRs |
| Hardware | Local and remote receivers, ownership, telemetry, and dependency setup |
| Decoders | Status and dedicated views for P25, digital voice, sensors, ADS-B, paging, ACARS, and AIS |
| Profiles | Built-in, imported, and custom channel/range/P25 configurations |
| Settings | Interface mode, display performance, retention, integrations, LAN access, and storage |

## Important operating boundary

GP-SDR does not transmit. It does not decrypt encrypted P25 traffic. Decoder
availability and reception depend on the receiver, antenna, local signals, USB
throughput, gain, frequency calibration, and any required third-party driver or
decoder. A process starting or a spectrum peak is not proof of successful
decoding; use the acceptance indicators described on each decoder page.

## More help

- [Data, imports, Sheets, and RadioReference](Data-and-Integrations)
- [Remote use, headless operation, and settings](Remote-and-Settings)
- [Troubleshooting and current limitations](Troubleshooting-and-Limits)
- [Source, licenses, and credits](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/THIRD_PARTY.md)
