# GP-SDR 1.4.0

## Receiver selection and P25

- Corrects the overly broad HackRF self-test blacklist: a successfully enumerated
  receiver remains selectable, with a visible diagnostic warning. Actual RF
  transmission remains blocked while that warning is present.
- Matches hardware serials to current USB bus/port identities on macOS instead
  of selecting stale SDRTrunk configurations by order. The isolated P25 runtime
  disables other identified tuners so they remain available for other jobs.
- Preserves saved HackRF amplifier/gain settings; P25 offers explicit amplifier,
  LNA and VGA overrides. Receiver-specific sample-rate options retain RTL limits.
- Includes the persistent RTL-SDR session improvements from 1.3.4.
- Uses non-claiming USB enumeration for RTL discovery on macOS rather than
  running the tuner benchmark every time a receiver menu is refreshed.

## Compact receiver console

- Shared native/mobile interface uses flatter panels, compact typography and
  controls, restrained colors, and a narrower navigation bar.
- Mapper separates job setup from receiver monitoring, with visible start/save
  controls, common-band presets, receiver refresh, and receiver range/warning
  information. Progress keeps updating while settings have keyboard focus.
- P25 has direct profile/receiver selection, start/stop controls, talkgroup
  search, and active-only filtering alongside the existing mixer.
- Fixes the Mapper workflow radio-button hit area.
- Fixes mobile navigation direction so every workspace remains reachable.
- Corrects transmit dry-run IQ to signed samples, consumes the whole WAV rather
  than repeating its first second, and removes an unintended AM carrier offset.
  These waveform corrections are covered by tests, not on-air TX acceptance.

## Verification and limits

Physical tests on the attached HackRF captured non-stuck IQ at 5, 10, and
20 MS/s on 100.1, 462.6, and 772.76875 MHz. Strong FM input clipped at the
chosen test gain; lower gain remains important. A fresh isolated P25 run
decoded EBRCS control frames and produced voice recordings; a local speech
model extracted speech from a recorded call without playing test audio.
This is not a subjective guarantee of audio quality or a full-band calibration.

The attached E4000 RTL-SDR also passed 2.4 MS/s capture and P25 lock checks,
but subsequently disappeared from USB. Long-duration RTL reliability and
simultaneous two-radio acceptance remain unresolved pending reconnection.

Automated unit/race checks are isolated from physical radio discovery; real
hardware tests require explicit opt-in and fail if no control lock is observed.
Not every optional protocol had an available on-air signal in this test window.
No RF transmission was performed. Microphone/digital-voice transmission, DCS,
and a trained RF-classification model are not added by this release.
