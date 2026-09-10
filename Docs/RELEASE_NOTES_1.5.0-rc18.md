# GP-SDR 1.5.0-rc18

## Results survive media cleanup

- Mapper frequency results and event history remain stored separately under `Data` when eligible IQ files or recordings are removed.
- Cleanup reconciliation removes missing media links without erasing frequency, hit, identity, transcript, decoder, profile, calibration, or channel metadata.
- A release regression test now verifies result and event files remain byte-for-byte unchanged during media cleanup.
- The Storage interface labels this boundary explicitly and reserves result deletion for the dedicated clear action.
- Age-based cleanup now includes expired media in its displayed removed-file count.

## Spectrum refresh

- One Spectrum refresh setting now clearly applies to Live, Tuner, Mapper, and the multi-receiver RF Monitor.
- Available choices range from 1 Hz for low load through 60 Hz for near-real-time display.
- The previous internal 40 ms polling floor, which limited the effective maximum to 25 Hz, is removed.
- Display refresh remains independent of SDR sample rate, RF bandwidth, capture quality, and decoder timing.

The 40–60 Hz modes increase browser and local-server load. They expose faster display polling but cannot create new RF frames faster than the active receiver workflow produces them.
