GP-SDR — WINDOWS 10/11

1. Extract the complete ZIP; keep GP-SDR.exe and gophertrunk.exe together.
2. Double-click GP-SDR.exe, or run:
     GP-SDR.exe -open
3. If the browser does not open, visit http://127.0.0.1:8073/
4. Connect receivers, open Hardware, and press Refresh.

Test the interface without a radio:
     GP-SDR.exe -open -demo

The complete P25 Phase 1/2 engine is included. GP-SDR also includes built-in
AM/NFM/WFM DSP, wideband channel banks, real spectrum/waterfall, recording,
and live audio. P25 and bank channels have independent mute, solo, volume,
activity, and logging controls.

Windows may require the correct WinUSB driver for HackRF or RTL-SDR hardware.
The Hardware page provides an Install or How to action. SoapySDR modules,
Whisper models, vendor-controlled drivers, and optional non-P25 decoders are
not included. THIRD_PARTY.md lists every bundled and integrated project.

To use the mobile interface on a trusted LAN:
     GP-SDR.exe -listen 0.0.0.0 -port 8073

The console prints a URL with a random access token. Keep it private. Use a VPN
or authenticated HTTPS reverse proxy for access outside a trusted network.
