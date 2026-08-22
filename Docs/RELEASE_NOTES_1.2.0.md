# GP-SDR 1.2.0

GP-SDR 1.2.0 turns Mapper results into an auditable identification workflow,
adds bounded storage for unattended capture, and expands receiver comparison
and calibration tools.

## Verified Mapper identities

- The Mapper header now counts successfully identified frequencies separately
  from raw hits and classifier guesses.
- A successful identity requires valid decoder output or an exact match from a
  nearby RadioReference import. Band-plan guesses and decoder candidates do not
  pass verification.
- RadioReference profiles retain their import center and radius. Matches beyond
  the observation area are rejected so an identical frequency in another state
  is not accepted as the local service.
- Google Sheet uploads have an **Identified only** option, enabled by default.
- CSV exports include verification status, reason, and reference distance.

## Faster, more selective surveys

- Discovery and Identify monitor multiple software VFOs in one receiver capture
  with a user-selectable 1–32 channel limit.
- Identify eligibility can require Discovery-only or combined hits, repeated
  observations, 10–100% successful checks, recent activity, and a maximum list.
- Combined Results adds verified, repeated-only, receiver, job, type, status,
  search, and expanded sorting controls.
- Live operation, current channel batches, receiver capture width, pass ETA,
  spectrum, waterfall, and expandable evidence remain visible during jobs.

## Receiver characterization

- Compare multiple connected receivers in parallel across nominal device,
  antenna, or custom frequency ranges.
- Configure sample rate, dwell, points, gains, amplifier state, and saved
  calibration for each ambient response test.
- Results include response/noise plots, current frequency, ETA, tested coverage,
  and CSV export. True calibrated sensitivity still requires a known RF source.

## Bounded capture storage

- Settings now shows separate Recordings and IQ usage.
- Separate size caps and age retention can be saved, with optional automatic
  cleanup and a confirmed **Clean now** action.
- Cleanup removes oldest GP-SDR capture files only, protects files written in
  the last ten minutes, and never targets profiles, Mapper history, calibration,
  range sync, or imported channel data.
- Automatic cleanup defaults off for existing and new installations.

## Verification

- Go unit, API acceptance, race, and static-analysis test suites
- JavaScript syntax and native/web UI regression checks
- Universal macOS, Linux amd64/arm64, and Windows amd64 compilation checks
- Connected HackRF and RTL-SDR discovery/capture checks where RF conditions allow

## Packages

- macOS Universal DMG and separate Apple Silicon and Intel ZIP packages
- Debian/Ubuntu amd64 and arm64 packages
- Windows 10/11 x86_64 ZIP package
- SHA-256 checksums for every downloadable artifact

The macOS packages are ad-hoc signed but not Apple-notarized. Windows packages
are not Authenticode-signed. GP-SDR remains receive-only; encrypted P25 traffic
is identified and skipped rather than decrypted.
