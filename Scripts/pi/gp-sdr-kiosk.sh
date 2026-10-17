#!/bin/sh
# Start a touch-only GP-SDR session. The access token is read from the running
# service environment and is neither written to disk nor printed.
set -eu

token="${GPSDR_WEB_TOKEN:-}"
# Keep the script usable during manual runs, while the service itself obtains
# the stable credential directly from the protected EnvironmentFile.
for entry in /proc/[0-9]*; do
    [ -n "$token" ] && break
    [ "$(cat "$entry/comm" 2>/dev/null || true)" = "gp-sdr" ] || continue
    token=$(tr '\0' '\n' < "$entry/environ" 2>/dev/null | sed -n 's/^GPSDR_WEB_TOKEN=//p' | head -n 1)
    [ -n "$token" ] && break
done

[ -n "$token" ] || { echo "GP-SDR token is unavailable" >&2; exit 1; }

# The keyboard belongs to the Pi's Wayland session, rather than the web app.
keyboard_pid=""
if command -v squeekboard >/dev/null 2>&1; then
    squeekboard &
    keyboard_pid=$!
fi
cleanup() { [ -z "$keyboard_pid" ] || kill "$keyboard_pid" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

/usr/bin/chromium \
    --kiosk --incognito --no-first-run --no-default-browser-check \
    --disable-features=Translate,MediaRouter --touch-events=enabled \
    --enable-features=UseOzonePlatform --ozone-platform=wayland \
    --user-data-dir=/home/sdr/.config/gp-sdr-kiosk \
    "http://127.0.0.1:8073/?display=touch" &
browser_pid=$!
wait "$browser_pid"
