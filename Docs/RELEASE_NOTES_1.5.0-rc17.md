# GP-SDR 1.5.0-rc17

- Analyze now absorbs new stored captures while Mapper is running instead of using only its startup queue snapshot.
- Local intelligence can request an explicit 8K through 256K Ollama context window; Automatic remains the safe default.
- Local-model evidence now includes bounded matching entries from imported local and RadioReference profiles, with RadioReference matches filtered by the recorded receive location.
- Settings can benchmark selected installed Ollama models against identical grounded radio-evidence cases, reporting structured-output reliability, grounding, and latency.
- The new RF Monitor page shows an independent live spectrum and waterfall for every receiver currently producing IQ.
- The local model inventory continues to exclude embedding-only models.

Raw IQ and audio stay in GP-SDR. Only bounded metadata and matching local-reference evidence are sent to the configured Ollama server.
