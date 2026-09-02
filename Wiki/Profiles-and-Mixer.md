# Profiles, Scanning, and the Mixer

In **1.5.0-rc9**, the navigation item **Channels** opens Channels & profiles (called **Profiles** in 1.4.1). Instructions referring to Profiles use this same workspace.

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
- Adjust analog/channel row volume and, in Advanced mode, left/right pan
- Set channel priority in the profile for ducking behavior
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

## P25 audio differs from channel audio

P25 talkgroup rows expose mute and solo and use the selected system audio output. They do not expose the analog rows’ volume/pan sliders. Use the P25 page’s search, Active only, and Order controls to find a talkgroup quickly. See [P25 and Decoders](P25-and-Decoders).

For a useful first scan, duplicate a small built-in bank, disable out-of-area channels, assign one free receiver, enable recording if needed, and start Live. Confirm the mixer shows activity on the expected channel; review a recording under [Activity](Activity-Recordings-and-Storage).

---

1.4.1 baseline source: [app.js](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/app.js), [channel_import.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/channel_import.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
