# GP-SDR 1.1.2

GP-SDR 1.1.2 expands Mapper into a faster multi-frequency survey workspace and
makes long-running jobs easier to understand at a glance.

## Simultaneous Mapper monitoring

- Discovery and Identify can monitor multiple nearby frequencies from one IQ
  capture without opening extra receiver processes or increasing USB bandwidth.
- Added a configurable **Channels at once** limit from 1 through 32.
- Auto defaults to 16 Discovery channels or 4 Identify channels to balance
  speed and CPU use.
- Frequencies are grouped only when their signal and local-noise windows fit
  inside 84% of the selected SDR sample rate; out-of-span frequencies move to
  later capture batches automatically.
- Multi-channel capture placement avoids the direct-conversion DC center when
  possible while retaining saved PPM, gain, amplifier, and I/Q calibration.

## Clearer live Mapper workspace

- Added an operation strip showing the active workflow, receiver, pass, batch,
  capture width, center frequency, and every software VFO being checked.
- Added a latest-capture spectrum and waterfall to the Mapper page.
- Current Mapper channels are highlighted over the spectrum and the display is
  labeled with the job and receiver that match the latest capture.
- Updated pass progress and ETA calculations to operate on simultaneous capture
  batches rather than implying one hardware retune per frequency.

## Results search and organization

- Search now covers numeric and formatted frequencies, names, protocols,
  modulation, callsigns, decoder evidence, automatic analysis, and transcripts.
- Added job, receiver, signal type, identification/activity, and sort controls.
- Results can be ordered by recency, hits, occupancy, confidence, or frequency.
- The complete results area can be collapsed, with the preference retained for
  the next session.

## Native startup recovery

- Extended the bundled receiver-service readiness window.
- **Reload Interface** now performs a fresh authenticated service check.
- Added an in-app Retry button if the receiver service is temporarily late,
  preventing a stale failure page after the service becomes available.

## Verification

- Race-enabled Go test suite and Mapper API acceptance tests
- Desktop and mobile-responsive UI interaction smoke tests
- JavaScript syntax validation
- Universal macOS, Linux amd64/arm64, and Windows amd64 compilation checks

## Packages

- macOS Universal DMG and separate Apple Silicon and Intel ZIP packages
- Debian/Ubuntu amd64 and arm64 packages
- Windows 10/11 x86_64 ZIP package
- SHA-256 checksums for every downloadable artifact

The macOS packages are ad-hoc signed but not Apple-notarized. Windows packages
are not Authenticode-signed. GP-SDR remains receive-only; encrypted P25 traffic
is identified and skipped rather than decrypted.
