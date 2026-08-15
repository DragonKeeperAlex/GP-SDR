# Bundled channel data

GP-SDR is receive-only. Imported duplex offsets and transmit settings are never
used to key or control a transmitter. Frequencies and system assignments can
change; use the location importer or current agency/database information to
refresh a profile before relying on it.

## San Ramon and East Bay

The `San Ramon · Verified Local` and `San Ramon · EBRCS P25` profiles were
reviewed against public, subscription-free sources on 2026-08-15:

- [East Bay Regional Communications System](https://www.radioreference.com/db/sid/5317)
- [San Ramon Valley Fire Protection District conventional channels](https://www.radioreference.com/db/subcat/34794)
- [California Highway Patrol Golden Gate Communications Center](https://www.radioreference.com/db/subcat/11163)
- [Contra Costa County amateur radio](https://www.radioreference.com/db/browse/ctid/189/ham)
- [Alameda County conventional channels](https://www.radioreference.com/db/browse/ctid/183)
- [Cal OES interoperability channels](https://www.radioreference.com/db/aid/1009)
- [Caltrans District 4](https://www.radioreference.com/db/subcat/35196)

Encrypted talkgroups are retained only so the UI can identify and automatically
skip them. GP-SDR does not include or attempt decryption.

The `BART · P25` profile comes from the public
[BART P25 system page](https://www.radioreference.com/db/sid/12049). Its
above-ground and underground site lists are separate inside the profile so
multiple receivers can be assigned when needed.

## Local programming archive

These built-in banks were generated from the maintainer's existing receive
programming files. Personal callsigns, comments, and transmit-only settings were
removed before publication:

| Built-in profile | Channels | Local source |
| --- | ---: | --- |
| San Ramon · Handheld Bank | 137 | `San_Ramon_CHIRP_Scanner(1).csv` |
| NorCal · Travel Bank | 199 | `TD-H3Plus_NorCal_GMRS_Travel_v1_199mem_CHIRP.csv` |
| NorCal · Rubicon Bank | 122 | `TD-H3Plus_NorCal_Travel_Rubicon_CHIRP.csv` |
| California · GMRS Repeaters | 84 | `CA GMRS.csv` |

The sanitized copies embedded in GP-SDR contain channel name, receive frequency,
and mode. California repeater names also preserve the receive tone or DCS value
where the local file specified one.

## Bulk imports

Profiles → Import accepts multiple files at once:

- GP-SDR JSON profiles
- CHIRP CSV files
- Generic CSV/TSV files with `Frequency` or `FrequencyHz` plus optional `Name`
  and `Mode` columns

Each CSV/TSV becomes a complete channel-bank profile in one operation. Values
below 1,000,000 are interpreted as MHz; larger values are interpreted as Hz.
AM, FM/NFM, WFM, P25, and common digital mode labels are recognized.

## Original diagrams

The Settings page includes two maintainer-provided SVG references from the local
radio project archive:

- `GMRS_Repeater_Rev3_Accurate_Schematic.svg`
- `GMRS_Repeater_Rev3_Exact_Solder_Map_v2.svg`

They are documentation only and are not connected to GP-SDR receiver control.
