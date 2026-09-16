# GP-SDR 1.5.0-rc32 feature-gap and test-preparation report

Updated September 15, 2026. This report distinguishes source presence, process
startup, real receiver operation, and successful over-the-air decoding. A menu,
running decoder process, or visible RF energy is not counted as a decoded signal.

## Release blockers

| Priority | Item | Current evidence | Completion gate |
| --- | --- | --- | --- |
| P0 | Publish rc31 and rc32 | Both commits, tags, release packages, and checksums exist locally. GitHub authentication timed out. | Push `main` and both tags, upload every checksum-listed asset, publish the Wiki, and verify Actions and downloads. |
| P0 | RTL-SDR USB stability | A 48 MB capture at 2.4 MS/s completed, then Linux logged USB errors `-110` and `-62` and could no longer enumerate the dongle. The Pi was not undervolted or throttled. | Replug or power-cycle the exact hub port; run direct-port and powered-hub 30-minute captures; record kernel errors, dropped samples, temperature, and post-test enumeration. Replace the cable/hub/dongle if the failure follows it. |
| P0 | Current P25 RF acceptance | HackRF starts SDRTrunk at 10 MS/s. Pluto starts OP25 through its real Soapy URI. The short rc32 checks searched the configured EBRCS channels but did not lock. RTL testing was blocked by its USB disappearance. | Obtain valid control-channel identity and grants on HackRF, RTL-SDR, and Pluto independently; record receiver, rate, gain, frequency error, NAC/WACN/SysID, grant count, and at least one intelligible unencrypted call where traffic permits. |
| P0 | DSD-FME on the Pi | Other optional decoder executables are installed. DSD-FME requires MBE setup and an upstream patent-notice acknowledgement. | User reviews the upstream notice; build/install MBE and DSD-FME; run parser fixtures and a real DMR or conventional P25 signal. |

## Implemented but still needing hardware or field acceptance

- **Long-duration multi-radio operation:** soak HackRF, RTL-SDR, and Pluto
  together for at least eight hours with simultaneous Mapper/P25/tuner work.
  Pass only with stable device identities, bounded memory/storage growth, no USB
  reset loop, and successful release/reacquisition of every receiver.
- **P25 voice quality and trunk following:** test Phase 1 and Phase 2, single-
  receiver and separate control/voice assignments, talkgroup mute/solo/order,
  recording, JMBE audio, encryption exclusion, reconnect, and control-channel
  rotation. Run on macOS and Linux; Windows still needs a physical host pass.
- **Pluto P25:** rc32 proves OP25 process and tuner startup, correct generated
  files, and honest searching status. It still needs a real decoded control
  channel, grants, traffic audio, USB-versus-Ethernet comparison, and a thermal
  soak.
- **Optional decoder bridges:** `rtl_433`, `dump1090-fa`, `multimon-ng`,
  `acarsdec`, and `AIS-catcher` are installed on the Pi and have parser/unit
  coverage. Each still needs recorded real-frame evidence through the actual
  GP-SDR UI/API and Mapper deferred-analysis path.
- **Mapper identification quality:** run known analog, P25, DMR, ADS-B, pager,
  sensor, ACARS, and AIS targets. Verify that noise remains a candidate rather
  than a successful identification, Identify increments history correctly,
  rejected IQ follows its deletion/quarantine policy, and results survive media
  cleanup and restart.
- **Local intelligence and transcription:** validate four model presets against
  labeled real captures, remote Ollama interruption/recovery, weak speech,
  static, silence, and non-speech audio. Measure false-positive and abstention
  rates; do not use LLM text as protocol proof.
- **Tuner, Band Monitor, and audio:** repeat end-to-end AM/NFM/WFM tests on each
  receiver, including click-to-tune/software VFO, DC removal, IQ correction,
  AGC/manual gain, per-device controls, squelch, CTCSS, master mute/volume,
  recording, and output-device selection.
- **Spectrum Analyzer:** run full-range and narrow sweeps with one and multiple
  receivers, zoom/hover/max-hold/reset/CSV, device disconnect, and bounded CPU.
  Compare peaks with a known working receiver app; Analyzer must not create
  Mapper jobs or retained IQ.
- **Analog FPV:** source and launch tests exist, but a powered NTSC/PAL
  transmitter has not provided live frame, latency, or picture-quality
  acceptance. Digital FPV is not implemented.
- **Storage and recovery:** exercise every capture policy, cap, quarantine,
  post-analysis deletion, result preservation, removable/offline media, crash
  repair, and NAS outage/recovery. Confirm checksum before deleting a local file
  after offload.
- **Remote/mobile web UI:** test phone/tablet layouts, authenticated LAN access,
  reconnect, all primary controls, spectrum refresh rates, and accessibility.

## Partially implemented or missing features

