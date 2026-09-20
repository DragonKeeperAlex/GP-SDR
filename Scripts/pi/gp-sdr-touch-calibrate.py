#!/usr/bin/env python3
"""Four-point calibration for the GP-SDR Pi-local ADS7846 touchscreen.

Run this while the dashboard service is stopped.  It writes only the measured
coordinate transform under the `sdr` user's state directory; it never changes
receiver, mapper, or radio configuration.
"""

from __future__ import annotations

import json
import os
import select
import struct
import time
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


WIDTH = int(os.environ.get("GPSDR_TFT_WIDTH", "320"))
HEIGHT = int(os.environ.get("GPSDR_TFT_HEIGHT", "240"))
FRAMEBUFFER = Path(os.environ.get("GPSDR_TFT_FRAMEBUFFER", "/dev/fb0"))
STATE_FILE = Path(os.environ.get("XDG_STATE_HOME", str(Path.home() / ".local/state"))) / "gp-sdr/touch-calibration.json"
TARGETS = ((26, 26), (WIDTH - 27, 26), (WIDTH - 27, HEIGHT - 27), (26, HEIGHT - 27))


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    path = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" if bold else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
    return ImageFont.truetype(path, size) if Path(path).exists() else ImageFont.load_default()


def write_frame(image: Image.Image) -> None:
    pixels = image.convert("RGB").load()
    packed = bytearray(WIDTH * HEIGHT * 2)
    index = 0
    for y in range(HEIGHT):
        for x in range(WIDTH):
            red, green, blue = pixels[x, y]
            value = ((red & 0xF8) << 8) | ((green & 0xFC) << 3) | (blue >> 3)
            packed[index], packed[index + 1] = value & 0xFF, value >> 8
            index += 2
    with FRAMEBUFFER.open("r+b", buffering=0) as fb:
        fb.seek(0)
        fb.write(packed)


def draw_target(index: int) -> None:
    image = Image.new("RGB", (WIDTH, HEIGHT), "#11151c")
    draw = ImageDraw.Draw(image)
    draw.rectangle((0, 0, WIDTH, 30), fill="#19212c")
    draw.text((10, 6), "GP-SDR TOUCH CALIBRATION", font=font(15, True), fill="#e5e9f0")
    draw.text((10, 42), f"Tap target {index + 1} of {len(TARGETS)}", font=font(18, True), fill="#42d3a2")
    draw.text((10, 66), "Use the center of the crosshair.", font=font(12), fill="#aab5c4")
    x, y = TARGETS[index]
    draw.line((x - 14, y, x + 14, y), fill="#f5b74e", width=2)
    draw.line((x, y - 14, x, y + 14), fill="#f5b74e", width=2)
    draw.ellipse((x - 4, y - 4, x + 4, y + 4), outline="#ffffff", width=2)
    write_frame(image)


def touch_device() -> int:
    for event in Path("/sys/class/input").glob("event*"):
        try:
            if "ADS7846" in (event / "device/name").read_text():
                return os.open(f"/dev/input/{event.name}", os.O_RDONLY | os.O_NONBLOCK)
        except OSError:
            continue
    raise RuntimeError("ADS7846 touchscreen was not found")


def read_release(fd: int) -> tuple[int, int]:
    raw_x = raw_y = None
    while True:
        ready, _, _ = select.select([fd], [], [], 30)
        if not ready:
            raise TimeoutError("No touch received within 30 seconds")
        data = os.read(fd, 24 * 32)
        for offset in range(0, len(data) - 23, 24):
            _, _, event_type, code, value = struct.unpack("llHHI", data[offset:offset + 24])
            if event_type == 3 and code == 0:
                raw_x = value
            elif event_type == 3 and code == 1:
                raw_y = value
            elif event_type == 1 and code == 330 and value == 0 and raw_x is not None and raw_y is not None:
                return raw_x, raw_y


def solve(matrix: list[list[float]], values: list[float]) -> list[float]:
    """Small Gaussian elimination solver for the affine least-squares system."""
    size = len(values)
    augmented = [matrix[row][:] + [values[row]] for row in range(size)]
    for column in range(size):
        pivot = max(range(column, size), key=lambda row: abs(augmented[row][column]))
        if abs(augmented[pivot][column]) < 1e-9:
            raise ValueError("Touch points are not sufficiently distinct")
        augmented[column], augmented[pivot] = augmented[pivot], augmented[column]
        scale = augmented[column][column]
        augmented[column] = [value / scale for value in augmented[column]]
        for row in range(size):
            if row != column:
                factor = augmented[row][column]
                augmented[row] = [left - factor * right for left, right in zip(augmented[row], augmented[column])]
    return [augmented[row][-1] for row in range(size)]


def affine(samples: list[tuple[int, int]]) -> dict[str, list[float]]:
    rows = [[float(x), float(y), 1.0] for x, y in samples]
    normal = [[sum(row[i] * row[j] for row in rows) for j in range(3)] for i in range(3)]
    target_x = [sum(row[i] * TARGETS[index][0] for index, row in enumerate(rows)) for i in range(3)]
    target_y = [sum(row[i] * TARGETS[index][1] for index, row in enumerate(rows)) for i in range(3)]
    return {"x": solve(normal, target_x), "y": solve(normal, target_y)}


def done() -> None:
    image = Image.new("RGB", (WIDTH, HEIGHT), "#11151c")
    draw = ImageDraw.Draw(image)
    draw.text((18, 72), "Calibration saved", font=font(24, True), fill="#42d3a2")
    draw.text((18, 110), "Returning to GP-SDR overview...", font=font(14), fill="#d8dee9")
    write_frame(image)


def main() -> None:
    fd = touch_device()
    samples: list[tuple[int, int]] = []
    try:
        for index in range(len(TARGETS)):
            draw_target(index)
            samples.append(read_release(fd))
            time.sleep(0.4)
        STATE_FILE.parent.mkdir(parents=True, exist_ok=True)
        STATE_FILE.write_text(json.dumps({"version": 1, "screen": [WIDTH, HEIGHT], "targets": TARGETS, "raw": samples, "affine": affine(samples), "calibratedAt": int(time.time())}, indent=2) + "\n")
        done()
        time.sleep(2)
    finally:
        os.close(fd)


if __name__ == "__main__":
    main()
