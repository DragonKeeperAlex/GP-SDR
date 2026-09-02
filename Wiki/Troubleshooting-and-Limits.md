# Troubleshooting and Current Limits

## Receiver is not listed

- Press Hardware → Refresh after reconnecting.
- Put a PortaPack in HackRF USB mode.
- Use a known data-capable cable and direct/powered USB connection.
- Close other SDR programs that may own the device.
- Follow the card's Install or How-to action.

## Every frequency appears active or Overloaded remains on

1. Turn the RF amplifier off.
2. Set LNA/VGA or general gain low.
3. Enable DC removal.
4. Stop and restart the receiver to reset stream state.
5. Raise Mapper's noise margin or squelch.
6. Reconnect without an antenna and compare the noise floor.

An internal center spike or broad overload is not a real transmission. Save
calibration only after the baseline is normal.

## Static but no intelligible audio

- Use WFM for broadcast FM, AM for aviation/broadcast AM, or NFM for common
  land-mobile voice.
- Open squelch temporarily.
- Check top-bar mute/volume and row mute/solo.
- Confirm Listen VFO is inside the Hardware center passband.
- Reduce excessive gain and verify the antenna covers the band.

## Waterfall or audio is sluggish

- Close competing SDR software.
- Check dropped-block and USB health notices.
- Reduce waterfall frame rate and detail first.
- Reduce sample rate if the stream still drops.
- Disable unnecessary simultaneous decoder jobs.
- Use a powered USB controller/hub when multiple SDRs are unstable.

## P25 does not lock

- Verify the control channel is current and belongs to the intended site.
- Confirm frequency/PPM calibration with a known signal.
- Match antenna coverage to the control-channel band.
- Adjust gain for clean frames rather than maximum displayed level.
- Look for NAC/WACN/system identity and grants; RF energy alone is insufficient.

## P25 locks but voice is silent or robotic

- Create/verify JMBE for unencrypted calls.
- Check encryption, talkgroup mute/solo, and master audio.
- Turn HackRF amplifier off and reduce overload.
- Watch dropped blocks and reduce capture width if necessary.
- Add a second receiver for more voice capacity.
- Compare the same signal in SDR++ with equivalent RF settings.

## Mapper does not find useful activity

- Use a band-appropriate antenna and a smaller test range first.
- Match step and mode to the service.
- Increase dwell for intermittent channels.
- Verify a known signal in Tuner before a wide sweep.
- Remember that Discovery loops and its ETA is per pass.
- Use Identify for longer listens, decoding, and transcription.

## Clear or Delete appears ineffective

Current builds use an in-app confirmation dialog. **Delete job** removes saved
job settings but keeps results. **Clear results** removes observation history
but keeps jobs. Cancel intentionally changes nothing.

## Current external or engineering limits

- Encrypted P25 is identified and skipped, not decrypted.
- GP-SDR 1.4.1 includes a guarded HackRF transmit workspace; digital voice
  transmission and unattended/continuous transmission are intentionally not implemented.
- RadioReference live API use needs Premium access and an approved application
  key; official local exports work without one.
- macOS public packages are ad-hoc signed and not notarized; Windows packages
  are not Authenticode-signed.
- Soapy hardware, remote servers, optional decoders, and live RF results depend
  on their driver, local hardware, antenna, and signals.
- DCS subtone decoding and broader adjacent-channel rejection testing remain
  future work.

When reporting a problem, include OS/CPU, receiver model, antenna band, center
and Listen VFO, sample rate, mode/bandwidth, gains/amplifier, squelch, displayed
noise/SNR, dropped-block count, and whether the same signal works in SDR++.

## A radio appears but the wrong P25 receiver is used

Update to 1.4.1, refresh Hardware, and check the receiver serial. On P25, use Profile assignments for an existing multi-radio plan or deliberately choose one receiver. Look for decoded system/grant evidence and the current receiver identity. The 1.4.1 fix removes a preferred-name mismatch; it does not repair a disconnected USB device.

## A HackRF warning is shown but reception is available

This is intentional in 1.4: successful enumeration keeps receive testing available while the warning remains visible. Verify IQ/audio rather than treating enumeration as a complete health check. RF transmit stays blocked until the warning is resolved; do not bypass it.

## No old recording, or storage keeps growing

Check whether recording was enabled when the event occurred and whether retention removed the media. In Settings, Save limits and Automatic cleanup are separate. Remove rejected IQ has its own timer and switch. Transmit audio/IQ is stored separately and is not covered by general Recording/IQ limits. See [Activity and Storage](Activity-Recordings-and-Storage).

## Limits specific to the current release

The 1.4 release record leaves long-duration RTL USB reliability and simultaneous two-radio acceptance unresolved. Optional protocol coverage is not fully field-tested. No RF transmit acceptance was performed. A classified waveform, a transcript, a receiver menu entry, and a successful build each provide different evidence; none alone proves full on-air performance.

---

1.4.1 baseline source: [RELEASE_NOTES_1.4.1.md](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Docs/RELEASE_NOTES_1.4.1.md), [RELEASE_NOTES_1.4.0.md](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/Docs/RELEASE_NOTES_1.4.0.md).

## rc9 Analyze or Local intelligence seems stuck

Analyze refreshes every five seconds while visible. Check queued records, active stage, frequency groups, and the log before restarting. Local model inference is serialized even with several offline workers. Ready on the model card proves service reachability, not that the configured model is installed. Completed files do not guarantee every optional stage produced evidence. See [Analyze and Schedule](Analyze-and-Schedule) and [Local Intelligence](Local-Intelligence).

If there are no pending event records, Analyze is not a general-purpose raw-IQ folder importer. Stop analysis before updating; unprocessed work can be continued afterward, but rc9 does not claim every capture-recovery case is complete.

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
