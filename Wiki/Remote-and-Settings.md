# Remote Use, Headless Operation, and Settings

## Native app and companion interface

The macOS app is the primary interface. It starts a private service on an
available loopback port and opens the complete UI in its WebKit window. The
companion interface uses the same API and is designed for desktop and mobile
browsers.

## LAN access

GP-SDR binds to `127.0.0.1` by default. To reach it from another device on a
trusted LAN, enable LAN listening in Settings or start:

```bash
gp-sdr -listen 0.0.0.0 -port 8073
```

Use the tokenized link shown by the app. The Web UI address is clickable in the
native app. Keep both devices on the same trusted LAN or VPN.

Do not expose the port directly to the public internet. For remote access, use
a VPN or an authenticated HTTPS reverse proxy that preserves the GP-SDR token.

## Always-on setup

1. Put GP-SDR in Applications or install the Linux service.
2. Add the macOS app to Login Items if desired.
3. Use a powered USB hub and name/assign persistent receiver roles.
4. Enable only trusted-LAN access.
5. Set finite recording and IQ retention.
6. Back up the data directory shown in Settings.

Active receiver, P25, and Mapper sessions prevent idle system sleep. Display
sleep is allowed and the assertion is released when GP-SDR exits.

## Display performance

Settings controls waterfall frame rate, quality/FFT detail, smoothing, peak
hold, and history. Suggested order when reducing lag:

1. Lower waterfall frame rate.
2. Lower FFT/detail quality.
3. Reduce waterfall history.
4. Reduce receiver sample rate only if USB or DSP is overloaded.

The spectrum can remain responsive at a lower display rate because capture and
audio processing are not tied to every visual frame.

## Storage and retention

The Data card separates total GP-SDR use, recordings, IQ evidence, and
journal/profile data. Profiles control capture choices and retention. Forever
is useful for carefully managed servers but is not recommended for unattended
IQ recording without external storage monitoring.

## Integration status

Integration cards show Ready, Optional, Missing, or Setup state for Live DSP,
SoapySDR, P25, transcription, RadioReference, and protocol decoders. Optional
components do not block navigation or basic analog use. Use **Install** where
automated setup is supported or **How to** for vendor/OS steps.

## Security model

- Local-only binding is the default.
- Non-loopback binding requires an access token and generates one when absent.
- RadioReference credentials remain in the process environment, not profiles or
  browser storage.
- Imported profiles are untrusted and validated.
- Only media paths inside GP-SDR storage can be served.
