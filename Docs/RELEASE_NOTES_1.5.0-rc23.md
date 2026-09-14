# GP-SDR 1.5.0-rc23

## Spectrum Analyzer

- Adds a dedicated Spectrum Analyzer workspace for continuous full-device or custom-range sweeps.
- Runs one selected receiver or every connected receiver concurrently, using each radio's own tuning and sample-rate limits.
- Shows current receiver slices plus a persistent combined peak graph or one accumulated graph per receiver.
- Retains repeated activity as searchable Mapper evidence, exports analyzer results as CSV, and queues selected peaks as focused Identify jobs.
- Keeps analyzer start and stop controls isolated from unrelated Mapper jobs.

The accumulated graph represents peak observations gathered over repeated tuning windows. It is not a claim that a receiver instantaneously captures its entire tuning range; live cards show each receiver's current hardware capture window.
