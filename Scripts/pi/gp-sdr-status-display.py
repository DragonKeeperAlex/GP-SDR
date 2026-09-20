#!/usr/bin/env python3
"""Read-only, low-overhead GP-SDR overview for the Pi-local SPI TFT.

The service intentionally discovers the existing GP-SDR service token from the
running local process instead of storing a second copy of it.  It sends no
control requests and does not start, stop, tune, or transmit with a receiver.
"""

from __future__ import annotations

import datetime as dt
import json
import os
import re
import select
import struct
import subprocess
import time
import urllib.request
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


FRAMEBUFFER = Path(os.environ.get("GPSDR_TFT_FRAMEBUFFER", "/dev/fb0"))
WIDTH = int(os.environ.get("GPSDR_TFT_WIDTH", "240"))
HEIGHT = int(os.environ.get("GPSDR_TFT_HEIGHT", "320"))
REFRESH_SECONDS = max(1.0, float(os.environ.get("GPSDR_TFT_REFRESH_SECONDS", "1")))
PAGE_TIMEOUT_SECONDS = max(5.0, float(os.environ.get("GPSDR_TFT_PAGE_TIMEOUT_SECONDS", "30")))
DATA_ROOT = Path(os.environ.get("GPSDR_DATA_ROOT", "/mnt/gp-sdr-data"))
PAGE_NAMES = ("OVERVIEW", "POWER", "RADIOS", "SYSTEM")
_power_cache: dict[str, object] = {}
_power_checked = 0.0
_cpu_previous: tuple[int, int] | None = None
_network_previous: tuple[float, int, int] | None = None


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = [
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" if bold else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/truetype/freefont/FreeSansBold.ttf" if bold else "/usr/share/fonts/truetype/freefont/FreeSans.ttf",
    ]
    for candidate in candidates:
        if Path(candidate).exists():
            return ImageFont.truetype(candidate, size)
    return ImageFont.load_default()


FONTS = {"tiny": font(10), "small": font(12), "medium": font(15), "large": font(24, True), "title": font(18, True)}


def command(args: list[str]) -> str:
    try:
        return subprocess.check_output(args, text=True, stderr=subprocess.DEVNULL, timeout=2).strip()
    except (OSError, subprocess.SubprocessError):
        return ""


def gp_sdr_endpoint() -> tuple[str, str] | None:
    for entry in Path("/proc").iterdir():
        if not entry.name.isdigit():
            continue
        try:
            if (entry / "comm").read_text().strip() != "gp-sdr":
                continue
            values = [part.decode() for part in (entry / "cmdline").read_bytes().split(b"\0") if part]
        except OSError:
            continue
        try:
            port_flag = next(flag for flag in ("-port", "--port") if flag in values)
            token_flag = next(flag for flag in ("-token", "--token") if flag in values)
            return f"http://127.0.0.1:{values[values.index(port_flag) + 1]}", values[values.index(token_flag) + 1]
        except (StopIteration, ValueError, IndexError):
            continue
    return None


def api(path: str) -> object | None:
    endpoint = gp_sdr_endpoint()
    if endpoint is None:
        return None
    base, token = endpoint
    request = urllib.request.Request(f"{base}{path}", headers={"X-GP-SDR-Token": token})
    try:
        with urllib.request.urlopen(request, timeout=1.5) as response:
            return json.load(response)
    except Exception:
        return None


def direct_power() -> dict[str, object]:
    """Keep power visible even when GP-SDR is safely stopped for storage."""
    script = "import json;from pipower5.pipower5 import PiPower5;d=PiPower5().read_all();n=lambda k:float(d.get(k,0)or 0);print(json.dumps({'available':True,'inputVoltage':n('input_voltage')/1000,'inputCurrent':n('input_current')/1000,'inputPower':n('input_voltage')*n('input_current')/1000000,'outputVoltage':n('output_voltage')/1000,'outputCurrent':n('output_current')/1000,'outputPower':n('output_voltage')*n('output_current')/1000000,'batteryVoltage':n('battery_voltage')/1000,'batteryCurrent':n('battery_current')/1000,'batteryPercentage':n('battery_percentage'),'powerSource':'Battery'if int(n('power_source'))==1 else'External','inputPluggedIn':bool(d.get('is_input_plugged_in',False))}))"
    try:
        raw = subprocess.check_output(["/opt/pipower5/venv/bin/python3", "-I", "-c", script], text=True, stderr=subprocess.DEVNULL, timeout=1.5)
        return json.loads(raw)
    except Exception:
        return {}


def touch_device() -> int | None:
    for event in Path("/sys/class/input").glob("event*"):
        try:
            if "ADS7846" in (event / "device/name").read_text():
                return os.open(f"/dev/input/{event.name}", os.O_RDONLY | os.O_NONBLOCK)
        except OSError:
            continue
    return None


