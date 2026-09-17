# RF lab implementation and acceptance

Primary target: configured Raspberry Pi. Start with an independent TX-capable
radio and a separately assigned receiver. HackRF is half duplex; do not promise
simultaneous TX/RX on the same HackRF. Pluto full-duplex/second-channel support
requires a separately validated driver path.

## Implementation sequence

1. Install missing decoder executables and verify app discovery. Installation
   alone is not successful decoding. dump1090 and DSD-FME/mbelib are now installed
   after upstream-notice acknowledgement; real protocol acceptance remains open.
2. Enforce ownership across Mapper, tuner, P25, transmit and diagnostic tools.
   The new TX check prevents taking an actively collecting Mapper receiver.
   This is not yet a universal atomic reservation manager.
3. Verify bounded AM/NFM/WFM file/fixture generation offline, including round-trip
   modulation/demodulation, frequency offset, filtering and malformed input.
4. Add coordinated receiver-first start, transmitter start, emergency stop,
   timeout and a single lab status/result record. Preserve independent RX until
   explicitly stopped; avoid global Stop killing unrelated collection.
5. Extend standardized digital fixtures only after they pass the corresponding
   independent decoder. Generated arbitrary FSK is not DMR or P25 voice.
6. Perform physical acceptance only after confirming wiring, isolation/load,
   attenuation, receiver input limits and an appropriate frequency. Start with
   output disabled, then bounded low-gain bursts. Low gain alone is not adequate
   receiver protection or evidence that radiated use is authorized.

## Bench evidence

Record exact TX/RX serials, firmware/backend, sample rates, center/listen
frequencies, levels, cables/load/attenuation, fixture checksum, received IQ,
decoded payload/audio and stop behavior. Compare against known-good software.
Do not play test audio into the user's headphones. No RF was emitted during
the initial decoder-install/ownership work.
