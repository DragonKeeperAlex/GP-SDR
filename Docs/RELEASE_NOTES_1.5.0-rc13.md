# GP-SDR 1.5.0-rc13

This follow-up hardens the identification repair introduced in rc12.

- Valid protocol frames now take precedence over heuristic and local-model candidates.
- Existing valid decoder frames are reconciled from the event journal at startup.
- Journal rows whose retained source files were already removed are marked unavailable once instead of failing again in every offline-analysis run.
- Mapper identity counts continue to cover the complete stored frequency set rather than only the 5,000 rows returned to the UI.
- False DSD-FME startup banners and known empty-audio captions remain excluded and requeued for clean analysis.

Strictly verified identities still require either a valid decoder frame or a geographically plausible reference match. Waveform and model classifications remain visible as candidates rather than being mislabeled as confirmed.
