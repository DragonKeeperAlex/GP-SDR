# GP-SDR 1.5.0-rc30

## Expert transmit workflow

- Adds an **I know what I'm doing** setting for experienced local operators.
- Requires one explicit acknowledgement, remembers it locally, and removes the repeated per-job checkbox.
- Preserves local-only transmit control, receiver and frequency validation, emergency stop, firmware-health blocks, and the 60-second maximum job duration.

## Validation scope

- Automated coverage verifies the expert setting cannot remove backend safety guards.
- Soapy receive streaming now requests native signed 8-bit IQ, avoiding weak-signal resolution loss on PlutoSDR/AD936x receivers.
- Receive hardware checks use bounded IQ captures without routing audio to the system output.
- No over-the-air transmission is performed as part of release acceptance.
