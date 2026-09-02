# Remote Use, Headless Operation, and Settings

## Native app and companion interface

The macOS app is the primary interface. It opens the complete UI in its WebKit window and starts a token-protected
service on all interfaces at an available port. The
companion interface uses the same API and is designed for desktop and mobile
browsers.

## LAN access

The standalone command-line server and packaged Linux service bind to
`127.0.0.1` by default. The native macOS wrapper in 1.4.1 instead starts its
server with `0.0.0.0`, a random token, and a dynamically chosen port. The
Settings Web console card shows an address, not a LAN-enable toggle.

For a standalone server reachable on a trusted LAN, start:

```bash
gp-sdr -listen 0.0.0.0 -port 8073
```

Use the tokenized link shown by the app. The Web UI address is clickable in the
native app. Keep both devices on the same trusted LAN or VPN.

Do not expose the port directly to the public internet. For remote access, use
a VPN or an authenticated HTTPS reverse proxy that preserves the GP-SDR token.

For a complete Debian/Ubuntu systemd deployment or an unattended Windows Task
Scheduler deployment, use [Linux and Windows Server
Setup](Server-Setup-Linux-and-Windows). It includes service overrides, USB
permissions, private-network firewall rules, logs, updates, and backups.

## Always-on setup

1. Put GP-SDR in Applications or install the Linux service.
2. Add the macOS app to Login Items if desired.
3. Use a powered USB hub and name/assign persistent receiver roles.
4. Enable only trusted-LAN access.
5. Set finite recording and IQ retention.
6. Back up the data directory shown in Settings.

The native macOS app prevents idle system sleep while it is open. Display
sleep is allowed and the assertion is released when GP-SDR exits. A standalone
server needs the host’s power settings configured for unattended operation.

## Display performance

Settings controls waterfall frame rate, quality/FFT detail, smoothing, peak
hold, and display floor/ceiling. Suggested order when reducing lag:

1. Lower waterfall frame rate.
2. Lower FFT/detail quality.
3. Reduce receiver sample rate only if USB or DSP is overloaded.

The spectrum can remain responsive at a lower display rate because capture and
audio processing are not tied to every visual frame.

## Storage and retention

The Data card separates total GP-SDR use, recordings, IQ evidence, and
journal/profile data. Set separate caps for Recordings and IQ, choose an age
limit, and optionally enable automatic cleanup. Zero disables a particular
limit. **Clean now** applies the saved policy immediately after confirmation.
Files modified during the last ten minutes are protected so an active capture
is not removed. Cleanup stays inside GP-SDR's Recordings and IQ directories;
profiles, Mapper history, calibration, local channel databases, and range sync
data are never targets. General automatic cleanup is off until explicitly
enabled. Mapper's rejected-IQ cleanup is separate: after local analysis
finishes, quarantined low-value IQ remains recoverable for 24 hours by default and is then
removed. In rc9, the job’s Delete after analysis policy instead removes rejected IQ immediately after finalization, without that recovery timer. The Data and Managed IQ cards show analyzing, retained, and rejected
usage separately; the recovery period can be set from one hour through seven
days.

Raw IQ is high-rate noise-like data and often compresses poorly. Mapper instead
channelizes each detected signal out of the wider receiver capture before
archiving it, then keeps only evidence that survives local analysis. Bounded
retention remains the predictable default for unattended reception.

## Integration status

Integration cards show Ready, Optional, Missing, or Setup state for Live DSP,
SoapySDR, P25, transcription, RadioReference, and protocol decoders. Optional
components do not block navigation or basic analog use. Use **Install** where
automated setup is supported or **How to** for vendor/OS steps.

See [Optional Components](Optional-Components) for the supported executable
names and platform-specific setup and verification steps.

## Security model

- Local-only binding is the standalone server default; the native macOS wrapper enables a token-protected LAN listener.
- Non-loopback binding requires an access token and generates one when absent.
- RadioReference credentials use the process environment or, on macOS, the supported Keychain form; they are not embedded in shared profiles.
- Imported profiles are untrusted and validated.
- Only media paths inside GP-SDR storage can be served.

## Connect a phone or tablet

1. Keep GP-SDR running on the host and both devices on the same trusted LAN/VPN.
2. Use **Settings → Web console → Address** to obtain the current tokenized URL. If it shows a loopback address, substitute the host’s LAN IP while retaining the displayed port and token.
3. Open that URL in the other device’s browser. Do not assume the native app uses port 8073 every time; its token/port may change after restart.
4. Scroll the bottom navigation to reach Tuner, Mapper, and other pages. The same host jobs are controlled by all connected clients.
5. If connection fails, check the listener, host firewall, IP, current port, and token before changing radio settings.

The SDR stays attached to the host. Local database-folder changes and transmit controls are restricted to the host’s local interface. P25’s native system-output audio may play on the host; do not assume every decoder’s live audio is streamed to the phone. Recording playback is a separate workflow.

For strict loopback-only operation, run the standalone server explicitly with `-listen 127.0.0.1`. The built-in server uses HTTP; keep external access behind a VPN or an appropriately configured HTTPS proxy. No app UI toggle changes the native wrapper’s listen arguments in this version.

---

1.4.1 baseline source: [GPSDRApp.swift](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/macos/GPSDRApp.swift), [main.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/main.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
