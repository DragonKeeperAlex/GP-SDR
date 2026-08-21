# Profiles, Scanning, and the Mixer

Profiles group scan ranges, fixed channels, P25 systems, receiver roles, and
logging behavior into a shareable configuration.

## Built-in profiles

GP-SDR includes GMRS, NOAA Weather Radio, MURS, CB, AM and FM broadcast, civil
air, marine VHF, 2 m, 70 cm, public-safety discovery, decoder target ranges,
California GMRS repeaters, and sanitized regional channel banks. Regional or
repeater assignments can change; treat bundled data as a starting point and
retain its source/date when updating it.

Built-in profiles are read-only. Duplicate one before editing.

## Create a custom profile

1. Open **Profiles** and choose **New**.
2. Name the profile and add ranges, channels, or P25 systems.
3. For a range, set start/end, step, dwell, mode, and enabled state.
4. For a channel, set name, receive frequency, bandwidth, mode, decoder, and
   priority.
5. Assign receivers by role or leave them automatic.
6. Choose audio recording, unidentified IQ capture, transcription, revisit
   timing, retention, and P25 capture width.
7. Save, select the profile, and press **Start** on Live.

## Receiver roles

- **Control** follows a P25 control channel.
- **Voice** receives granted P25 traffic channels.
- **Survey** sweeps ranges for activity.
- **Tuner** provides direct listening.
- **Channel bank** demodulates fitting fixed channels from one wide IQ stream.

Explicit device assignments are honored first. Compatible idle receivers fill
unassigned roles. Multiple SDRs can increase coverage or separate P25 control,
voice, and Mapper work.

## Import and export

Profiles accepts multiple files in one operation:

- GP-SDR JSON profiles
- CHIRP CSV
- Generic CSV/TSV with `Frequency` or `FrequencyHz`, plus optional `Name` and
  `Mode`

Each CSV/TSV becomes a complete channel-bank profile. Values under 1,000,000
are interpreted as MHz; larger values are treated as Hz. Export a profile to
share a channel, bank, range set, device-role plan, or P25 configuration.

## Channel and talkgroup mixer

The mixer shows active channels and P25 calls with signal/activity state.
Controls are independent:

- Mute or unmute one row
- Solo a row
- Adjust row volume and left/right pan
- Mark priority channels for ducking behavior
- Mute all channels or use the top-bar master control

P25 talkgroups are ordered by recent or frequent activity rather than
alphabetically. The current control channel and system status are displayed on
the P25 page.

## Logging and retention

Activity events are journaled with time, frequency, identity, protocol,
callsigns, transcript, system/talkgroup, signal evidence, and linked recordings
when available. Retention can be 1, 7, 30, 90, or 365 days, or Forever. Long
retention and IQ capture can consume substantial storage; monitor the Data card
in Settings.
