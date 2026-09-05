# GP-SDR 1.5.0-rc12

This release corrects identification accounting and evidence integrity found in a large completed offline-analysis run.

- Mapper totals now cover the complete frequency database. The bounded UI payload prioritizes verified, active, and analyzed records instead of silently returning only the highest 5,000 frequencies.
- The Mapper overview separates strictly verified identifications from analyzed frequencies and unverified candidates.
- DSD-FME startup/status banners no longer count as received protocol frames. Only output containing received frame/sync/voice/data metadata can verify a digital protocol.
- On startup, known historical banner evidence is downgraded and affected captures are queued for reanalysis. Known standalone Whisper noise captions are cleared and queued again without deleting their source recordings.
- Parallel stored analysis no longer starts a shared 90-second deadline while work waits behind serialized decoder, Whisper, or Ollama queues. Each actual component retains its own bounded timeout.
- Local-model summaries are generated from the validated structured result so prose cannot contradict the stored family/modulation fields.
- Additional common standalone sound-effect hallucinations are suppressed for future transcripts.

Validation includes the complete Go test suite, race detector, static analysis, JavaScript syntax checks, installed local-model integration, installed optional-decoder bridge smoke tests, and isolated UI/API checks. Protocol startup and a model response remain insufficient evidence of live RF decode; valid frame metadata or geographically applicable authoritative data is required for “Successfully identified.”

The existing data audit found that most collected jobs lacked a saved observation location, and the downloaded statewide California banks do not include a precise reference area. They remain useful candidates but cannot safely verify a same-frequency local identity.