def tapped(fd: int | None) -> bool:
    if fd is None or not select.select([fd], [], [], 0)[0]:
        return False
    try:
        data = os.read(fd, 24 * 12)
    except OSError:
        return False
    for offset in range(0, len(data) - 23, 24):
        _, _, event_type, code, value = struct.unpack("llHHI", data[offset:offset + 24])
        if event_type == 1 and code == 330 and value == 0:
            return True
    return False


def number(value: object, default: float = 0.0) -> float:
    try:
        return float(value)  # type: ignore[arg-type]
    except (TypeError, ValueError):
        return default


def frequency(hz: object) -> str:
    value = number(hz)
    if value >= 1_000_000:
        return f"{value / 1_000_000:.4f} MHz"
    if value >= 1_000:
        return f"{value / 1_000:.1f} kHz"
    return "--"


def thermal() -> tuple[str, str]:
    raw = command(["vcgencmd", "measure_temp"])
    match = re.search(r"([0-9.]+)", raw)
    temperature = f"{match.group(1)} C" if match else "--"
    throttle = command(["vcgencmd", "get_throttled"])
    return temperature, "OK" if throttle.endswith("0x0") else (throttle or "--")


def free_space() -> str:
    try:
        stats = os.statvfs(DATA_ROOT)
        gib = stats.f_bavail * stats.f_frsize / (1024**3)
        return f"{gib:.1f} GiB free"
    except OSError:
        return "storage unavailable"


def system_metrics() -> dict[str, str]:
    """Return lightweight, host-local telemetry for the status panel."""
    global _cpu_previous, _network_previous
    result: dict[str, str] = {"cpu": "--", "ram": "--", "storage": free_space(), "network": "--", "rate": "--", "gpu": "--"}
    try:
        fields = Path("/proc/stat").read_text().splitlines()[0].split()[1:]
        values = [int(value) for value in fields]
        total, idle = sum(values), values[3] + (values[4] if len(values) > 4 else 0)
        if _cpu_previous:
            total_delta, idle_delta = total - _cpu_previous[0], idle - _cpu_previous[1]
            if total_delta > 0:
                result["cpu"] = f"{100 * (total_delta - idle_delta) / total_delta:.0f}%"
        _cpu_previous = (total, idle)
    except (OSError, ValueError, IndexError):
        pass
    try:
        info = {line.split(":", 1)[0]: int(line.split()[1]) for line in Path("/proc/meminfo").read_text().splitlines() if ":" in line}
        total, available = info.get("MemTotal", 0), info.get("MemAvailable", 0)
        if total:
            result["ram"] = f"{(total - available) / 1048576:.1f}/{total / 1048576:.1f}G"
    except (OSError, ValueError, IndexError):
        pass
    try:
        rx = tx = 0
        for line in Path("/proc/net/dev").read_text().splitlines()[2:]:
            name, values = line.split(":", 1)
            if name.strip() != "lo":
                counters = values.split()
                rx += int(counters[0]); tx += int(counters[8])
        now = time.monotonic()
        if _network_previous:
            elapsed = now - _network_previous[0]
            if elapsed > 0:
                result["rate"] = f"{(rx - _network_previous[1]) / elapsed / 1024:.0f}K↓ {(tx - _network_previous[2]) / elapsed / 1024:.0f}K↑"
        _network_previous = (now, rx, tx)
        addresses = command(["ip", "-o", "-4", "addr", "show", "up", "scope", "global"])
        match = re.search(r"\binet\s+([0-9.]+/[0-9]+)", addresses)
        if match:
            result["network"] = match.group(1)
    except (OSError, ValueError, IndexError):
        pass
    clock = command(["vcgencmd", "measure_clock", "v3d"])
    match = re.search(r"=(\d+)", clock)
    if match:
        result["gpu"] = f"{int(match.group(1)) / 1_000_000} MHz"
    return result


def short_name(device: dict[str, object]) -> str:
    name = str(device.get("name") or device.get("kind") or "Receiver")
    name = name.replace("FISHBall-PlutoSky (Z7010-AD9361)", "PlutoSDR")
    return name[:23]


def write_rgb565(image: Image.Image) -> None:
    image = image.convert("RGB")
    pixels = image.load()
    packed = bytearray(WIDTH * HEIGHT * 2)
    offset = 0
    for y in range(HEIGHT):
        for x in range(WIDTH):
            red, green, blue = pixels[x, y]
            value = ((red & 0xF8) << 8) | ((green & 0xFC) << 3) | (blue >> 3)
            packed[offset] = value & 0xFF
            packed[offset + 1] = value >> 8
            offset += 2
    with FRAMEBUFFER.open("r+b", buffering=0) as framebuffer:
        framebuffer.seek(0)
        framebuffer.write(packed)


