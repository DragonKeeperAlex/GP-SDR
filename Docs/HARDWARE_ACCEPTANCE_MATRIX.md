# GP-SDR hardware acceptance matrix

Copy one row per bounded test. Preserve logs and evidence paths; do not mark a
decoder accepted from process startup or RF energy alone.

| Date | Host/OS | GP-SDR version | Receiver/serial | USB/network path | Antenna | Workflow | Frequency/system | Rate/bandwidth | Gain/AGC | Duration | USB drops | IQ drops | CPU/RAM/temp | Decoder evidence | Audio/video result | Evidence path | Pass/fail |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-09-15 | Pi/Linux ARM64 | 1.5.0-rc32 | RTL-SDR/E4000 `00000001` | USB hub port `1-1.1` | — | bounded IQ capture | 98.1 MHz WFM | 2.4 MS/s | 19 dB | 10 s | Device later failed enumeration with `-110/-62` | Capture itself completed exactly | Pi not throttled | Not a decoder test | Not routed | kernel journal and rc32 notes | Fail stability |
| 2026-09-15 | Pi/Linux ARM64 | 1.5.0-rc32 | HackRF ending `c5cb` | USB | — | bounded IQ capture | 98.1 MHz WFM | 10 MS/s | configured by test | 2 s | 0 observed | exact 40 MB | — | Not a decoder test | Not routed | rc32 notes | Pass capture |
| 2026-09-15 | Pi/Linux ARM64 | 1.5.0-rc32 | Pluto `ARTAXZBXHAFTNQYI` | Soapy USB | — | bounded IQ capture | 98.1 MHz WFM | 4 MS/s | 30 dB | 10 s | 0 observed | sustained until bounded timeout | — | Not a decoder test | Not routed | rc32 notes | Pass capture |
| 2026-09-16 | Pi/Linux ARM64 | 1.5.0-rc32 | HackRF ending `c5cb` | native SDRTrunk | — | P25 Phase 2 | EBRCS ALCO East / 774.45625 MHz | 10 MS/s | profile | live session | 0 observed | — | — | Locked; NAC `501` / `0x1F5`, system `497` / `0x1F1`; grants and TDULC/FACCH/SACCH frames observed for multiple talkgroups | Digital frames accepted; intelligible audio remains pending JMBE and an unencrypted call | runtime P25 event/log files | Pass RF/trunk; audio pending |
| 2026-09-15 | Pi/Linux ARM64 | 1.5.0-rc32 | Pluto `ARTAXZBXHAFTNQYI` | OP25/Soapy USB | — | P25 | EBRCS ALCO East | 1 MS/s OP25 source | `LNA:36` | 12 s | 0 observed | — | — | Real device opened; control channels rotated; no control identity/grant | No accepted call | runtime OP25 log | Incomplete RF acceptance |

## Minimum matrices before a stable release

### Receiver capture

- Every supported receiver on every packaged OS where hardware is available.
- Direct port and intended hub; reconnect after stop; 30 minutes minimum plus an
  eight-hour combined-radio soak.
- Known WFM, NFM, and AM signals; manual and automatic gain; calibration and DC
  correction; supported low/default/high rates.

### P25

- Receiver × engine × Phase 1/Phase 2 × single/multi-radio assignment.
- Searching, lock, NAC/WACN/SysID, grant, traffic tune, unencrypted audio,
  encrypted exclusion, control-channel change, reconnect, and recording.

### Mapper and analysis

- Discovery, Identify, Map, scheduled phases, capture-now/analyze-later, parallel
  channel counts, all-connected partitioning, caps, quarantine, and cleanup.
- Known positive and negative samples for every decoder; successful identity
  requires valid frames or an authoritative geographically applicable match.

### User interfaces

- Native app and responsive web UI; every visible button and field; narrow and
  wide window sizes; keyboard/focus; disconnected/busy receiver; server restart;
  stale browser reconnect; phone/tablet LAN access.
