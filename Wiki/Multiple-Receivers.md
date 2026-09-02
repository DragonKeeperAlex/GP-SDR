# Multiple receivers and ownership

GP-SDR supports independent work on multiple radios. A software VFO is a channel inside one captured passband; it is not another USB receiver. More software VFOs increase analysis work, while more physical radios can cover independent bands.

## Prepare the radios

1. Connect and verify each radio separately in [Tuner](Tuner-and-Live).
2. Open **Hardware → Refresh** and identify each by model and serial. Use unique RTL-SDR serials when possible; identical generic labels are not a reliable identity.
3. Stop any competing SDR program using those radios. Leave a receiver owned by another service alone unless you intend to release it.
4. Check the owner/job and health state before assigning a new workflow.

## Run two Mapper jobs

1. Open Mapper, press **New job**, select radio A, and configure its range, workflow, step, and observation time.
2. Save/start it and confirm its receiver and progress on the job card.
3. Press **New job** again, select idle radio B, and configure a different range or workflow.
4. Save/start the second job. Each has independent progress and Stop controls; **Stop all** stops all Mapper jobs.
5. Filter Combined results by receiver or job to inspect provenance. Export CSV to keep both sources with the observations.

**Use all connected receivers** applies one template to available radios; it does not automatically divide a wide range into non-overlapping slices. For a split survey, create jobs with explicit bounds. Busy radios are skipped and reported.

Within a job, **Channels at once** lets nearby targets share a capture. In rc9, Auto uses up to 512 Discovery targets, 64 Map targets, or one Identify target; the passband can reduce the actual batch. Version 1.4.1 used 16/four for Discovery/Identify. Set 1 if analysis falls behind and increase gradually after checking CPU/USB health.

## P25 alongside another job

Create a P25 profile with explicit Control/Voice assignments. On the P25 page choose **Profile assignments** to preserve that plan. Choosing a named receiver instead creates a single-receiver configuration when starting. Leave a different radio idle for Mapper or tuning, and verify both workflows’ ownership displays.

A single wideband receiver can follow control and traffic channels that fit its bandwidth; multiple radios increase potential coverage and voice capacity. They do not fix an incorrect control channel, encryption, or missing JMBE.

In 1.4.1, SDRTrunk preferred names use hardware serials while USB identities keep configuration associations. Refresh after reconnecting devices or moving USB ports, and verify the intended receiver is selected rather than assuming an old menu label is still correct.

## Handoff and troubleshooting

Stop the current owner, wait for it to release the receiver, then start the next workflow. Do not launch `rtl_test`, SDR++, or a second SDRTrunk instance while GP-SDR owns that radio. Diagnostic programs also claim devices and can create an expected temporary unavailable state.

If a receiver disappears from the operating system, verify its cable, power, and direct USB connection before changing profiles. The 1.4 release notes still leave long-duration RTL and simultaneous two-radio acceptance open; multi-receiver support does not establish reliability of every physical setup.

---

1.4.1 baseline source: [mapper.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/mapper.go), [sdrtrunk.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/sdrtrunk.go).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