def text(draw: ImageDraw.ImageDraw, xy: tuple[int, int], value: str, style: str = "small", fill: str = "#d8dee9") -> None:
    draw.text(xy, value, font=FONTS[style], fill=fill)


def line(draw: ImageDraw.ImageDraw, y: int) -> None:
    draw.line((10, y, WIDTH - 10, y), fill="#2e3440", width=1)


def dashboard(page: int = 0) -> Image.Image:
    status = api("/api/status")
    service_online = isinstance(status, dict)
    devices = api("/api/devices")
    p25 = api("/api/p25/status")
    status = status if isinstance(status, dict) else {}
    devices = devices if isinstance(devices, list) else []
    p25 = p25 if isinstance(p25, dict) else {}
    image = Image.new("RGB", (WIDTH, HEIGHT), "#11151c")
    draw = ImageDraw.Draw(image)
    ready = service_online
    active = bool(status.get("running"))
    accent = "#42d3a2" if ready else "#f5b74e"
    power = status.get("powerHat") if isinstance(status.get("powerHat"), dict) else direct_power()
    if HEIGHT <= 240:
        draw.rectangle((0, 0, WIDTH, 30), fill="#19212c")
        draw.rectangle((0, 28, WIDTH, 30), fill=accent)
        text(draw, (10, 6), f"GP-SDR  |  {PAGE_NAMES[page]}", "title")
        if page == 1:
            text(draw, (12, 46), f"BATTERY  {round(number(power.get('batteryPercentage')))}%", "large", "#e9c46a")
            text(draw, (12, 88), f"SOURCE   {power.get('powerSource') or 'Unknown'}", "medium")
            text(draw, (12, 116), f"INPUT    {number(power.get('inputVoltage')):.2f}V  {number(power.get('inputCurrent')):.2f}A  {number(power.get('inputPower')):.2f}W", "small")
            text(draw, (12, 144), f"OUTPUT   {number(power.get('outputVoltage')):.2f}V  {number(power.get('outputCurrent')):.2f}A  {number(power.get('outputPower')):.2f}W", "small")
            text(draw, (12, 172), f"BATTERY  {number(power.get('batteryVoltage')):.2f}V  {number(power.get('batteryCurrent')):.2f}A", "small")
        elif page == 2:
            connected = sum(1 for device in devices if isinstance(device, dict) and device.get("connected"))
            label = f"CONNECTED  {connected}/{len(devices)}" if status else "SERVICE OFFLINE"
            text(draw, (12, 46), label, "large", accent)
            if not status:
                text(draw, (12, 82), "Receiver discovery resumes when", "small", "#8e9bad")
                text(draw, (12, 102), "the GP-SDR data disk is mounted.", "small", "#8e9bad")
            y = 128 if not status else 86
            for device in devices[:5]:
                if isinstance(device, dict):
                    online = bool(device.get("connected") and device.get("available"))
                    draw.ellipse((14, y + 4, 22, y + 12), fill="#42d3a2" if online else "#c05260")
                    text(draw, (30, y), short_name(device), "medium")
                    y += 24
        elif page == 3:
            temperature, throttle = thermal()
            metrics = system_metrics()
            text(draw, (12, 44), f"CPU  {metrics['cpu']}  {temperature}", "medium")
            text(draw, (12, 68), f"GPU V3D  {metrics['gpu']}", "small")
            text(draw, (12, 92), f"RAM  {metrics['ram']}", "medium")
            text(draw, (12, 116), f"DISK  {metrics['storage']}", "small")
            text(draw, (12, 140), f"NET  {metrics['network']}", "medium")
            text(draw, (12, 164), f"RATE  {metrics['rate']}", "small")
            text(draw, (12, 188), f"THROTTLE  {throttle}", "small", "#42d3a2" if throttle == "OK" else "#c05260")
        if page:
            text(draw, (10, 212), f"Touch: next page • Overview in {int(PAGE_TIMEOUT_SECONDS)}s • {dt.datetime.now().strftime('%H:%M:%S')}", "tiny", "#8e9bad")
            return image
        text(draw, (10, 40), "RECEIVER SERVICE", "tiny", "#8e9bad")
        text(draw, (10, 54), "ACTIVE" if active else "READY" if ready else "OFFLINE", "large", accent)
        text(draw, (10, 84), str(status.get("mode") or "No active mode")[:29], "small")
        text(draw, (10, 112), "P25", "tiny", "#8e9bad")
        locked = str(p25.get("reception") or "").lower() == "locked"
        text(draw, (48, 109), "LOCK" if locked else "STANDBY", "medium", "#42d3a2" if locked else "#8593a6")
        text(draw, (10, 130), frequency(p25.get("controlChannelHz")), "small")
        text(draw, (10, 148), str(p25.get("note") or "Decoder inactive")[:32], "tiny", "#8e9bad")
        draw.line((160, 38, 160, 198), fill="#2e3440", width=1)
        connected = sum(1 for device in devices if isinstance(device, dict) and device.get("connected"))
        text(draw, (174, 40), f"RECEIVERS  {connected}/{len(devices)}" if status else "RECEIVERS: OFFLINE", "tiny", "#8e9bad")
        y = 55
        for device in devices[:4]:
            if isinstance(device, dict):
                online = bool(device.get("connected") and device.get("available"))
                draw.ellipse((175, y + 3, 181, y + 9), fill="#42d3a2" if online else "#c05260")
                text(draw, (186, y), short_name(device), "tiny")
                y += 14
        if power and power.get("available"):
            percent = max(0, min(100, round(number(power.get("batteryPercentage")))))
            text(draw, (174, 122), f"POWER  {power.get('powerSource') or 'Unknown'}  {percent}%", "small", "#e9c46a")
            text(draw, (174, 140), f"IN {number(power.get('inputVoltage')):.1f}V {number(power.get('inputPower')):.1f}W", "tiny")
            text(draw, (174, 154), f"OUT {number(power.get('outputPower')):.1f}W", "tiny")
        draw.line((10, 204, WIDTH - 10, 204), fill="#2e3440", width=1)
        text(draw, (10, 212), f"Touch for Power • Radios • System  |  {dt.datetime.now().strftime('%H:%M:%S')}", "tiny", "#8e9bad")
        return image
    draw.rectangle((0, 0, WIDTH, 38), fill="#19212c")
    draw.rectangle((0, 36, WIDTH, 38), fill=accent)
    text(draw, (10, 8), "GP-SDR  |  PI OVERVIEW", "title")
    text(draw, (10, 46), "RECEIVER SERVICE", "tiny", "#8e9bad")
    text(draw, (10, 60), "ACTIVE" if active else "READY" if ready else "OFFLINE", "large", accent)
    text(draw, (10, 91), str(status.get("mode") or "No active mode")[:31], "small")
    line(draw, 110)

    locked = str(p25.get("reception") or "").lower() == "locked"
    p25_color = "#42d3a2" if locked else "#8593a6"
    text(draw, (10, 118), "P25", "tiny", "#8e9bad")
    text(draw, (48, 115), "LOCK" if locked else "STANDBY", "medium", p25_color)
    text(draw, (10, 136), frequency(p25.get("controlChannelHz")), "small")
    text(draw, (10, 153), str(p25.get("note") or "Decoder inactive")[:35], "tiny", "#8e9bad")
    line(draw, 169)

    connected = sum(1 for device in devices if isinstance(device, dict) and device.get("connected"))
    text(draw, (10, 177), f"RECEIVERS  {connected}/{len(devices)}", "tiny", "#8e9bad")
    y = 192
    for device in devices[:4]:
        if not isinstance(device, dict):
            continue
        online = bool(device.get("connected") and device.get("available"))
        draw.ellipse((11, y + 3, 17, y + 9), fill="#42d3a2" if online else "#c05260")
        text(draw, (22, y), short_name(device), "tiny")
        y += 13
    line(draw, 247)

    power = status.get("powerHat") if isinstance(status.get("powerHat"), dict) else {}
    if power and power.get("available"):
        percent = max(0, min(100, round(number(power.get("batteryPercentage")))))
        text(draw, (10, 255), f"POWER  {power.get('powerSource') or 'Unknown'}  {percent}%", "small", "#e9c46a")
        text(draw, (10, 272), f"IN {number(power.get('inputVoltage')):.1f}V {number(power.get('inputPower')):.1f}W  OUT {number(power.get('outputPower')):.1f}W", "tiny")
    else:
        text(draw, (10, 255), "POWER  telemetry unavailable", "small", "#c05260")

    temperature, throttle = thermal()
    metrics = system_metrics()
    text(draw, (10, 290), f"CPU {metrics['cpu']} {temperature} • RAM {metrics['ram']} • GPU {metrics['gpu']}", "tiny")
    text(draw, (10, 304), f"{metrics['storage']} • NET {metrics['network']} {metrics['rate']} • {dt.datetime.now().strftime('%H:%M:%S')}", "tiny", "#8e9bad")
    return image


def main() -> None:
    page, touch = 0, None
    last_touch = time.monotonic()
    while True:
        try:
            if touch is None:
                touch = touch_device()
            if tapped(touch):
                page = (page + 1) % len(PAGE_NAMES)
                last_touch = time.monotonic()
            elif page and time.monotonic() - last_touch >= PAGE_TIMEOUT_SECONDS:
                page = 0
            write_rgb565(dashboard(page))
        except Exception:
            # A display should never destabilize the GP-SDR service. Retry on
            # device/driver startup or a temporary framebuffer failure.
            pass
        time.sleep(REFRESH_SECONDS)


if __name__ == "__main__":
    main()
