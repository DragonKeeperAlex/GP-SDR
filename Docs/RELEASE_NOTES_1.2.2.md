# GP-SDR 1.2.2

GP-SDR 1.2.2 adds selectable DMR and conventional digital-decoder workflows
throughout the native application.

## Added

- DMR, conventional P25, NXDN, D-STAR, YSF, M17, POCSAG/FLEX, and ACARS choices
  in Tuner and Live.
- DMR discriminator routing to DSD-FME with decoded voice returned through
  GP-SDR audio.
- DMR time-slot, color-code, talkgroup, source-radio, and encrypted-state
  parsing when present in valid decoder output.
- Decoder selectors for every custom channel and range, plus a New
  configuration action on decoder pages.
- A per-job Mapper decoder selector covering DSD-FME/DMR, rtl_433, dump1090,
  multimon-ng, acarsdec, and AIS-catcher.
- Automatic Mapper routing of locally classified digital captures to DSD-FME.
- A DMR Conventional built-in profile split into receiver-safe subranges.

## Fixed

- Map, Discovery, and Identify are all visible in Beginner and Advanced modes.
- Mapper passes compacted IQ capture metadata to file-based decoders.
- Digital channels are no longer omitted from channel/range receiver loops when
  a decoder is assigned.
- Wideband channel banks no longer send digital discriminator noise to speakers.

## Verification

- Go unit, HTTP acceptance, UI regression, race, and vet suites.
- JavaScript syntax validation.
- Executable smoke tests for DSD-FME/DMR, rtl_433, dump1090, multimon-ng,
  acarsdec, and AIS-catcher on the release Mac.
- Universal macOS architecture, signature, DMG, archive, checksum, and launch
  verification. RF decoding still depends on an active compatible signal,
  antenna, receiver calibration, and local conditions. Encrypted voice is not
  decrypted.
