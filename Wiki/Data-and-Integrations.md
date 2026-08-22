# Data, Imports, Google Sheets, and RadioReference

## Channel and profile imports

Use **Profiles → Import** to select multiple GP-SDR JSON, CHIRP CSV, or generic
CSV/TSV files. Generic tables need `Frequency` or `FrequencyHz`; `Name` and
`Mode` are optional. Imported data is validated for size, counts, frequency
range, and required P25 fields before it becomes active.

Imported duplex offsets and transmit settings are not used to transmit. Remove
personal callsigns, access notes, or exact private locations before sharing.

## Shared range updates

Settings can synchronize read-only scan-range profiles from a public or
published Google Sheet. Supply a normal Sheets link, preserve its `gid` when the
data is on another tab, choose an interval, enable Auto sync, and Save.

One row defines one range. Supported columns are `Profile`, `Name`, `Start MHz`,
`End MHz`, `Step kHz`, `Mode`, `Dwell ms`, `Enabled`, and `Summary`. Invalid
downloads never replace the last known-good cache. GP-SDR accepts only HTTPS
Google Sheets hosts and limits exports to 5 MB.

## Mapper Additions Queue

Create a sheet tab named `Additions Queue`. Put these headers in row 4, columns
A through P:

`Date Added`, `Confirmed Contact`, `Contributor`, `Type`, `Name/Label`,
`RX MHz`, `TX MHz`, `Mode`, `Tone/Code`, `Location/System`, `What Was Heard`,
`Date/Time Heard`, `Source URL/File`, `Confidence`, `Review Status`, and
`Reviewer Notes`.

In Mapper:

1. Download the setup script.
2. Paste it into **Extensions → Apps Script** in the target sheet.
3. Replace `CHANGE_ME` with a private random secret.
4. Deploy as a web app executing as the sheet owner.
5. Copy the `/exec` URL into GP-SDR.
6. Add the normal sheet URL, contributor name, and matching secret.
7. Save, then test one result with **Queue**.

The script locks simultaneous appends and neutralizes spreadsheet-formula
prefixes in decoded text. Keep the webhook URL and secret private. GP-SDR marks
new observations `New`; it never changes the sheet's human review status. Keep
**Identified only** enabled when you want the app to send only records backed by
valid decoder output or a nearby authoritative RadioReference match.

## RadioReference

Two supported paths are intentionally separate:

- **Live lookup** requires your RadioReference Premium account and an approved
  application key. GP-SDR cannot provide or share a key for other users.
- **Local database folder** accepts official exports you downloaded while
  signed in. Point Settings at the folder and use location/radius import without
  putting account credentials in GP-SDR.

Location import supports 5, 10, 25, 50, or 100 miles and a custom 1–100-mile
radius. Review the returned counties, conventional channels, and P25 sites
before creating profiles. The import saves its center, radius, and provider in
the profile. Mapper will not use that RadioReference identity as verification
unless the current observation location is inside the imported area (with a
small allowance for approximate or city-level locations). This prevents a
same-frequency listing in another state from being accepted as the local use.

RadioReference does not offer a licensed one-click global mirror through a
normal Premium account. Its standard web service requires an approved
application key and each user must have their own Premium subscription; its
published terms also prohibit recreating or mirroring the database without a
separate commercial license. GP-SDR therefore supports authorized live lookup
and user-supplied official exports, not site scraping or an embedded global
database.

## Activity data

Activity offers indexed search across frequency, label, transcript, callsign,
protocol, system, talkgroup, and decoder output. Recordings and IQ files remain
in GP-SDR-owned storage. The Settings Data card can enforce separate Recording
and IQ caps, age-based retention, and optional automatic cleanup. Cleanup is
off by default and removes oldest captures only; Mapper results, profiles,
calibration, range data, and imported channel files are preserved.
