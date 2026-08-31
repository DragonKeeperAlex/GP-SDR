# GP-SDR 1.4.1

Includes the compact console and receiver fixes from [1.4.0](RELEASE_NOTES_1.4.0.md).

The final hardware log review found that SDRTrunk's preferred receiver names
are not the same as its USB configuration IDs. HackRF selection now uses its
formatted hardware serial; RTL-SDR uses the EEPROM serial and known tuner
label. USB bus/port identities remain the keys for saved settings and disabling
unassigned radios. This removes silent fallback to another available tuner.

A fresh physical HackRF run at 774.89375 MHz decoded valid EBRCS control
frames (NAC 0x1F2, WACN 0xBEE00, system 0x1F1) without the previous
preferred-tuner fallback messages. The Go unit, race, and vet checks pass.
The earlier HackRF voice recordings remain separate audio evidence; this patch
does not claim complete protocol coverage or resolution of the RTL USB dropout.
