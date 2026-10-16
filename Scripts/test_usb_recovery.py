#!/usr/bin/env python3
"""Presence-only fixtures: never write state, probe buses, or open a receiver."""
import os
import pathlib
import subprocess
import tempfile

script = pathlib.Path(__file__).parent / 'pi' / 'gp-sdr-usb-recovery'
with tempfile.TemporaryDirectory() as directory:
    root = pathlib.Path(directory)
    def device(name, vendor, product, serial=''):
        path = root / name
        path.mkdir()
        (path / 'idVendor').write_text(vendor)
        (path / 'idProduct').write_text(product)
        (path / 'serial').write_text(serial)
    device('hackrf', '1d50', '6089', '0000000000000000922c63dc217ea447')
    device('rtl', '0bda', '2838')
    env = {**os.environ, 'GPSDR_USB_SYSFS_ROOT': str(root)}
    def check():
        return subprocess.run(['sh', str(script), '--check-only'], env=env, check=False).returncode
    assert check() == 0, 'one usable HackRF and RTL suffice; Pluto can use Ethernet'
    (root / 'hackrf' / 'serial').write_text('000000000000000024b862dc3140c5cb')
    assert check() == 1, 'excluded HackRF must not satisfy receiver presence'
    (root / 'hackrf' / 'serial').write_text('0000000000000000922c63dc217ea447')
    (root / 'rtl' / 'idProduct').write_text('0000')
    assert check() == 1, 'missing RTL must be reported'
print('USB presence and excluded-device fixtures passed.')
