# Signal Harbor 0.1.0-preview.1

This first developer preview establishes the distributable application foundation.

## Included

- Native Apple Silicon, Intel, and Universal macOS applications
- Native Windows 10/11 x86-64 server and web console
- Debian packages for x86-64 and ARM64
- Custom scan ranges and fixed channel profiles
- Import, export, duplication, and single-channel sharing
- Multiple concurrent SDR role assignments
- Persistent event journal and grouped signal activity
- Per-channel mixer controls
- Hardware discovery plus raw-IQ process adapters for HackRF and RTL-SDR
- Installed SoapySDR device discovery
- Optional decoder discovery including OP25 and DSD-FME
- Authenticated headless/LAN server mode
- Clearly identified demo receiver for interface testing

## Important preview limitation

Live IQ capture, audio demodulation, digital voice decoding, P25 trunk following, transcription, and RadioReference integration are not complete in this preview. Hardware and decoder tools are detected and modeled, but a detected tool is not presented as a verified live decode path. See the roadmap for the acceptance checks planned for each integration.

## Distribution notes

The macOS applications are ad-hoc signed, not Developer-ID signed or notarized. Windows artifacts are also unsigned. SHA-256 checksums are included with all release files.
