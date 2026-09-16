#!/bin/sh
set -eu
apps=${GPSDR_OP25_APPS:-/usr/local/lib/gp-sdr/op25/apps}
export PYTHONPATH="$apps:$apps/tx:$apps/tdma:$apps/corr:$apps/util${PYTHONPATH:+:$PYTHONPATH}"
exec /usr/bin/python3 "$apps/multi_rx.py" "$@"
