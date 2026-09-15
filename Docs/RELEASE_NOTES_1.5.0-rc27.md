# GP-SDR 1.5.0-rc27

- Adds a dedicated low-latency analog FPV receiver page for HackRF and PlutoSDR.
- Includes common 5.8 GHz Raceband and Band A presets, manual tuning, NTSC/PAL selection, receiver gain controls, DC removal, live frame status, and fullscreen video.
- Uses the separately installed GPL-licensed FPV Viewer decoder backend without incorporating its code into GP-SDR's MIT-licensed source.
- Keeps FPV receive independent from Mapper, spectrum sweeps, and persistent IQ capture.
