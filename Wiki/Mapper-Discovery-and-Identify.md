# Mapper: Discovery and Identify

Mapper runs unattended receiver jobs and keeps RF evidence for later review.
Each physical receiver may own one independent job, so two SDRs can scan
different ranges, split a large survey, or run Discovery and Identify at the
same time.

## Discovery

Discovery repeatedly sweeps the configured start/end range until stopped. It
records frequencies that exceed the adaptive noise threshold, how many checks
and hits occurred, occupancy, strongest signal, noise floor, first/last seen,
and activity by hour.

Configure:

- Receiver
- Start and end frequency
- Step size
- Dwell time per step
- Preferred mode or Auto
- Capture width/sample rate
- Optional location and precision

The job card shows current frequency, pass and channel progress, elapsed time,
hits, and an ETA for the current pass. Because Discovery loops, ETA is the end
of the pass—not the end of the job.

For a 10 MHz–6 GHz survey, use realistic steps and expect a long pass. Splitting
the spectrum by antenna and receiver produces more useful results than one
undifferentiated sweep.

## Identify

Identify revisits a list of discovered or imported frequencies and spends
longer on each one. Listening time is adjustable from 5 seconds through 7 days.
It attempts modulation/type classification, decoder matching, callsign and
identity extraction, local transcription when enabled, and comparison with
available local/reference data.

The result table expands when a frequency is clicked. Details include:

- Identified name and evidence source
- Peak activity hours and time zone
- Hits, checks, occupancy, peak/noise levels, and confidence
- Modulation or protocol classification
- Decoder evidence and installation state
- Decoded or transcript-derived callsigns
- Last observation and optional location
- Local analysis and offline transcript

Protocol labels remain evidence-based. A signal inside an ADS-B or P25 band is
only a candidate until the matching decoder produces valid frames or metadata.

## Saved jobs and shared configurations

Use New, Save, Edit, Duplicate, Start, Stop, Delete, Export, or bulk Import on
job cards. Deleting a job removes its saved settings but keeps mapped results.
Clearing results removes the complete observation history while preserving jobs
and settings; both actions use an in-app confirmation dialog.

## Export and spreadsheet queue

- **Save CSV** writes under the server's `Exports/Mapper` directory.
- **Download** saves the CSV to the current client device.
- **Queue** sends one reviewed observation to a configured Google Sheet.
- **Send all** queues pending results.
- **Auto upload** appends new observations automatically when configured.

Mapper writes to an **Additions Queue**, never directly to a trusted master
channel list. Rows remain `New` until a person verifies them. See
[Data and Integrations](Data-and-Integrations) for the 16-column schema and Apps
Script setup.

## Location privacy

Location is opt-in. Choose approximate, exact, or city/region precision and add
a label such as Home or Field site. The macOS app requests system location
permission when **Use current location** is pressed; coordinates can also be
entered manually. Disable location for exports you plan to share publicly.
