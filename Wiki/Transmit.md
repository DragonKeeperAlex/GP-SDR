# Transmit (HackRF)

The Transmit page is receive-only by default. It accepts a local PCM WAV file, converts it to bounded HackRF IQ, and offers AM, NFM, or WFM modulation.

1. Select an available HackRF, frequency, mode, duration, and TX gain.
2. Select **Dry run** to generate and validate IQ without enabling RF output. This is the recommended first test.
3. For an on-air test, connect a suitable dummy load, clear Dry run, and check the explicit safety confirmation. The job is limited to 60 seconds and can be stopped immediately.

RTL-SDR devices cannot transmit. GP-SDR blocks remote web clients from uploading audio or starting/stopping RF output. Digital voice transmission, repeaters, and unattended transmission are intentionally not implemented.

Never connect a HackRF transmitter directly to another receiver or antenna without confirming power limits, isolation, licensing, and local interference rules.
