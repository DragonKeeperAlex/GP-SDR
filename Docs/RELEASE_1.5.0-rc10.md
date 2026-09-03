# GP-SDR 1.5.0-rc10

- New Explore workspace: hourly recorded-activity heatmap, daily timeline, offline collection/reference-area plot, frequency/area grouping, date/frequency/text/location/mode filters, sorting, paging, bookmarks, tags, and saved views.
- Inspect contributing captures, decoder messages, transcripts, model summaries, and geographically applicable reference candidates.
- Conservative silence/static transcription gate and cleanup of standalone non-speech captions. Weak speech remains eligible for transcription.
- Reject invalid stored-analysis concurrency before starting a run; clear stale active-stage indicators when analysis ends and preserve canceled captures for retry.

Explore is an event-history viewer, not an occupancy measurement or transmitter locator. See [Explore documentation](../Wiki/Explore.md) for coverage and evidence limitations. Existing recordings and transcripts are preserved.

Validation covers automated application/race tests and an isolated UI check; this release does not represent new live-RF decoder acceptance tests.

The installed whisper.cpp/base.en model also passed an opt-in speech-file integration test, transcribing a generated spoken radio-test sentence correctly. Silence, stationary static, and quiet modulated waveform gating have regression tests. This is not a field test of weak radio speech.
