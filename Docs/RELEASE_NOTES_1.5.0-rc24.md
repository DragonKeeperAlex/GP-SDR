# GP-SDR 1.5.0-rc24

## Bandwidth-aware discovery

- Separates scan spacing from the RF bandwidth measured and retained around each candidate.
- Automatic Mapper and Spectrum Analyzer discovery now measures US broadcast FM centers with a 200 kHz channel instead of truncating them to a 12.5 kHz scan step.
- Keeps off-center fine-grid probes narrow to prevent one wide FM station from being logged as many duplicate channels.
- Uses appropriate automatic widths for AM broadcast, civil airband, narrow voice/digital channels, and ADS-B.
- Adds an explicit Spectrum Analyzer signal-width selector for narrow, broadcast-FM, wide-unknown, and wide-data surveys.

The receiver sample rate remains the width of the instantaneous hardware capture. Signal width controls the per-candidate FFT, demodulation, and retained IQ window inside that capture.
