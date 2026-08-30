# Linux and Windows Server Setup

This guide configures GP-SDR as an always-on receiver with a browser interface.
The same interface works from a desktop or phone. Start on a trusted LAN; use a
VPN for access from elsewhere.

## Before enabling network access

- Install and test GP-SDR locally first.
- Confirm at least one receiver is shown as **Connected** under Hardware.
- Choose a fixed data directory with enough storage and configure recording/IQ
  limits under Settings.
- Use a long, unique token. Anyone with the token can control the receiver.
- Never forward port 8073 directly from an internet router.

## Debian or Ubuntu with systemd

### 1. Install the package

Choose the package matching the host:

```bash
sudo apt install ./gp-sdr_1.3.2_amd64.deb
# 64-bit ARM host:
sudo apt install ./gp-sdr_1.3.2_arm64.deb
sudo systemctl enable --now gp-sdr
```

The packaged service listens only on `127.0.0.1:8073`. Its restricted dynamic
account writes GP-SDR data beneath `/var/lib/gp-sdr/GP-SDR`.

Check the local service:

```bash
systemctl status gp-sdr --no-pager
curl -I http://127.0.0.1:8073/
journalctl -u gp-sdr -n 100 --no-pager
```

### 2. Install receiver tools and USB rules

```bash
sudo apt update
sudo apt install hackrf rtl-sdr soapysdr-tools
hackrf_info
rtl_test -t
SoapySDRUtil --find
```

Only run one receiver test at a time. Stop `rtl_tcp`, SDR++, or another program
that already owns the device. If a PortaPack is attached, put it in HackRF USB
mode.

Many Debian-family SDR packages install udev rules. Reconnect the receiver after
installation. If the device node is restricted to `plugdev`, add the group to
the service without changing the packaged unit:

```bash
sudo systemctl edit gp-sdr
```

Add:

```ini
[Service]
SupplementaryGroups=plugdev
```

Then apply it:

```bash
sudo systemctl daemon-reload
sudo systemctl restart gp-sdr
```

Do not add this override when the host does not have a `plugdev` group. Inspect
the receiver's `/dev/bus/usb` permissions or distribution udev rules first.

### 3. Allow trusted-LAN access

Generate a random token and save it in a password manager. Then override the
service:

```bash
sudo systemctl edit gp-sdr
```

Replace the example token:

```ini
[Service]
ExecStart=
ExecStart=/usr/bin/gp-sdr -listen 0.0.0.0 -port 8073 -token REPLACE_WITH_A_LONG_RANDOM_TOKEN
```

Restart and inspect the log:

```bash
sudo systemctl daemon-reload
sudo systemctl restart gp-sdr
journalctl -u gp-sdr -n 50 --no-pager
```

If UFW is enabled, restrict the rule to the local subnet rather than opening it
globally. Adjust the subnet to match the network:

```bash
sudo ufw allow from 192.168.1.0/24 to any port 8073 proto tcp
```

Open `http://SERVER-LAN-IP:8073/?token=YOUR_TOKEN` from a device on that LAN.

### 4. Optional helper executables

The service searches `/usr/local/bin`, `/usr/bin`, `/snap/bin`, normal `PATH`,
and `GPSDR_HELPERS`. For manually installed decoders in a private directory,
extend the same systemd override:

```ini
[Service]
Environment=GPSDR_HELPERS=/opt/gp-sdr/helpers
```

Keep one executable per supported decoder name and restart GP-SDR after changes.
See [Optional Components](Optional-Components) for exact names and verification.

### 5. Update, back up, and recover

```bash
sudo systemctl stop gp-sdr
sudo cp -a /var/lib/gp-sdr/GP-SDR /path/to/backup/GP-SDR
sudo apt install ./gp-sdr_NEW-VERSION_amd64.deb
sudo systemctl start gp-sdr
```

Confirm the version in Settings after every update. `systemctl revert gp-sdr`
removes local overrides, so record the token and LAN/USB configuration before
using it.

## Windows 10 or 11

### 1. Prepare a fixed installation

1. Extract the complete x86_64 ZIP to a stable folder such as `C:\GP-SDR`. Do
   not run it from inside the ZIP.
2. Keep `GP-SDR.exe`, `sdrtrunk`, and `jmbe-creator` together.
3. Create a writable data folder such as `C:\GP-SDR-Data`.
4. Test locally from PowerShell:

