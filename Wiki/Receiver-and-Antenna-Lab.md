# Receiver and antenna lab

The lab compares **observed RF response** across selected receivers and frequency ranges. It does not measure geographic range, calibrated antenna gain, SWR, or absolute sensitivity. Ambient transmitters and noise vary with place and time.

## Run a comparison

1. Stop workflows occupying the selected radios and open **Hardware → Receiver & antenna lab**.
2. Check the receivers to compare. Verify their labels/serials first.
3. Choose **Each receiver’s full nominal range**, **Antenna-rated range**, or **Custom range**. Enter the corresponding minimum/maximum or start/end values in MHz.
4. Give the antenna a useful name so the result has context.
5. Choose **Test points** (24, 48, 96, 192, or 384) and **Dwell per point** (0.1, 0.18, 0.4, or 1 second). More points sample the range more densely; more dwell gives each point more observation time.
6. Leave **Capture width** on Auto per receiver initially. A rate suitable for HackRF may be unsupported by an RTL-SDR.
7. Keep **Apply saved calibration** enabled for ordinary comparisons. For controlled manual tests, switch to Advanced, disable that option, and set receiver gain or HackRF LNA/VGA/RF amp deliberately.
8. Press **Start comparison**. Inspect current frequency, progress, ETA, response/noise graphs, detection and overload counts, and the recommendation for each radio.
9. Press **Export CSV** before clearing results or changing the setup. Use **Stop** to end early; a partial run is not complete coverage.

## Compare antennas fairly

Keep location, antenna position, cable, receiver, capture width, gain, points, and dwell consistent. Run antenna A, export, change only the antenna, then repeat for B. Record the time and configuration with each export because local activity may change between runs.

A sparse full-range sweep can miss a narrow signal. Start with the antenna-rated range or a smaller band, then increase points where needed. The best observed frequency reflects received activity at that moment, not the antenna’s guaranteed best operating frequency.

If many points overload, lower gain and turn the RF amplifier off before comparing. An overloaded receiver can make a worse setup look artificially strong. For calibrated sensitivity or gain measurements, use a known RF source and calibrated attenuation in an appropriate test setup.

Related: [Hardware and Drivers](Hardware-and-Drivers), [Tuner calibration](Tuner-and-Live), [Multiple Receivers](Multiple-Receivers).

---

1.4.1 baseline source: [index.html](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/index.html).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
