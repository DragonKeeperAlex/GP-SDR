# GP-SDR 1.5.0-rc9

Release candidate with a dedicated stored-capture Analyze workspace, configurable parallel frequency groups, and a fix for live analysis status polling. The Analyze page refreshes progress, active stages, logs, and ETA every five seconds while visible.

Includes the accumulated Mapping navigation, Band Monitor, local evidence analysis, retention controls, and native window improvements since 1.4.1. Analysis groups queued event records by frequency and approximate receive area. The local model currently serializes inference to limit memory use; decoder and other worker stages can run concurrently.

Validation: Go tests and race checks, JavaScript syntax checks, universal macOS packaging, and installed native UI inspection. These checks do not establish live RF decoding performance for every protocol. Android remains experimental; this release's downloadable packages target macOS, Windows, and Linux.

Your profiles and recordings remain outside the application bundle. Stop analysis before replacing the app, then use Analyze to continue pending work. This is a release candidate, not a claim that all requested features or all capture-recovery cases are complete.