| Priority | Feature | Work needed |
| --- | --- | --- |
| P1 | DCS subtone decoding | Add a decoder with confidence and age metadata, expose DCS beside CTCSS in Band Monitor/results, and test generated vectors plus real radios. |
| P1 | Explorer at large scale | Add incremental aggregation beyond the current loaded-event window, true checked-versus-unobserved occupancy, receiver/job filters, map time playback, and drill-down from cells/points to evidence. |
| P1 | Automatic multi-SDR range partitioning | `Use all connected receivers` currently clones a template rather than dividing a wide range. Add deterministic non-overlapping partitioning, capability-aware bounds, failover, and a visible assignment preview. |
| P1 | Direct Pluto dual-channel IIO | The current upstream Soapy module exposes one RX and one TX even with 2R2T firmware. A direct-IIO multi-buffer adapter is needed to expose simultaneous RX1/RX2 safely; the channels share AD9361 tuning constraints. |
| P1 | Android release candidate | The source preview has HackRF USB and RTL-TCP foundations. Add direct RTL-SDR USB, native audio/DSP acceptance, storage-access-framework SD-card testing, P25 strategy, signed APK/AAB packaging, and physical-device performance/thermal tests. |
| P1 | iOS/iPadOS release candidate | No finished iOS target is shipped from this checkout. Integrate the prototype, resolve DriverKit entitlement/signing, validate physical RTL/HackRF access, storage, audio, DSP, and P25 feasibility. Do not expose nonfunctional controls. |
| P2 | Secure credentials on Linux/Windows | Add OS credential-store editing for RadioReference and persistent remote credentials; migrate without writing secrets into profiles, logs, or exports. |
| P2 | Signed update channel | Add signed manifests, downgrade/replay protection, rollback, notarized macOS packages, and signed Windows installer/update packages. |
| P2 | Microphone and standardized digital transmit | Current transmit supports bounded guarded analog WAV/fixtures. Microphone input, digital voice/data modes, repeater behavior, unattended transmit, and image/fax-style transfer are not implemented. Each requires protocol compliance, spectral validation, authorization controls, and dummy-load/attenuated bench tests. |
| P2 | Digital FPV/video | Only conventional analog NTSC/PAL is in scope today. A digital implementation requires a specific air protocol and decoder; generic 5.8 GHz RF reception is insufficient. |
| P2 | Calibrated receiver/antenna measurements | Characterization compares observed ambient response; it does not measure calibrated sensitivity, antenna gain, SWR, or geographic range. Add a controlled signal source/attenuator workflow before presenting absolute measurements. |

## Packaging and platform tests still open

- Open and smoke-test the packaged rc32 application, not only source builds, on
  Intel macOS, Apple Silicon macOS, Windows 10/11, Linux AMD64, and Linux ARM64.
- Verify install, upgrade with preserved data, rollback, uninstall, startup at
  boot/login, driver guidance, firewall instructions, updater behavior, and
  clean-machine missing-component prompts.
- Verify all bundled licenses/source offers and the live Wiki match the shipped
  packages. OP25 itself remains an optional Linux installation; GP-SDR ships its
  launcher and credits but not the OP25 native stack.
- Public macOS notarization and Windows code signing remain unavailable until
  suitable signing credentials are supplied.

## Prepared execution order

### Phase A: unblock and baseline

1. Publish rc31/rc32 and Wiki after GitHub authorization.
2. Re-enumerate the RTL-SDR and complete the direct-versus-hub stability matrix.
3. Preserve a receiver inventory: serial, USB path, antenna, firmware, driver,
   supported rate/range, and known-good 98.1 MHz capture for every radio.
4. Review the MBE notice, then finish DSD-FME only if accepted.

### Phase B: decoding acceptance

1. Lock one known P25 system separately with RTL-SDR, HackRF, and Pluto.
2. Run Phase 1/2 control/grant/audio and multi-radio soak matrices.
3. Capture one genuine frame set for every optional decoder available locally.
4. Feed the labeled captures through live and deferred Mapper analysis, then
   record precision, false positives, abstentions, runtime, and storage cost.

### Phase C: high-value implementation

1. DCS decode.
2. Automatic multi-SDR range partitioning and failover.
3. Scalable Explorer aggregation and occupancy accounting.
4. Direct-IIO Pluto RX1/RX2 feasibility prototype.
5. Android RC, followed by iPadOS only after entitlement feasibility is proven.

### Phase D: distribution hardening

1. Clean-machine package matrix and long soaks.
2. Secure credential stores and signed update manifests.
3. macOS notarization and Windows signing when credentials are available.
4. Publish only after package, hardware, decoder-evidence, documentation, and
   rollback gates all pass.

## Actions requiring the user

1. Approve GitHub Mobile sudo-mode authorization so releases can be published.
2. Physically replug or power-cycle the RTL-SDR/hub port; swap cable/port if it
   disappears again.
3. Review and explicitly acknowledge the MBE/DSD-FME upstream patent notice
   before that codec dependency is built.
4. Provide signing credentials only when notarized macOS/Windows distribution
   is desired; they are not required for continued source and hardware work.
5. Supply an authorized, active transmitter or suitable local signals for FPV,
   DMR, and other decoder field acceptance when ambient traffic is unavailable.
