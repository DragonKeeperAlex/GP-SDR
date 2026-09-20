# Pi-local status display

The optional SHCHV/LCDWIKI 2.4-inch SPI display is a read-only GP-SDR
overview. It does not expose transmit, tuning, mapper, or receiver controls.
It reports the local service state, P25 lock/control channel, connected
receivers, PiPower5 values, Pi temperature/throttle state, and free data-drive
space every two seconds.

## Supported panel

This setup is for the 320x240 resistive-touch panel with an ILI9341 LCD and
XPT2046 touch controller. The LCD occupies SPI0 CE0, the unused touch
controller occupies CE1, GPIO22 is LCD D/C, and GPIO27 is reset. The PiPower5
HAT telemetry uses I2C and was verified on a separate bus/address before this
overlay was enabled. Touch is deliberately left disabled: the dashboard is an
at-a-glance display, not an unsafe duplicate control surface.

## Installation on the Pi

1. Compile `Scripts/pi/gp-sdr-ili9341-overlay.dts` with `dtc -@ -I dts -O dtb`
   and install it as `/boot/firmware/overlays/gp-sdr-ili9341.dtbo`. Preserve
   `/boot/firmware/config.txt`, then append the single overlay line from
   `Scripts/pi/gp-sdr-status-display-overlay.conf` if it is not already present.
2. Install `Scripts/pi/gp-sdr-status-display.py` as
   `/usr/local/lib/gp-sdr/gp-sdr-status-display.py`, and install the matching
   systemd unit as `/etc/systemd/system/gp-sdr-status-display.service`.
3. Reload systemd, enable and start `gp-sdr-status-display.service`.
4. Reboot once so the boot overlay owns the framebuffer. The service will then
   start after GP-SDR and automatically retry if the framebuffer appears late.

The installed configuration uses the panel-specific 270-degree orientation,
16 MHz SPI rate, and a 32 KiB transfer buffer. These values avoid the rotated
and striped output that can occur with the generic 32 MHz/default-buffer setup.

The dashboard reads the existing local GP-SDR process token at runtime. It
does not store or print that token, and sends only local `GET` requests.

## Recovery

If the panel is blank or rotated incorrectly, disable
`gp-sdr-status-display.service`, remove the one overlay line, and reboot. This
leaves GP-SDR, receiver configuration, PiPower5, and stored data untouched.
