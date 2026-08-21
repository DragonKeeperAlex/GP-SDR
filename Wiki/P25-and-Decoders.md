# P25 Trunking and Decoder Pages

## P25 stack

Release packages include SDRTrunk for P25 Phase 1/2 trunk following and JMBE
Creator for unencrypted voice codec support. GP-SDR creates an isolated
SDRTrunk playlist, starts and supervises the headless engine, and normalizes
system, grant, call, talkgroup, encryption, and audio state into the app.

## Configure a P25 system

1. Import a verified location profile or create a profile under **Profiles**.
2. Add the system name and current control-channel frequencies.
3. Add NAC, WACN, system ID, TDMA control setting, and talkgroups when known.
4. Assign one wideband receiver, or separate Control and Voice receivers.
5. Choose P25 capture width. Auto starts at 10 MS/s and falls back if the USB
   stream becomes unstable.
6. Start the profile and open **Decoders → P25/SDRTrunk**.
7. Create JMBE locally if unencrypted calls decode but remain silent.

## P25 acceptance indicators

Successful control-channel reception means more than a peak or audible digital
noise. Look for:

- Control channel locked
- NAC, WACN, and system identity
- Channel grants
- Talkgroup/source identifiers
- Traffic-channel assignments
- Valid decoded events and encryption flags

For voice, require intelligible unencrypted audio. Encrypted calls are labeled
and skipped; they cannot be decrypted by GP-SDR.

## Improve robotic or choppy P25

1. Confirm the control channel is current and correctly calibrated.
2. Turn the HackRF RF amplifier off and reduce gain if overloaded.
3. Raise LNA/VGA gradually while monitoring control error and dropped samples.
4. Use 8–10 MS/s when the traffic channels fit and USB remains stable; reduce
   width if dropped blocks increase.
5. Close competing SDR applications.
6. Verify JMBE and talkgroup mute/solo state.
7. Compare raw reception with SDR++ using the same center, bandwidth, antenna,
   and gain to separate hardware/RF issues from GP-SDR processing.

A wider sample rate increases simultaneous coverage, not antenna sensitivity.
Multiple SDRs can keep one receiver on control while others follow more voice
channels.

## Other decoder pages

| Page/tool | Targets | Successful evidence |
| --- | --- | --- |
| DSD-FME | Conventional P25 and DMR | Valid frames, protocol/system metadata, and intelligible unencrypted voice |
| rtl_433 | ISM sensors, weather sensors, TPMS | Parsed device/model/ID and field values |
| dump1090 | ADS-B and Mode S | Valid aircraft hex IDs, messages, positions, or flight metadata |
| multimon-ng | POCSAG, FLEX, MDC1200, DTMF | Parsed protocol messages or signaling identifiers |
| acarsdec | ACARS | Valid aircraft/message fields |
| AIS-catcher | Marine AIS | Valid MMSI and vessel/message fields |
| Analog | AM, NFM, WFM and CTCSS | Intelligible audio and validated tone where detected |

Decoder bridges normalize output into Activity and Mapper. The Hardware and
Decoders pages show whether each executable is bundled, installed, optional, or
missing, with Install/How-to actions where available. A frequency matching a
known decoder target is only a candidate until valid output is received.

## Transcription

Optional whisper.cpp transcription runs locally. Install the executable and
pinned model from its integration card, then enable transcription in a profile
or **Dictate voice** in Mapper Identify. Transcripts appear in Activity search,
expanded Mapper evidence, and exports. Poor or digital/encrypted audio will not
produce reliable speech text.
