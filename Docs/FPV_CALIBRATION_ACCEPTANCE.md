# FPV and calibrated-RF acceptance runbook

This document defines the evidence required to close VID-01, VID-02, and
CAL-01. It is a test procedure, not a claim that those items are currently
accepted. The active acceptance target is the Raspberry Pi deployment.

## VID-01: analog FPV

### Required fixture

- One known-good analog camera and video transmitter with a documented NTSC or
  PAL output, powered from a stable supply.
- A receiver path that is known to cover the selected frequency. Use a shielded
  or conducted setup where possible; if radiated, document antenna, distance,
  orientation, channel, transmitter power, and the test location.
- A timestamped LED or frame counter visible in the source video. A second
  camera or photodiode may be used to measure end-to-end latency.
- A known-good monitor or capture device connected in parallel to the source
  for the reference picture.

### Procedure

1. Record the camera standard, channel/frequency, transmitter power, receiver
   model/serial, antenna, sample rate, gain, and GP-SDR revision.
2. Run the same source at the reference monitor and through GP-SDR. Test the
   declared standard first, then the other standard as a negative/control test.
3. Record time to first stable synchronization from receiver start, whether
   sync is retained for at least 10 minutes, and every loss/reacquisition.
4. Measure latency using the visible timestamp/LED. Report median and worst
   observed delay over at least 30 samples; do not infer latency from process
   startup time.
5. Compare the reference and GP-SDR images for geometry, interlace/field
   behavior, color, tearing, noise, dropped/duplicated frames, and readable
   detail. Save representative reference and received frames.
6. Repeat at the minimum three documented signal conditions: strong, nominal,
   and near the first repeatable loss of lock. Change only one variable at a
   time when comparing gain, antenna, or receiver.
7. Stop the receiver and confirm the process exits, the frame endpoint stops
   changing, and no Mapper job or retained IQ capture is created.

VID-01 can be marked physically verified only when picture lock, latency,
quality, and the receiver-coverage conditions are recorded for both standards
that GP-SDR claims to support. A decoder process, changing PNG, or source-test
pass alone is insufficient.

## VID-02: digital FPV

Keep this item open until one specific protocol is selected. The acceptance
record must name the protocol, band, modulation/codec, compatible receiver or
capture hardware, and legal test source. “Digital FPV” or generic 5.8 GHz
energy is not a supported-protocol claim. If no protocol is selected, record
the item as deferred rather than adding a generic detector.

## CAL-01: controlled calibrated-RF workflow

### Required fixture

- A traceable or recently verified signal generator covering the test
  frequencies, with a known output level and documented uncertainty.
- A calibrated fixed attenuator/coupler and 50-ohm cables/connectors rated for
  the frequency and level. Use a dummy load for any transmitter path.
- A power meter or spectrum analyzer suitable for checking the delivered level.
- One reference receiver or reference path, and one device under test. Record
  serial numbers, firmware, antenna, cable loss, gain settings, bandwidth,
  sample rate, PPM/IQ settings, temperature, and supply voltage.

### Procedure

1. Verify the generator-to-receiver level at the receiver plane, including
   cable, coupler, and attenuator loss. Start at the highest attenuation.
2. Sweep at least five known input levels around the expected noise floor at
   each target frequency and bandwidth. Capture raw IQ and the application’s
   measured power/quality result for every point.
3. Repeat each point at least three times, including a zero-input/noise-floor
   observation. Keep the receiver gain and DSP settings fixed within a sweep.
4. Derive detection threshold, usable dynamic range, and any correction table
   with uncertainty. Store the fixture, source, instrument, and software
   provenance alongside the results.
5. For antenna comparisons, use the same calibrated source and common feed,
   then change only the antenna and repeat the sweep. Report relative results;
   do not label them antenna gain or range without an appropriate reference
   and field/anechoic method.
6. Do not infer SWR from receiver amplitude. Measure SWR/return loss with a
   suitable VNA or directional measurement setup and retain its calibration
   evidence separately.

CAL-01 can be marked physically verified only when repeatability and
uncertainty are reported and the resulting calibration is shown to improve a
separate held-out check. Ambient characterization remains useful for noise
baselines, but it is not absolute sensitivity, antenna gain, SWR, range, or
absolute receiver performance.

## Evidence record

For each run, retain a small manifest containing date/time, operator, host,
GP-SDR revision, hardware serials, fixture diagram, settings, raw-file hashes,
reference media, result summary, and unresolved limitations. Never overwrite
existing profiles, recordings, findings, or calibration files while preparing
the fixture; use temporary test profiles and remove only those temporary
artifacts after the evidence is archived.
