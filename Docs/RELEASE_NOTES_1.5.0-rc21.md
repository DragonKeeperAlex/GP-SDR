# GP-SDR 1.5.0-rc21

## Tezuka and PlutoSDR support

- Detects the hardware model, firmware build, serial, usable transport, RX/TX count, duplex mode, AGC support, sample-rate range, filter limit, and tuning range reported by the attached Pluto-compatible board.
- Rejects advertised USB or network transports that cannot actually open, preventing an unusable duplicate from hiding a working path.
- Applies hardware AGC, manual gain, filter bandwidth, and frequency correction to SoapySDR receive streams.
- Adds guarded AM, NFM, and WFM WAV transmission for PlutoSDR/compatible boards. Dry run remains the default and no on-air transmission is performed by automated tests.
- Shows the detected Tezuka firmware and transport capabilities on Hardware.

## FISHBall-PlutoSky channel note

The tested Tezuka 0.3.21 configuration says `mode=2r2t`, and its underlying IIO graph contains two complex RX and two complex TX paths. The current upstream SoapyPlutoSDR module reports one RX and one TX and cannot safely expose the second pair as independent GP-SDR receivers. GP-SDR therefore reports only the host streams it can open through the installed module. It does not duplicate a stream or claim two independent tuners. A future direct-IIO multi-buffer adapter is required for simultaneous use of both pairs; both pairs share the AD9361 tuning architecture rather than acting as four unrelated tuners.

## Validation

- Full Go test suite passed.
- Universal Soapy helper compiled.
- Live network receive produced non-empty IQ using hardware AGC and a 2 MHz filter.
- Transmit modulation and selection paths passed dry-run tests; RF output was not keyed.
