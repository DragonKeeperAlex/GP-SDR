# GP-SDR 1.3.0

This major update focuses on a simpler, more utilitarian SDR++-style workflow,
responsive long-running Mapper surveys, and receiver-aware validation.

## Highlights

- Mapper results stay responsive during large surveys: the table renders the
  first 250 sorted/filtered rows while retaining the complete dataset for
  search, export, uploads, and retention policies.
- Mapper now keeps the current workflow, receiver, software-VFO batch, ETA,
  spectrum, waterfall, gain, amplifier, clipping, and overload state visible
  while a job runs.
- Discovery and Identify remain separate one-click workflows, with configurable
  dwell time (0.1 seconds through 7 days), concurrent channels, hit/occupancy
  thresholds, repeated-only filtering, and expandable evidence.
- Live and Tuner retain immediate gain, HackRF amplifier, IQ/DC correction,
  squelch, hardware-center, software-VFO, demodulation, and decoder controls.
- P25 Phase 1/2 trunk following, DMR, DSD-FME, rtl_433, dump1090, multimon-ng,
  ACARS, and AIS bridges are included in the packaged application.
- Storage retention and IQ quarantine controls prevent unattended surveys from
  consuming unbounded disk space.

## Validation

- `go test ./...`, race tests, `go vet ./...`, web syntax checks, and decoder
  bridge smoke tests pass locally.
- Two attached HackRF One receivers were enumerated and exercised with real
  receive captures and FM demodulation. SDRTrunk stayed healthy during a
  bounded P25 control-channel test; no local P25 lock was observed in that
  window, so the release does not claim a decoded P25 frame without RF evidence.
- RTL-SDR hardware validation remains pending when a supported RTL-SDR is
  visible to the host; the application keeps conservative RTL sample-rate and
  gain defaults.

See [INSTALL.md](INSTALL.md) and the [GP-SDR Wiki](https://github.com/DragonKeeperAlex/GP-SDR/wiki)
for setup, hardware troubleshooting, and headless server use.
