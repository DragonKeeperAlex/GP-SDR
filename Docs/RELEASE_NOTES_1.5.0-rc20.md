# GP-SDR 1.5.0-rc20

This release replaces the Mapper Overview's single rotating receiver view with
a combined live monitor. Every running job now has a persistent workflow,
receiver, batch, channel, rate, center-frequency, spectrum, and waterfall view.

PlutoSDR and compatible AD936x clone boards are supported for receive through
libiio and SoapyPlutoSDR. GP-SDR preserves the discovered USB or network URI,
uses device-reported tuning limits when available, and exposes the receiver to
Tuner, Band Monitor, calibration, RF Monitor, and concurrent Mapper jobs.

The receive capture ceiling remains 20 MHz. Some firmware reports higher sample
transport rates or expanded tuning ranges; GP-SDR labels those as driver-reported
capabilities rather than treating them as guaranteed clean RF bandwidth.

No Pluto transmit path or firmware modification is enabled by this release.
