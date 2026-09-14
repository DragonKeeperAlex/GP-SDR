# GP-SDR 1.5.0-rc26

- Replaces the Mapper-backed analyzer with a dedicated, non-persistent live spectrum service.
- Captures receiver slices concurrently and sweeps continuously without creating Mapper jobs, events, IQ files, recordings, or identification work.
- Adds wheel zoom, drag-to-pan, exact hover frequency/level, full-range reset, clear, and CSV export.
- Keeps one combined spectrum for one or many connected receivers.
