# GP-SDR 1.0.13

## Local signal intelligence

- Offline IQ classification identifies AM, narrow FM, wide FM, constant-envelope digital candidates, and weak carriers with confidence and measured evidence.
- The native tuner has an Auto mode and shows the current local classification.
- Offline whisper.cpp setup now installs a pinned English model without an API key.
- Transcripts extract strict callsign candidates from ordinary spelling and NATO phonetics.
- Mapper records, CSV exports, event history, and decoder pages retain analysis, transcripts, callsigns, and decoder evidence.

## Receiver and decoding improvements

- Multiple assigned SDRs divide conventional channels and scan ranges into non-overlapping receiver workloads; SDRTrunk continues to allocate multiple P25 tuners.
- Native and Soapy views of the same physical HackRF are deduplicated, and hardware refreshes preserve the active inventory instead of probing a radio while it is receiving.
- Measured clipping telemetry replaces ambiguous overload state and reports the clipped-sample percentage.
- Separate hardware center and software VFO tuning, immediate RF/IQ controls, master audio, stereo pan, priority ducking, and Mapper ETA remain available in the native app.
- rtl_433, multimon-ng, and DSD-FME have executable file/audio bridges; a candidate is promoted to confirmed only after a decoder returns a real message or frame.
- SDRTrunk P25 calls can be recorded, attached to GP-SDR events, and passed to offline transcription when enabled.
- P25 events now retain structured system, talkgroup, source-radio, and encryption fields. The isolated HackRF P25 runtime forces the RF amplifier off to avoid overload-driven robotic voice.
- The P25 mixer shows the currently decoded control channel and orders talkgroups by most recent activity or total calls received, with active calls pinned first.
- HackRF P25 capture now defaults to a tested 10 MS/s, automatically retries at 5 MS/s after a decoder transport failure, and exposes Auto/5/8/10/20 MS/s controls in both the P25 workspace and profile editor.
- Mapper has an independent Auto/2/3.2/8/10/20 MS/s capture-width control; live HackRF scanning now captures at 8 MS/s before digitally narrowing each channel, while the tuner retains manual control through 20 MS/s.
- Missing optional components appear in a dismissible startup list with automatic Install actions where supported and upstream instructions otherwise.
- A tested macOS helper installs DSD-FME, dump1090, multimon-ng, acarsdec, and AIS-catcher from their maintained upstream sources, including the dependencies needed by GP-SDR's decoder bridges.

## Evidence and storage

- Unknown activity can retain a bounded two-second IQ capture with a metadata sidecar.
- The configured recording-retention period now prunes only dated GP-SDR recording and IQ folders.
- RadioReference and other key-based integrations are unchanged and remain optional.
- Universal macOS releases now include a drag-to-Applications DMG in addition to ZIP packages.
