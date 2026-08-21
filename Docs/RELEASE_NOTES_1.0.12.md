# GP-SDR 1.0.12

## Unattended monitoring

- Prevents idle system sleep while the native macOS app is open, allowing long
  scans, Mapper sessions, recordings, and P25 monitoring to continue.
- Releases the macOS activity assertion immediately when GP-SDR quits. The
  display may still sleep normally.

## HackRF P25 control-channel acquisition

- Prioritizes the live-verified EBRCS CCCO Central control channel at
  774.45625 MHz while retaining the alternate control channels for rotation.
- Gives the HackRF longer to synchronize before rotating to the next control
  candidate, improving acquisition of weak or simulcast P25 signals.
- Automatically reduces inherited HackRF rates above 5 MS/s for compact P25
  sites, lowering USB/channelizer load while retaining enough bandwidth for
  the site's control and traffic channels. Wider systems keep their configured
  rate.
- Reports a control-channel lock only from decoder output created during the
  current session, preventing an older event log from producing a stale lock.
- Verified receive-only with the RTL-SDR disconnected: SDRTrunk locked to NAC
  `0x1F5`, system `0x1F1`, WACN `0xBEE00`, RFSS 1/site 5, and followed live
  Phase 1/2 traffic grants through the connected HackRF.

## Mapper visibility and deciphering

- Shows the current tuned frequency, channel/pass position, checks, hits,
  completed passes, elapsed time, last activity, and live pass progress.
- Adds a lightweight progress endpoint so the native app and mobile Web UI can
  update Mapper state without repeatedly downloading the complete result list.
- Lets Decipher listen to each frequency for 5 seconds through 7 days, using
  seconds, minutes, hours, or days. Long sessions use bounded receive windows
  so Stop remains responsive and memory use stays controlled.
- Makes every Mapper result expandable with identity source, classification,
  callsigns, transcript, signal evidence, location, and peak activity hours.
- Prefers exact matches from imported RadioReference data, the configured local
  database, and saved profiles before applying the built-in US band plan.
- Records hourly activity and identification provenance in Mapper CSV exports.

Mapper identifications remain receive-side observations. Database matches and
automatic classifications should be reviewed before relying on them.
