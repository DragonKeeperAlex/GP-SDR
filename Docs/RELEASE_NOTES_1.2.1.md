# GP-SDR 1.2.1

GP-SDR 1.2.1 adds a unified adaptive Mapper workflow, truthful automatic RF
gain control, and a managed IQ evidence lifecycle for unattended surveys.

## Adaptive Mapper

- **Map** is the new beginner/default workflow. It repeatedly sweeps the chosen
  range and combines activity detection, local modulation classification,
  available protocol decoders, and enabled offline transcription.
- Observation time is adjustable from 0.1 seconds through seven days per
  channel. Long observations use bounded capture windows so Stop remains
  responsive and memory use stays finite.
- Discovery and Identify remain separate options in Advanced mode and existing
  saved jobs remain compatible.
- Mapper cards and the live receiver strip show the actual gain stages,
  amplifier state, signal-above-noise threshold, and latest automatic decision.

## Automatic receiver tuning

- RTL-SDR Auto uses the tuner's driver AGC.
- HackRF Auto now controls LNA and VGA gain between captures and can enable the
  approximately 11 dB RF amplifier after repeated weak-input measurements.
- Clipping or excess headroom immediately disables the HackRF RF amplifier
  before reducing the other gain stages.
- Auto, saved-calibration, and manual gain modes are available, along with
  Auto, weak-signal, balanced, conservative, and manual sensitivity targets.

## Managed IQ evidence

- Mapper shifts an active channel to baseband and resamples it before archival,
  avoiding full-rate 8–20 MS/s files for narrowband evidence.
- IQ remains in **Analyzing** until local classification, available decoding,
  and enabled transcription finish.
- Evidence with decoder output, transcription/callsigns, a supported protocol,
  confident waveform classification, or strong signal evidence is retained.
- Low-value evidence enters a recoverable rejected-IQ quarantine. It is removed
  automatically after 24 hours by default; Settings offers one hour through
  seven days.
- Storage caps discard quarantine before retained evidence and protect pending
  analysis for at least 24 hours. Settings reports analyzing, retained, and
  rejected IQ separately.

## Verification

- Go unit, API, static-analysis, adaptive gain, IQ lifecycle, quarantine, and
  multi-receiver Mapper tests
- JavaScript syntax and simulated adaptive-Mapper API smoke tests
- Universal Intel/Apple Silicon macOS package, Linux amd64/arm64 packages, and
  Windows x86_64 package with checksum and archive verification

The macOS release is ad-hoc signed and not Apple-notarized. GP-SDR remains
receive-only; encrypted P25 traffic is identified and skipped, not decrypted.
