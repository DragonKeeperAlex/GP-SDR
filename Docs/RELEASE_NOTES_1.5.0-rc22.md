# GP-SDR 1.5.0-rc22

- Removes the legacy 20 MS/s limit for PlutoSDR and compatible SoapySDR receivers.
- Tuner, Mapper, Band Monitor, and hardware characterization can select 24, 30.72, 40, 50, or 61.44 MS/s when the attached driver reports support.
- Keeps HackRF at its supported 20 MS/s ceiling and RTL-SDR at its device limit.
- Reports the distinction between capture sample rate and analog RF filter bandwidth. The tested Tezuka firmware accepts 61.44 MS/s but reports a maximum 10 MHz analog filter.
- Shows explicit original-IQ storage estimates: about 442 GB/hour at 61.44 MS/s in GP-SDR's CS8 archive format.
