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
waterfall history controls. Reduce these before lowering receiver sample rate
when the interface is sluggish but audio is otherwise stable.

## Audio and mixer

The top bar controls whole-app mute and volume. Live and decoder mixers add
per-channel or per-talkgroup mute, solo, volume, pan, and priority ducking.
If spectrum activity is present but audio is silent, check master mute, the
selected row, solo state, squelch, mode, output device, and P25 JMBE status.
