# P25 Trunking and Decoder Pages

In **1.5.0-rc9**, the navigation item **Channels** opens Channels & profiles (called **Profiles** in 1.4.1). Instructions referring to Profiles use this same workspace.

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
5. Choose P25 capture width. HackRF Auto starts at 10 MS/s and falls back if
   transport becomes unstable. RTL-SDR Auto uses 2.4 MS/s; its dropdown excludes
   HackRF-only rates. Wider capture does not itself improve antenna sensitivity.
6. Start the profile and open **Decoders → P25/SDRTrunk**.
7. Create JMBE locally if unencrypted calls decode but remain silent.

## P25 acceptance indicators

In version 1.4, select a System profile and Receiver directly on the P25 page,
then Start P25. **Profile assignments** preserves your existing multi-radio plan;
selecting one named receiver makes a single-receiver configuration. Search and
Active only filter the talkgroup mixer; Order selects most recent or most heard.
HackRF RF amp/LNA/VGA fields preserve saved SDRTrunk settings when left at Saved
or blank. Explicit overrides are saved to the profile and restart active P25.
They are HackRF-only controls, not an RTL-SDR RF amplifier.

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

Installation and verification instructions for every decoder are in [Optional
Components](Optional-Components).

### DMR

Choose **DMR** in Tuner for one known channel, or open **Decoders → DSD-FME →
New configuration** to create a reusable channel bank or scan range. Profiles
allow the mode and decoder to be selected for each channel and range. When
DSD-FME reports them, GP-SDR records the time slot, color code, talkgroup,
source radio ID, and encrypted state. Both DMR time slots are enabled. Encrypted
calls remain identified but are not decrypted.

## Transcription

Optional whisper.cpp transcription runs locally. Install the executable and
pinned model from its integration card, then enable transcription in a profile
or **Dictate voice** in Mapper Identify. Transcripts appear in Activity search,
expanded Mapper evidence, and exports. Poor or digital/encrypted audio will not
produce reliable speech text.

## Start another decoder from its workspace

1. Open Decoders and choose the tool for the signal type in the table above.
2. Use its Install/How to action until the component is Ready. A ready executable is only setup confirmation.
3. Choose a matching built-in profile, or press **New configuration** and create a channel bank or range with verified frequencies and the intended decoder.
4. Save/use that profile, assign an idle receiver, and start Live.
5. Check the decoder’s **Recent activity** and Activity Timeline for valid messages. Return to Tuner to verify RF quality if there are no messages.

For Mapper, choose the same backend in the job’s **Decoder** field, or Auto for routing by signal/target. For a known DMR channel, start in Tuner with DMR and confirm DSD-FME produces frames before widening the survey. DMR trunk following is not the same capability as the bundled P25 trunk engine; selecting DMR does not create a DMR trunking controller.

The P25 control display distinguishes **Configured primary** from **Decoded current**. Only decoded status and valid system/grant evidence establish reception. Search/filter controls affect the displayed mixer, while receiver and capture/gain changes can restart the active profile.

---

1.4.1 baseline source: [app.js](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/app.js), [decoder_runner.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/decoder_runner.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
