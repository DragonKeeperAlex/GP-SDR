# Band monitor

**Added in 1.5.0-rc9.** Band monitor gives a channel bank a dedicated listening page, with independent channel audio and CTCSS evidence. It uses a shared receiver capture for channels that fit the available bandwidth.

## Start a band

1. Verify one known channel in Tuner, then stop Tuner to release the radio.
2. Open **Band monitor** and choose **Band**. This list uses profiles with more than one fixed channel and no P25 system. Use **Channels → Import** or **New profile** if your bank is missing.
3. Choose an idle Receiver. Verify the bank’s entire frequency span can fit the selected capture width if you need simultaneous coverage. The page’s Capture summary reflects the selected bank span, not an independent measurement of USB sample rate.
4. Select a sample rate appropriate to the receiver. Choosing HackRF sets 20 MS/s in this page; choosing RTL-SDR sets 2.4 MS/s. Receiver limits still apply, and narrower radios cannot cover a wide bank simultaneously merely because its channels are listed.
5. Begin with Auto gain and DC removal enabled, RF amp off, and the default squelch. Use manual generic gain or HackRF LNA/VGA controls only as needed for measured signal quality.
6. Press **Monitor band**. Inspect Channels, Active now, Last tone, and the active channel rows.
7. Search by channel, frequency, or code. Use **M** to mute, **S** to solo, and the slider for that channel’s volume. The header M button mutes/unmutes all rows.
8. Press **Stop** before handing the radio to another workflow. Changing receiver or RF settings while running reapplies/restarts the active band configuration.

## Carrier mode and privacy codes

**Carrier · all codes** listens without requiring a matching tone. So-called privacy codes are squelch signaling, not encryption. A CTCSS value appears only when detected in received audio; **No code detected** means there is insufficient tone evidence, not proof that a transmitter used no code. DCS decoding is not implemented.

The channel’s most recent event can supply the displayed tone, so check its age before treating it as the current transmission’s tone. An active meter does not establish intelligible audio; verify the signal and record/replay when diagnosing reception.

## If a band is silent or incomplete

Check master audio, mute/solo, squelch, antenna coverage, receiver ownership, and whether all frequencies fit. Create a narrower bank or use additional receivers for wider coverage. Reduce gain if many channels appear active from overload. P25 trunk following belongs on its dedicated decoder page, not this fixed-bank monitor.

Source: [Band monitor controls and rendering](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/app.js), [interface](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
