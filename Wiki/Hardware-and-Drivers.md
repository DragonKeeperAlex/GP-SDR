# Hardware and Drivers

## Supported receiver paths

| Receiver path | Typical capabilities |
| --- | --- |
| HackRF One / PortaPack in USB mode | Wide tuning range, selectable sample rate, LNA/VGA, RF amplifier, DC/IQ correction; half-duplex hardware; receive workflows plus guarded local WAV transmission |
| RTL-SDR family | Low-cost receive-only IQ source, gain control, model-dependent direct sampling and bias tee |
| SoapySDR | Other local SDRs supported by an installed Soapy module |
| `rtl_tcp` | Remote RTL-SDR stream with reduced hardware-control visibility |

Multiple receivers can be connected simultaneously. GP-SDR assigns exactly one
capture owner to a physical device at a time, preventing the tuner, Mapper, and
P25 engine from fighting over the same USB receiver.

## Hardware page status

Each card reports the detected name, driver readiness, serial where available,
current owner/job, center frequency, sample rate, gains, amplifier/bias state,
stream health, dropped blocks, and overload or noise-only notices. Press
**Refresh** after connecting hardware or installing a component.

## Receiver and antenna lab

The Hardware page can compare multiple connected receivers in parallel. Choose
each receiver's complete nominal tuning range, an antenna-rated range, or a
custom range. GP-SDR displays the current test frequency, progress and ETA,
observed response/noise graph, best observed frequency, detection count,
overload count, and a practical recommendation for every receiver. Export CSV
before changing antennas or locations to retain comparable runs.

Use the same antenna position, cable, gain, sample rate, point count, and dwell
when comparing runs. Saved calibration is used by default. Advanced mode lets
you instead hold generic gain or HackRF LNA/VGA/amplifier settings constant.
Running a full 1 MHz–6 GHz HackRF sweep is intentionally sparse unless many test
points are selected; an antenna-range test is usually faster and more useful.

These are ambient-response measurements: they show what that receiver and
antenna observe at that place and time. They cannot measure absolute receiver
sensitivity or antenna gain because local transmitters and the noise environment
are unknown. Absolute sensitivity testing requires a calibrated RF source and
known attenuation. The tuner, Scanner, P25, and Mapper remain mutually exclusive
with this test so two features cannot silently fight over one receiver.

## HackRF setup

On a PortaPack, enter **HackRF USB mode** before refreshing GP-SDR. Start with:

- RF amplifier off
- LNA around 16–24 dB
- VGA around 16–24 dB
- DC removal on
- antenna suited to the target band

The RF amplifier can overload the front end and make every frequency look
active. Increase gain gradually while watching the noise floor and target SNR.
Higher sample rates widen instantaneous coverage but increase USB and CPU load;
they do not automatically improve a weak or overloaded signal.

## RTL-SDR setup

Only one program can normally claim an RTL-SDR at a time. Close SDR++, SDRTrunk,
`rtl_tcp`, or other SDR software before starting GP-SDR. On Linux, install the
distribution's `rtl-sdr` package and udev rules. On Windows, follow the in-app
WinUSB guide for the exact dongle interface.

## Install and How-to actions

GP-SDR bundles its own interface and P25 stack. User-space host tools, vendor
drivers, optional decoder programs, and large transcription models may remain
separate for licensing and platform reasons. Missing cards provide:

- **Install** when GP-SDR can perform a reviewed, platform-specific install.
- **How to** when a driver must come from the OS or device vendor.
- **Ignore** to keep using the app without that optional capability.

Never replace a USB driver unless the selected interface is positively
identified as the SDR.

## Remote RTL-SDR

Open **Hardware → Remote RTL-SDR**, enter the `rtl_tcp` host and port, save it,
and refresh. Remote gain, bias tee, calibration, and supported sample rates
depend on the remote server. Use a VPN for untrusted networks; do not expose an
unauthenticated `rtl_tcp` port to the public internet.

## Version 1.4 receiver behavior

A HackRF that enumerates successfully can remain selectable for reception despite a diagnostic warning. Read the warning and verify actual receive data; RF transmission remains blocked. Version 1.4.1 fixes preferred P25 receiver selection by hardware serial. Refresh and check the intended receiver after reconnecting or moving ports.

Native RTL-SDR captures reuse persistent sessions instead of reopening USB for every short scan. Refresh uses non-claiming USB enumeration on macOS. These changes do not prove long-duration USB reliability; a dongle disappearing from USB still needs physical troubleshooting.

Follow [Receiver and Antenna Lab](Receiver-and-Antenna-Lab) for a complete comparison workflow and [Multiple Receivers](Multiple-Receivers) for ownership and concurrent jobs.

---

1.4.1 baseline source: [discovery.go](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/internal/app/discovery.go), [index.html](https://github.com/DragonKeeperAlex/GP-SDR/blob/26501f8/server/web/index.html).

Current additions checked against [1.5.0-rc9](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md) and its [interface source](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/server/web/index.html).
