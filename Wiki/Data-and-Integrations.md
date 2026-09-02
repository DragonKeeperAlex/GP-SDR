# Data, Imports, Google Sheets, and RadioReference

In **1.5.0-rc9**, the navigation item **Channels** opens Channels & profiles (called **Profiles** in 1.4.1). Instructions referring to Profiles use this same workspace.

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

## Import a folder of official channel exports

On the GP-SDR host, open **Settings → Local database → Choose folder**, select the export folder, and scan it. GP-SDR recursively reads `.csv`, `.tsv`, and GP-SDR `.json` files. Large tables are split into stable 4,000-channel banks so rescanning updates those banks rather than creating a fresh duplicate set. Check the resulting profile names and channel counts before scanning.

Folder selection/scanning is local-computer only. Keep downloaded source files as a recoverable reference and use the source’s receive-area information; an exact frequency match alone cannot establish local identity.

For live API lookup, configure `GPSDR_RR_USERNAME`, `GPSDR_RR_PASSWORD`, and `GPSDR_RR_APP_KEY` in the host process environment, then restart. Keep these credentials out of shared profiles and screenshots.

Detailed formats and examples: [Channel data/import columns](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/CHANNEL_DATA.md), [Sheets range schema and example CSV](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/GOOGLE_SHEETS_SYNC.md), [Mapper queue deployment](https://github.com/DragonKeeperAlex/GP-SDR/blob/main/Docs/MAPPER_SHEET_WRITE.md). Read-only range synchronization and writable Mapper queue upload are separate integrations with separate configuration.

---

1.4.1 baseline source: [CHANNEL_DATA.md](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Docs/CHANNEL_DATA.md), [MAPPER_SHEET_WRITE.md](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Docs/MAPPER_SHEET_WRITE.md).

## macOS RadioReference credentials

On the local Mac, the RadioReference account card accepts Username, Password, and Application key with **Save to Keychain**. Enter your own authorized credentials, save, then use **Channels → Location import**. **Clear** removes the configured Keychain credentials. This form is host-local and macOS-specific; other hosts use the environment variables above. Never include credentials in a shared profile, wiki example, or screenshot.

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
