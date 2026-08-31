# Mapper: Map, Discovery, and Identify

Mapper runs unattended receiver jobs and keeps RF evidence for later review.
Each physical receiver may own one independent job, so two SDRs can scan
different ranges, split a large survey, or run Discovery and Identify at the
same time.

Enable **Use all connected receivers** to fan one job template out across every
currently connected and available SDR. Each receiver gets an independent job
with its own capture settings; a receiver already occupied by another workflow
is skipped and reported instead of interrupting that workflow.

All three workflow buttons—**Map**, **Discovery**, and **Identify**—remain
visible in Beginner and Advanced modes. **Decoder** defaults to automatic:
frequency-specific tools are used where applicable and digital-looking captures
are offered to DSD-FME. A job can instead force DMR, another DSD-FME protocol,
rtl_433, dump1090, multimon-ng, acarsdec, or AIS-catcher. For DMR, use about
2.5 seconds or more per channel when practical so a burst has enough audio to
produce a valid frame.

## Map

### Console layout (1.4)

Job setup is on the left and live progress, spectrum/waterfall, and saved jobs
are on the right. On phones these stack vertically; swipe the bottom navigation
bar to reach every page. Start and Save are directly below the workflow buttons.
Use Range preset for a common band or enter custom bounds. Receiver refresh is
next to the receiver selector and is unavailable while jobs own radios. A
diagnostic warning does not automatically exclude a working HackRF from RX.
Counters continue updating while you edit a job. Changes apply when you save or
start that job, not silently to another running job.

Map is the beginner/default workflow. It repeats a start/end range until
stopped and performs discovery, local waveform classification, available
protocol decoding, and enabled offline transcription during the same job.
Time per channel is adjustable from 0.1 seconds through 7 days. Longer values
are processed as bounded captures, so Stop remains responsive and the app never
allocates a multi-hour IQ buffer.

Receiver gain, sensitivity, and the HackRF RF amplifier default to **Auto**.
HackRF has no native RF AGC, so GP-SDR adjusts LNA/VGA and the amplifier between
captures. The amplifier is enabled only after repeated weak captures and is the
first stage removed when clipping or overload appears. The job card displays
the actual selected values and most recent decision. Advanced mode exposes
saved-calibration and manual controls.

For native RTL-SDR devices, Auto deliberately starts with a low manual tuner
gain and steps through common E4000/R820 gain values from capture telemetry.
The dongle driver's own automatic gain can saturate on nearby FM or paging
transmitters, so GP-SDR does not use that setting for unattended Mapper jobs.

## Managed IQ evidence

Mapper no longer archives every narrowband observation at the full wideband
HackRF rate. It shifts the detected channel to baseband and stores only the rate
needed for later local analysis and decoders. Evidence stays in **Analyzing**
until classification, available decoding, and enabled transcription finish.
Useful evidence moves to **Retained**. Low-value evidence moves to a recoverable
**Rejected** quarantine, which is removed automatically after 24 hours by
default. The recovery period is adjustable from one hour to seven days in
Settings. Pending and retained evidence are not part of that short cleanup.

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
- Channels at once (1–32, or Auto)
- Optional location and precision

Nearby frequency steps are measured from the same wide IQ capture instead of
retuning once for every step. Auto monitors up to 16 Discovery steps at once.
The actual batch may be smaller when the selected sample rate cannot contain
all requested frequencies. One receiver process remains in control, so this
increases DSP work without multiplying USB bandwidth.

The job card shows current frequency, pass and channel progress, elapsed time,
hits, and an ETA for the current pass. Because Discovery loops, ETA is the end
of the pass—not the end of the job.

The live operation strip lists the current workflow, receiver capture width,
batch number, and every software VFO being checked. The spectrum marks those
VFOs over the latest IQ capture, while the waterfall shows activity over time.
With multiple receiver jobs, this display follows the job matching the most
recent capture and labels the receiver responsible for it.

For a 10 MHz–6 GHz survey, use realistic steps and expect a long pass. Splitting
the spectrum by antenna and receiver produces more useful results than one
undifferentiated sweep.

## Identify

Identify revisits a list of discovered or imported frequencies and spends
longer on each one. Listening time is adjustable from 5 seconds through 7 days.
It attempts modulation/type classification, decoder matching, callsign and
identity extraction, local transcription when enabled, and comparison with
available local/reference data.

Use the eligibility controls to avoid spending time on one-off noise:

- **Minimum hits** requires a chosen number of successful observations.
- **Hit history** can use Discovery only, preventing Identify revisits from
  promoting a one-off result, or use the combined Discovery + Identify history.
- **Successful checks** requires 10–100% occupancy, calculated as hits divided
  by checks; **Any percentage** disables that threshold.
- **Last active**, **Maximum channels**, and **Channel order** bound and
  prioritize each Identify pass.

Identify observations still increment the combined hit/check totals and are
shown separately from Discovery totals in the expanded evidence view and CSV.

Identify can monitor nearby found frequencies at the same instant when they fit
inside the SDR's sampled bandwidth. Auto uses up to four at once because
classification, demodulation, decoder matching, recording, and transcription
cost more CPU than Discovery. Select 1 for the lightest load, or raise the limit
through 32 on a faster computer. Frequencies outside the current sample window
are automatically split into later capture batches.

The result table expands when a frequency is clicked. Details include:

- Identified name and evidence source
- Peak activity hours and time zone
- Hits, checks, occupancy, peak/noise levels, and confidence
- Modulation or protocol classification
- Decoder evidence and installation state
- Decoded or transcript-derived callsigns
- Last observation and optional location
- Local analysis and offline transcript

The top bar separately counts **Successfully identified** frequencies. This is
a deliberately stricter result than a classifier guess: it requires either a
valid decoder message or an exact frequency match in a RadioReference profile
whose imported search area is close enough to the Mapper observation location.
Band-plan labels, likely modulation, decoder candidates, and distant reference
matches remain useful evidence but do not increase the verified count.

Results can also be searched across frequency, names, protocols, modulation,
callsigns, decoder evidence, automatic analysis, and transcription. Job,
receiver, type, identification/activity, and sort controls can be combined.
Enable **Repeated only** to hide every frequency with one or zero hits. More
specific result filters can show one-offs, repeats, 10+ hits, recently seen
channels, decoder evidence, transcripts, or callsigns. Results can be ordered
by hits, checks, occupancy, signal level, SNR, confidence, receiver count,
identity, discovery time, recency, or frequency. Collapse the complete results
area when only live Mapper progress is needed.

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
- **Send eligible now** queues pending results.
- **Auto upload** appends new observations automatically when configured.
- **Identified only** (recommended) permits only successfully identified rows.

The CSV always includes `fully_identified`, `verification_reason`, and
`reference_distance_miles` columns so a saved file can be audited independently
of the interface.

Mapper writes to an **Additions Queue**, never directly to a trusted master
channel list. Rows remain `New` until a person verifies them. See
[Data and Integrations](Data-and-Integrations) for the 16-column schema and Apps
Script setup.

## Location privacy

Location is opt-in. Choose approximate, exact, or city/region precision and add
a label such as Home or Field site. The macOS app requests system location
permission when **Use current location** is pressed; coordinates can also be
entered manually. Disable location for exports you plan to share publicly.
