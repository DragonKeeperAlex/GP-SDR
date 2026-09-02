# Tuner, Spectrum, Waterfall, and Live Audio

## Hardware center and Listen VFO

GP-SDR separates two frequencies:

- **Hardware center** is where the SDR's IQ passband is physically centered.
- **Listen VFO** is the software-demodulated frequency inside that passband.

Leaving Hardware center blank lets GP-SDR place the VFO away from the receiver's
DC spike. You can center a HackRF at 450 MHz while listening at 452 MHz without
retuning the hardware, provided 452 MHz remains inside the captured bandwidth.
Clicking the spectrum or waterfall moves the software VFO when possible.

## Demodulation controls

- **Auto** analyzes local IQ and selects AM, NFM, or WFM.
- **AM** is typical for civil aviation and broadcast AM.
- **NFM** is typical for land-mobile, GMRS, MURS, marine, and many conventional
  voice channels.
- **WFM** is used for broadcast FM audio.
- **Bandwidth** changes the software channel filter, not the SDR's entire
  capture width.
- **Sample rate** controls the instantaneous RF width and USB/CPU load.

## Gain and front-end controls

Changes apply immediately.

- **Gain** is the general receiver setting used by supported devices.
- **LNA** and **VGA** expose HackRF gain stages.
- **RF amp** enables the HackRF input amplifier; use it sparingly.
- **Antenna power** controls supported bias-tee output. Enable it only for an
  antenna device designed for the receiver's voltage.
- **AGC** automatically adjusts audio level; it cannot repair RF overload.

If the display is solid with activity, turn the RF amplifier off and reduce
gain before changing detection thresholds.

## DC and I/Q correction

- **DC removal** suppresses the center spike common to zero-IF receivers.
- **Q balance** compensates I/Q amplitude mismatch.
- **IQ phase** compensates quadrature phase error.
- **Swap I/Q** corrects reversed spectral orientation on affected sources.
- **PPM correction** compensates oscillator frequency error.
- **Apply saved calibration** loads settings saved for the selected device.

Use **Auto calibrate** only on a strong, known reference. Review its result,
then save it per device. Poor reference selection can make tuning worse.

## Signal, squelch, and noise reduction

The signal indicator compares channel energy with the measured noise floor.
Squelch is a margin above that floor, not an absolute guarantee of speech.

- Enable **Open squelch** while diagnosing audio.
- Raise squelch until idle noise closes without clipping the start of calls.
- **Voice** noise reduction is a low-latency cleanup for speech.
- **Strong** removes more stationary noise but can make marginal speech sound
  processed; use Off when evaluating raw decoder or RF quality.

## Spectrum and waterfall

The spectrum shows current power over frequency; the waterfall adds time with
newest samples at the top. Hover for frequency/power and click to move the VFO.
Settings provides frame rate, quality/FFT detail, smoothing, peak hold, and
display floor/ceiling controls. Reduce these before lowering receiver sample rate
when the interface is sluggish but audio is otherwise stable.

## Audio and mixer

The top bar controls whole-app mute and volume. Live and decoder mixers add
mute/solo; analog channel rows also offer volume and Advanced-mode pan. P25
talkgroup rows use system output and do not have those sliders. Channel priority
settings control ducking.
If spectrum activity is present but audio is silent, check master mute, the
selected row, solo state, squelch, mode, output device, and P25 JMBE status.

## Digital modes

The **Mode / decoder** control on Tuner and **Mode** on Live include DMR,
conventional P25, NXDN, D-STAR, YSF, M17, POCSAG/FLEX, and ACARS. Digital voice
uses a 12.5 kHz channel by default. GP-SDR sends discriminator audio to the
installed decoder and returns decoded unencrypted speech through the app audio
controls. A valid decoder frame, rather than RF energy alone, is required before
Activity identifies a protocol.

P25 trunk following remains in the dedicated P25/SDRTrunk workspace. The Tuner
P25 choice is for conventional single-frequency experiments.

## Tune within a fixed capture

1. Select an idle receiver and enter the desired **Listen VFO** frequency.
2. Leave **Hardware center** blank for automatic offset placement, or enter a center whose passband contains the desired channel.
3. Start reception, then enable **Lock center · software VFO** when you want to keep the hardware tuned while listening to other channels within that window.
4. Click the spectrum/waterfall to move the software VFO. Keep the complete channel bandwidth inside the capture, not just its center frequency.
5. Use **Recent frequencies** to return to a previous tuning. If the next signal is outside the capture window, unlock/change the center as needed.

If audio is distorted, verify mode and channel bandwidth, then reduce front-end gain before applying stronger audio cleanup. CTCSS detection can appear in Activity for analog signals; DCS decoding is not implemented.

---

1.4.1 baseline source: [index.html](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/index.html), [app.js](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/app.js).

## rc9 compact tuning controls

In 1.5.0-rc9, click above or below a digit in the frequency readout to step that digit. The VFO row labels are **Step**, **Recent**, and **Lock hardware center**. Use **Save** beside Start/Stop to save the current frequency as a reusable profile. **Audio level auto** labels audio AGC; **Auto calibration** applies saved calibration. These controls do not make RF overload disappear or establish calibrated receiver sensitivity.

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
