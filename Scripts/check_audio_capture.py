"""Inspect a saved /api/live-audio stream without playing it or changing it."""
import math
import struct
import sys

with open(sys.argv[1], "rb") as capture:
    data = capture.read()
position = frames = samples = power = peak = 0
duration = 0.0
rates, channels = set(), set()
while position + 10 <= len(data):
    length, rate, count = struct.unpack_from("<HII", data, position)
    end = position + 10 + length + count * 2
    if end > len(data) or rate == 0:
        break
    channels.add(data[position + 10:position + 10 + length].decode())
    rates.add(rate)
    values = struct.unpack_from("<" + str(count) + "h", data, position + 10 + length)
    frames += 1
    samples += count
    duration += count / rate
    power += sum(value * value for value in values)
    peak = max(peak, max(map(abs, values), default=0))
    position = end
print(dict(frames=frames, samples=samples, seconds=round(duration, 3),
           rates=sorted(rates), channels=sorted(channels),
           rms=round(math.sqrt(power / max(samples, 1)), 1), peak=peak,
           trailing_bytes=len(data) - position))
