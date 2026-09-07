# GP-SDR 1.5.0-rc16

This release corrects radio transcription contamination and supports Ollama inference on a private-network computer.

- Repeated Whisper sound-effect annotations such as tires, engines, gunfire, helicopters, static, clicking, buzzing, and similar non-speech captions are removed.
- Spoken text on the same line as a noise annotation is preserved.
- Historical contaminated transcripts are cleaned at startup and eligible source captures are requeued for analysis.
- Ollama endpoints may use localhost or a private-network IPv4/IPv6 address over HTTP or HTTPS; public destinations and embedded URL credentials remain blocked.
- Settings now clearly labels the Ollama server field and saves with a connection test.
- Ollama reasoning output is disabled for schema-constrained classification, fixing empty responses from Qwen 3.5 models.
- Raw IQ and audio remain local; only bounded DSP, decoder, transcript, frequency, and location metadata is sent to the selected Ollama endpoint.
