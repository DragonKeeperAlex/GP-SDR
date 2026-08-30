# GP-SDR 1.3.1

This follow-up hardens the 1.3 Mapper overhaul for multi-receiver operation.

- Added **Use all connected receivers** to Mapper. Starting a template now
  fans it out into independent concurrent jobs, one per receiver currently
  connected and available; occupied receivers are skipped with a clear status.
- Receiver selectors now distinguish duplicate models as `HackRF One 1`,
  `HackRF One 2`, `RTL-SDR 1`, and so on while preserving stable device IDs.
- Mapper result rendering remains capped to a responsive visible page while
  exports, filters, and uploads retain the full result set.
- Added regression coverage for fan-out jobs, duplicate receiver labels, and
  the new start-all API route.

Real hardware validation remains receive-only. Two HackRF One units were
enumerated and exercised; RTL-SDR validation is performed when a supported
device is visible to the host.
