# GP-SDR 1.5.0-rc19

## Built-in macOS updater

- Settings now includes a Software update card with automatic launch checks and an on-demand check.
- Available releases show their version, notes, and official GitHub release link.
- GP-SDR downloads the universal macOS ZIP directly from the official repository.
- The updater requires the package to appear in the release SHA-256 manifest, verifies its digest, validates the extracted bundle identifier, and runs strict recursive code-signature validation.
- Active receiver, Mapper, and deferred-analysis work is stopped before installation.
- The running application bundle is replaced only after verification and GP-SDR restarts automatically.
- A previous application bundle is retained beside the app as a one-version rollback copy until the next update.

GP-SDR's Application Support directory is not part of the app bundle. Profiles, settings, Mapper results, event history, recordings, IQ evidence, calibrations, and local learning data are therefore left in place during an update.
