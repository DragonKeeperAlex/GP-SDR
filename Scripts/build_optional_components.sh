#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUTPUT_ROOT=${1:-"$PROJECT_ROOT/build/helpers"}
mkdir -p "$OUTPUT_ROOT"

if command -v cmake >/dev/null 2>&1; then
  BUILD_DIRECTORY=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-soapy.XXXXXX")
  trap 'rm -rf "$BUILD_DIRECTORY"' EXIT INT TERM
  cmake -S "$PROJECT_ROOT/helpers/soapy_capture" -B "$BUILD_DIRECTORY" -DCMAKE_BUILD_TYPE=Release
  cmake --build "$BUILD_DIRECTORY" --config Release
  if [ -f "$BUILD_DIRECTORY/gpsdr-soapy" ]; then
    cp "$BUILD_DIRECTORY/gpsdr-soapy" "$OUTPUT_ROOT/gpsdr-soapy"
  elif [ -f "$BUILD_DIRECTORY/Release/gpsdr-soapy.exe" ]; then
    cp "$BUILD_DIRECTORY/Release/gpsdr-soapy.exe" "$OUTPUT_ROOT/gpsdr-soapy.exe"
  fi
  printf 'Built SoapySDR stream helper in %s\n' "$OUTPUT_ROOT"
else
  printf 'CMake was not found; the SoapySDR helper was skipped.\n' >&2
fi
