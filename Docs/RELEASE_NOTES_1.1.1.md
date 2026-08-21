# GP-SDR 1.1.1

GP-SDR 1.1.1 is a Mapper usability, reliability, and Google Sheets
compatibility update for the native app and companion web interface.

## Mapper workflow controls

- Replaced the workflow dropdown with compact **Discovery** and **Identify**
  buttons so the active operating mode is immediately visible.
- Discovery continuously sweeps its configured range until stopped and exposes
  the active step size and dwell time.
- Identify revisits discovered or imported frequencies and exposes its extended
  listening duration and transcription setting.
- Renamed the user-facing Decipher workflow to Identify while retaining
  compatibility with existing saved jobs and exported configurations.

## Native app reliability

- Replaced browser-native confirmation prompts with GP-SDR dialogs for deleting
  Mapper jobs and clearing Mapper results.
- Cancel reliably preserves the selected job or result history; confirmed
  actions update the interface and status immediately.
- The confirmation dialogs work consistently in the native macOS WebView and
  the mobile-responsive companion interface.

## Google Sheets integration

- Updated Mapper uploads to the complete 16-column Additions Queue schema,
  including the Confirmed Contact field.
- Updated the bundled Apps Script to use the current spreadsheet scope, lock
  simultaneous appends, and neutralize formula-prefixed imported text.
- Preserved receiver/job provenance, identification evidence, activity timing,
  location, transcript, callsign, and protocol fields in exported rows.

## Packages

- macOS Universal DMG and separate Apple Silicon and Intel ZIP packages
- Debian/Ubuntu amd64 and arm64 packages
- Windows 10/11 x86_64 ZIP package
- SHA-256 checksums for every downloadable artifact

The macOS packages are ad-hoc signed but not Apple-notarized. Windows packages
are not Authenticode-signed. Encrypted P25 calls are identified and skipped;
GP-SDR does not decrypt encrypted traffic or transmit.
