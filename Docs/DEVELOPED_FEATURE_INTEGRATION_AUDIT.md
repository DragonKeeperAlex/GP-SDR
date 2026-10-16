# Developed-feature integration audit — October 16, 2026

This pass compares the active Pi-first development branch, local unpublished
changes and named remote feature branches. It is not an exhaustive archaeology
of every historical commit. Preserve unrelated unfinished work rather than
merging prototypes simply because they exist.

## Integrated and verified in this batch

| Feature | Decision and evidence |
| --- | --- |
| RTL streaming recovery | Close the per-window rtl_tcp client while keeping helper/USB ownership; stream/capture regressions and full Go suite pass. |
| Pluto transport selection | Prefer a usable network URI and preserve device transport choices; discovery regressions pass. |
| Mapper identity reconciliation | Model/band guesses remain candidates; measured evidence can clear stale digital-voice guesses, while decoder proof is retained. Store/Mapper regressions pass. |
| Home-analysis fallback | Existing live/deferred paths queue evidence for the configured home model when local synthesis fails. Queue tests and live historical review exercised. |
| FM audio limiting | Apply the existing soft discriminator limiter below full PCM rails. DSP tests pass; this cannot repair receiver ADC overload. |
| Receiver/decoder settings | Forward shared RTL P25 gain, retain per-receiver overrides and recognize supported alternate decoder tools. Integration tests pass. |
| P25 recording boundaries | Recorder-only DRAIN/DROP control events, bounded queue, gap warnings, interrupted spool revocation and no empty PCM recordings. Pi/Windows fixtures pass. |
| Windows deferred worker | Separate filesystem-capacity implementation fixes native build; eight checked raw IQ copies processed and analyses imported without replacing Pi media paths. |
| Pi operational helpers | Add missing source for existing kiosk, display-session, power and USB-rescan service contracts. Exact-action sudoers parse successfully; power actions were not exercised. |
| USB recovery policy | One usable HackRF plus RTL; Ethernet Pluto allowed; excluded HackRF cannot satisfy presence; defer probes during active/unknown app state. Presence-only fixtures and live check pass. |
| Small Pi status screen | Select ILI9341 by identity, retain deployed rotation, report boot/data space, and repair the System-page return. Compiled and service restarted; physical appearance still requires observation. |

## Already present; retain the newer implementation

The two unpublished commits on `origin/feature/receiver-characterization`
(`d3f2674`, `4fa9521`) contain receiver characterization, Mapper eligibility,
verified identities and bounded capture storage. Corresponding functionality is
already in the active implementation. Do not cherry-pick the old branch wholesale.
The live-analysis/context-benchmark and Mapper-sheet-sync remote branches have
no unseen commits relative to the current branch.

## Preserve separately; do not promote yet

- `origin/feature/ipados-mobile` (`43210b9`, `e84cb4b`): receive-only prototype
  and installable iPad project. Keep deferred under the Pi-first remediation plan;
  an installable project is not device acceptance.
- Local macOS launcher changes: preserve for a separate review and Mac-specific
  checks. They do not improve the current Pi field path.
- Local release packaging, Wiki and release-document edits: preserve separately;
  a public release needs its own coherent version/artifact review. They are not
  evidence that remaining RF or physical gates passed.
- Future customizable pages/tiles/themes: keep planned for later, as requested.
  Browser/box display rules already exist and remain distinct.

## Next achievable gates, in order

1. Add authoritative call identity at the decoder/audio boundary and verify
   encryption changes and back-to-back/overlapping call attribution. Current
   untagged UDP PCM and polled metadata cannot establish sample-level identity.
2. Repeat bounded real-RF tests with the large antenna on each receiver, then
   exercise concurrent traffic. Keep HackRF at least 8 MS/s and measure Pi load,
   storage latency and gaps. Current three-stream fixtures are software evidence.
3. Review recorded audio/transcript usefulness and field operation on the 10-inch
   touchscreen and external browser. Do not infer this from API/static tests.
4. Complete deliberate interruption/power/storage acceptance with backups and
   recovery checks. No unattended power-cycle acceptance is claimed.
5. Reconcile group 4 against the master backlog, then run the requested GPT-6
   Astra full review. Broader UI redesign follows its own agreed scope.

Private IQ/audio/transcripts, queue manifests, host credentials and source-data
backups remain outside Git. The raw PC batch preserved originals and imported
analysis fields only; its output does not establish digital-frame decoding or
transcript correctness. A separate 256-group metadata review is resumable and
was still active during this audit.
