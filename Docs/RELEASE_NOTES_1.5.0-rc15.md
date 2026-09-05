# GP-SDR 1.5.0-rc15

This release keeps offline analysis consistent with automatic storage cleanup.

- When a cap or retention cleanup removes an IQ/audio payload, its event journal entry is reconciled immediately while GP-SDR remains open.
- Events with no remaining source media are marked unavailable and removed from the pending Analyze queue.
- Events that still have another usable payload retain that payload and remain eligible for analysis.
- Missing-file failures no longer wait for an application restart to be resolved.

Validation includes the complete Go test suite, race detector, static analysis, and a regression covering runtime removal of pending IQ evidence.
