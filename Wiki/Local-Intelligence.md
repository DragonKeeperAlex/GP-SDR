# Local signal intelligence and confirmed examples

**Available in 1.5.0-rc9.** This optional layer summarizes existing DSP, decoder, transcript, frequency, and location evidence using a local Ollama model. It does not replace protocol decoding or prove a transmitter’s identity.

## Set up the local model

1. Install/start Ollama on the GP-SDR host. Use the component’s installation guidance where offered.
2. Download the documented default model once:

```bash
ollama pull qwen2.5:1.5b
```

3. Open **Settings → Local intelligence → Local signal intelligence** on the host.
4. Enable **Analyze Mapper evidence locally**. Leave Local service at `http://127.0.0.1:11434`, Model at `qwen2.5:1.5b`, Performance at **Lightweight**, and Minimum confidence at **55%** to begin.
5. Press **Save**. Confirm the service is reachable, then analyze a real capture and inspect its evidence. Ready checks the service response; it does not guarantee that the named model has been downloaded or can generate successfully.
6. Use [Analyze](Analyze-and-Schedule) for queued work or Live analysis timing for new Mapper jobs.

The endpoint must be localhost or a loopback IP over HTTP; a LAN/remote Ollama server is rejected. Configuration changes are restricted to the GP-SDR host. GP-SDR sends metadata, including available transcript, decoder fields, and location, to that local service; raw audio/IQ is not sent to the model endpoint. Model downloads themselves need network access.

## Performance and confidence

Lightweight, Balanced, and Deep analysis set increasing model context budgets (2,048, 4,096, and 8,192). They do not download a different model automatically. Enter a model name only after making it available in Ollama. Start small, then increase if memory and response time permit.

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
- **Remote address rejected:** use localhost on the host; there is no remote-model mode here.
- **Unknown:** verify RF/decoder evidence before lowering the confidence threshold.
- **High memory use or slow processing:** use Lightweight, a smaller model, and fewer parallel Analyze groups.

Source: [local model](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/local_ai.go), [confirmed examples and JSONL](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/internal/app/local_learning.go).