```powershell
cd C:\GP-SDR
.\GP-SDR.exe -listen 127.0.0.1 -port 8073 -data C:\GP-SDR-Data -open
```

### 2. Configure receivers

HackRF and RTL-SDR normally require WinUSB. Use Zadig only on the exact SDR
interface:

- HackRF: select the HackRF interface and WinUSB.
- RTL-SDR: select **Bulk-In, Interface 0** for the dongle and WinUSB.
- Never replace a keyboard, mouse, storage-device, Bluetooth, or unrelated USB
  driver.

Reconnect the radio, open Hardware, and press **Refresh**. Test one known FM
station before attempting P25 or Mapper.

### 3. Run an unattended server

Generate a long random token and keep it private. First test the intended
command in PowerShell:

```powershell
C:\GP-SDR\GP-SDR.exe -listen 0.0.0.0 -port 8073 -token REPLACE_WITH_A_LONG_RANDOM_TOKEN -data C:\GP-SDR-Data
```

Then open **Task Scheduler → Create Task**:

1. Name it `GP-SDR Server`.
2. Choose **Run whether user is logged on or not** if the receiver must operate
   before sign-in.
3. Trigger it **At startup**.
4. Set **Program/script** to `C:\GP-SDR\GP-SDR.exe`.
5. Set **Add arguments** to the flags shown above.
6. Set **Start in** to `C:\GP-SDR`.
7. Enable automatic restart after failure and disable any short execution-time
   limit.

Run the task manually once and confirm `GP-SDR.exe` is listening:

```powershell
Get-Process GP-SDR
Test-NetConnection 127.0.0.1 -Port 8073
```

The token is stored in the Task Scheduler configuration and is readable by
administrators. Use a dedicated random token, not a reused account password.

### 4. Add a private-network firewall rule

Run PowerShell as Administrator:

```powershell
New-NetFirewallRule -DisplayName "GP-SDR trusted LAN" -Direction Inbound `
  -Action Allow -Protocol TCP -LocalPort 8073 -Profile Private `
  -RemoteAddress LocalSubnet
```

Do not add a Public-profile or Any-address rule. Open
`http://WINDOWS-LAN-IP:8073/?token=YOUR_TOKEN` from a trusted LAN device.

### 5. Optional helper executables

Place supported native Windows decoder executables and their required DLLs in a
dedicated folder, then add that folder to the account's `PATH` or set
`GPSDR_HELPERS` before restarting the task:

```powershell
[Environment]::SetEnvironmentVariable(
  "GPSDR_HELPERS", "C:\GP-SDR\Helpers", "User")
```

GP-SDR cannot directly launch a Linux binary from the native Windows package;
use a native build or run GP-SDR on a Linux host.

See [Optional Components](Optional-Components) for supported executable names,
upstream downloads, and readiness checks.

### 6. Update, back up, and recover

1. Stop the scheduled task.
2. Copy `C:\GP-SDR-Data` to a backup location.
3. Extract the new release to a new folder.
4. Preserve the previous program folder until the new version is verified.
5. Update the scheduled task's path only when the folder name changed.
6. Start the task and confirm the reported version, Hardware state, and decoder
   readiness.

## Remote receiver alternative

An `rtl_tcp` source can keep the USB dongle on another computer, but remote
gain, bias tee, calibration, and sample-rate controls may be reduced. Never
expose unauthenticated `rtl_tcp` directly to the internet; keep it on a trusted
LAN or VPN.

## Server troubleshooting

| Symptom | Check |
| --- | --- |
| Browser cannot connect locally | Confirm the process/service is running and port 8073 is listening. |
| Local works, phone does not | Confirm `-listen 0.0.0.0`, the Private/LAN firewall rule, server IP, and token. |
| Receiver appears but cannot open | Stop competing SDR software and verify USB driver/udev permissions for the service account. |
| Decoder says Setup | Install the executable, put it on a searched path or `GPSDR_HELPERS`, restart, then Refresh. |
| Server restarts repeatedly | Inspect `journalctl -u gp-sdr` on Linux or Task Scheduler History/Event Viewer on Windows. |
| Storage grows unexpectedly | Configure recording/IQ caps, rejected-IQ retention, and automatic cleanup under Settings. |
