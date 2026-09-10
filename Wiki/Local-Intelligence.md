# Local signal intelligence and confirmed examples

This optional layer summarizes existing DSP, decoder, transcript, frequency, location, and bounded local-reference evidence using an Ollama model. It does not replace protocol decoding or prove a transmitter’s identity.

## Set up the local model

1. Install/start Ollama on the GP-SDR host. Use the component’s installation guidance where offered.
2. Download the documented default model once:

```bash
ollama pull qwen2.5:1.5b
```

3. Open **Settings → Local intelligence → Local signal intelligence** on the host.
4. Enable **Analyze Mapper evidence locally**. Use `http://127.0.0.1:11434` for Ollama on the Mac, or a private-LAN address such as `http://192.168.1.54:11434` for a trusted compute server. Start with a small model, **Automatic** context, and Minimum confidence at **55%**.
5. Press **Save**. Confirm the service is reachable, then analyze a real capture and inspect its evidence. Ready checks the service response; it does not guarantee that the named model has been downloaded or can generate successfully.
6. Use [Analyze](Analyze-and-Schedule) for queued work or Live analysis timing for new Mapper jobs.

The endpoint must be localhost or a private-network address; public Internet model endpoints are rejected. Configuration changes are restricted to the GP-SDR host. GP-SDR sends bounded text metadata, including available transcript, decoder fields, capture location, and nearby imported channel matches. Raw audio/IQ is not sent to the model endpoint. RadioReference candidates are admitted only when the capture has location evidence and the reference area passes GP-SDR's distance filter.

## Performance and confidence

Lightweight, Balanced, and Deep analysis set default context budgets of 2K, 4K, and 8K. The explicit Context control can request 8K through 256K, but long context consumes much more model memory and normally does not improve GP-SDR's short evidence payloads. It does not download a different model automatically.

Use **Models to compare** and **Run benchmark** to check installed generation models against the same five grounded radio-evidence cases. The result measures structured-output compatibility, evidence grounding, and latency on that server. It is not an RF-identification accuracy claim and does not replace real captures or decoder validation.

Minimum confidence gates the returned label; a below-threshold answer becomes Unknown. Model modulation that conflicts with stronger measured DSP evidence is corrected and confidence limited. Even a high-confidence summary is not valid protocol frames or an authoritative station identification.

One model request runs at a time to limit memory. Multiple Analyze workers can overlap other stages but still queue for model inference. A long model stage is not proof that the UI has frozen; inspect the active stages/log and service status.

## Add a confirmed example

1. Find a real event under **Activity → Timeline** and independently verify its modulation/protocol from decoded frames or other reliable evidence.
2. Press the event’s **checkmark** to open **Confirm signal example**.
3. Enter **Confirmed modulation**, **Confirmed protocol or type**, and an **Evidence note**.
4. Optionally select **Keep references to this event’s IQ/audio**. This stores references, not a separate archival copy or a guarantee against retention cleanup.
5. Press **Add confirmed sample**. The Settings card displays the learned sample count.

Simulated events cannot be learned. Unconfirmed Mapper guesses are not automatically added. Confirming the same event updates its sample. Relevant confirmed examples can be retrieved immediately for later model prompts; this does not retrain model weights.

## Export the training set

Use **Export confirmed training set** in Settings after adding examples. The JSONL contains metadata, transcript/decoder evidence, labels, and notes for later training outside GP-SDR. Review it before sharing: transcripts and notes can contain information you do not want to publish. Exporting does not itself train a classifier.

## Troubleshooting

- **Runtime needed:** start Ollama on this host and check the local service address.
- **Ready but no summary:** verify the exact model is installed, inspect the analysis stages, and confirm there is usable evidence. Optional-stage failures may not increment the overall failed-file count.
- **Remote address rejected:** use localhost or an RFC1918/private-LAN address; public endpoints are intentionally blocked.
- **Unknown:** verify RF/decoder evidence before lowering the confidence threshold.
- **High memory use or slow processing:** use Lightweight, a smaller model, and fewer parallel Analyze groups.

Source: [local model](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/local_ai.go), [confirmed examples and JSONL](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/local_learning.go).
