# GP-SDR architecture

GP-SDR uses one local Go service for the native macOS app, headless server, and
mobile/desktop browser console. The macOS app always starts its own bundled
service on an available loopback port, so it does not attach to an unrelated or
stale server process.

```text
HackRF / RTL-SDR / Soapy sources
              │
              ├── built-in IQ path ── AM/NFM/WFM ── audio/logs
              │          │
              │          └── FFT ── spectrum/waterfall/activity
              │
              └── bundled P25 engine ── control + voice taps
                                           │
                                           ├── talkgroup metadata/logs
                                           └── per-talkgroup audio
                              │
                              ▼
                    runtime + event journal
                              │
                    authenticated local API
                              │
                ┌─────────────┴─────────────┐
                ▼                           ▼
         native macOS WebKit app     responsive web client
```

## Receiver ownership

Exactly one capture path owns a physical receiver at a time. The runtime stops
the tuner/survey capture before delegating selected devices to the P25 engine and
stops P25 before returning them. Profiles express desired roles; explicit device
assignments are honored first and compatible connected devices fill remaining
roles.

A single P25 receiver uses a wideband configuration with control, signaling, and
voice taps when the system fits its instantaneous span. Multiple devices can use
independent control and voice roles. GopherTrunk's loopback HTTP API supplies
active calls, talkgroup state, and filtered WAV streams; it is never exposed as
the public GP-SDR API.

## Analog and channel-bank path

The built-in capture adapters normalize HackRF signed 8-bit IQ, RTL-SDR unsigned
8-bit IQ, and Soapy CS8 input. The tuner and survey engines share the FFT,
demodulation, squelch, audio fan-out, recording, and event components. A fitting
fixed channel bank consumes one wide IQ stream while maintaining independent
activity, mute, solo, volume, and log state for every channel.

## Data and trust boundary

Versioned JSON profiles are untrusted on import, so their size, counts,
frequencies, ranges, and required P25 fields are validated. Events use an
append-only JSON-lines journal; WAV audio is stored separately and only paths
inside GP-SDR storage may be served. RadioReference credentials stay in the
process environment and are never placed in a profile or browser storage.

The service binds to `127.0.0.1` by default. A non-loopback bind requires a token
and generates one when omitted. Public-internet deployment should add TLS and
access control through a VPN or reverse proxy.

## Packaging boundary

The server embeds the web interface and uses only Go's standard library. The
macOS shell is AppKit/WebKit. Release bundles include architecture-matched server
and GopherTrunk executables plus license notices. HackRF/RTL analog host tools,
Soapy modules, Whisper models, RadioReference credentials, and optional decoder
programs are separate and discoverable through the app's setup pages.
