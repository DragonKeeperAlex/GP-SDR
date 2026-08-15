# Google Sheets range sync

GP-SDR can automatically update read-only scan-range profiles from a shared
Google Sheet. Packaged profiles remain in the app as offline fallbacks; sheet
profiles are added alongside them and are cached after every successful sync.

## Connect a sheet

1. In Google Sheets, share the sheet so anyone with the link can view it, or use
   **File → Share → Publish to web**.
2. Open **Settings → Range updates** in GP-SDR.
3. Paste the normal Google Sheets link. Keep the `gid` in the link when the
   ranges are on a tab other than the first tab.
4. Choose an update interval, enable **Auto sync**, and press **Save**.

GP-SDR accepts the normal edit/share link and converts it to a receive-only CSV
export internally. It only connects to HTTPS Google Sheets hosts and limits a
sheet export to 5 MB.

## Columns

The first row contains column names. One row defines one scan range:

| Column | Required | Example | Notes |
| --- | --- | --- | --- |
| `Profile` | No | `East Bay` | Rows with the same value become one built-in profile. |
| `Name` | No | `Civil Airband` | Range label shown in GP-SDR. |
| `Start MHz` | Yes | `118` | `MHz`, `kHz`, or `Hz` suffixes are also accepted. |
| `End MHz` | Yes | `136.975` | Must be higher than the start frequency. |
| `Step kHz` | No | `25` | Defaults to 12.5 kHz. |
| `Mode` | No | `AM` | AM, NFM/FM, WFM, digital, P25, or auto. |
| `Dwell ms` | No | `160` | Defaults to 180 ms; minimum 20 ms. |
| `Enabled` | No | `yes` | Accepts yes/no, true/false, on/off, or 1/0. |
| `Summary` | No | `Portable aviation ranges` | Profile note; the first non-empty value is used. |

Example:

```csv
Profile,Name,Start MHz,End MHz,Step kHz,Mode,Dwell ms,Enabled,Summary
Common ranges,Civil Airband,118,136.975,25,AM,160,yes,Shared receive ranges
Common ranges,GMRS,462.55,467.725,12.5,NFM,180,yes,
Public safety,700 MHz,769,775,12.5,digital,180,yes,Public-safety discovery
```

If a download or validation fails, GP-SDR keeps the last valid cached profiles
and displays the error in Settings. A partially invalid sheet is never applied.
