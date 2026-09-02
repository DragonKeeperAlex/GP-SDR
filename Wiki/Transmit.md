# Guarded HackRF audio-file transmission

GP-SDR normally receives. **Transmit** is an explicit local-computer workflow for bounded AM, NFM, or WFM playback from a WAV file. RTL-SDR cannot transmit. Microphone input, digital voice, repeater operation, and unattended/continuous transmission are not implemented.

## Prepare a dry run

1. Use GP-SDR on the computer hosting the radio. Remote clients cannot upload transmit audio or start/stop transmission, including the dry-run start endpoint.
2. Release the HackRF from Live, Tuner, Mapper, P25, and other SDR programs; select the connected, available HackRF.
3. Choose a **16-bit PCM WAV**, mono or stereo, no larger than 50 MB. MP3, floating-point WAV, and a renamed non-WAV file are not accepted. Stereo is mixed to mono.
4. Enter Frequency in MHz, choose AM, NFM (2.5 kHz deviation), or WFM (75 kHz deviation), and set Duration from 0.1 to 60 seconds. TX gain accepts 0–47 dB; start at 0 for later bench tests.
5. Keep **Dry run (generate IQ, no RF)** checked. Press **Prepare / transmit**.
6. Confirm the status says **Dry run complete; no RF was transmitted**. The app creates signed 8-bit IQ under the data directory’s `Transmit/iq` folder. Uploaded WAV files are under `Transmit/audio`.

Dry runs validate preparation, not RF output, spectral compliance, or audio quality at another receiver. Generated IQ is approximately 4 MB per second at 2 MS/s; a 60-second job is about 240 MB. General Recording/IQ cleanup does not target the separate Transmit folders, so review those files separately.

## Bounded RF bench test

Use a suitable 50-ohm dummy load and an authorized test setup. If measuring with another receiver, use correctly rated attenuation and verify its input limits; never connect the transmitter directly to a receiver input.

1. Confirm the intended HackRF, frequency, short duration, and minimum TX gain.
2. Resolve any Hardware diagnostic warning. Receive and dry-run availability do not mean RF transmission is allowed.
3. Confirm `hackrf_transfer` is installed through Hardware’s setup guidance.
4. Clear Dry run and check **I am using a dummy load and understand this transmits RF**.
5. Press **Prepare / transmit** and monitor locally. **Stop** cancels the job; every job has a maximum 60-second duration.
6. Confirm completed/stopped status before changing the test wiring.

The 1.4 waveform corrections were tested in software; the release notes explicitly report no RF transmission during acceptance. Do not treat that validation as an on-air acceptance test.

## Common errors

| Error | Action |
| --- | --- |
| Choose a PCM WAV / no PCM voice | Export as standard 16-bit PCM WAV, mono or stereo. |
| Select a connected, available HackRF | Refresh Hardware and release competing receive jobs. |
| Diagnostic warning | Keep receive-only or dry-run operation until the hardware issue is resolved. |
| `hackrf_transfer` is not installed | Install the HackRF host tools and refresh/restart. |
| Local-computer restriction | Operate from the host’s local interface, not a LAN client. |
| Duration or gain rejected | Keep duration at most 60 seconds and TX gain within 0–47 dB. |

---

1.4.1 baseline source: [transmit.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/transmit.go), [decoder_runner.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/decoder_runner.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
