# GP-SDR 1.5.0-rc29

## SDR signal test fixtures

- Adds a native **Transmit & test lab** source selector.
- Generates deterministic CW, AM, NFM, WFM, OOK, 2-FSK, GFSK, GMSK-like, BPSK, and QPSK reference IQ.
- Stores exact payload/symbol ground truth, occupied-bandwidth estimates, EVM, impairment settings, and SHA-256 alongside every fixture.
- Adds bounded SNR, frequency offset/drift, IQ imbalance, DC offset, and clipping controls.
- Dry-run fixtures work without connected hardware; guarded RF playback continues to require a local computer, an explicit safety confirmation, and a transmit-capable HackRF or PlutoSDR.
- Synthetic fixture truth stays separate from user-confirmed over-the-air learning samples.

## Reliability

- The P25 release packager now reuses verified cached SDRTrunk/JMBE components and bounds network downloads.
- Mapper digital evidence and manual confirmation from rc28 remain included.

## Validation

- All Go unit/integration tests pass.
- Browser JavaScript type/syntax validation passes.
- Every supported fixture family produces deterministic bounded IQ in automated tests.
- A local QPSK dry run produced IQ and a matching truth manifest without an SDR or RF output.

RF transmission and independent receiver decode are deliberately not claimed by the dry-run validation.
