# Mapper files and Google Sheets

GP-SDR keeps discoveries reviewable. It does not write a detected frequency
directly into a trusted master channel tab. **Identify results** can instead
send observations to the spreadsheet's **Additions Queue**, where a person can
verify the source, identity, and current assignment.

## Save or download a file

- **Save CSV** writes a timestamped file to the GP-SDR data directory under
  `Exports/Mapper` on the server Mac.
- **Download CSV** downloads the same columns to the device using the web app.
- Exports contain frequency, identity, modulation/protocol, timestamps,
  checks, hits, occupancy, signal/noise levels, confidence, decoded callsigns,
  transcript, and optional location.

## Connect an Additions Queue

1. Create or open the target Google Sheet. Add a tab named `Additions Queue`.
2. Put these exact headers in row 4, columns A through O:

   `Date Added`, `Contributor`, `Type`, `Name/Label`, `RX MHz`, `TX MHz`,
   `Mode`, `Tone/Code`, `Location/System`, `What Was Heard`,
   `Date/Time Heard`, `Source URL/File`, `Confidence`, `Review Status`, and
   `Reviewer Notes`.

3. In GP-SDR, open **Mapper**, then select **Download setup script**.
4. In the target sheet, open **Extensions → Apps Script** and replace the
   editor contents with the downloaded script.
5. Replace `CHANGE_ME` with a private random secret. Do not put that secret in
   shared profiles or screenshots.
6. Deploy the script as a web app that executes as the sheet owner. Copy its
   `/exec` URL.
7. In GP-SDR, enter the normal master-sheet URL, Apps Script webhook URL,
   contributor name, and matching shared secret, then press **Save**.

Use **Queue** beside one Identify result, **Send all to sheet** for pending
results, or enable automatic queuing. New rows are deliberately marked `New`;
GP-SDR never marks its own observations verified. The included receiver checks
the exact 15-column layout, uses a document lock for simultaneous submissions,
and neutralizes spreadsheet-formula prefixes in decoded text.

The webhook grants a URL holder the ability to append review rows through the
script, so keep its URL and secret private. Re-deploy or change the secret if
either is shared accidentally.
