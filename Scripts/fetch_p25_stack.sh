#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SDRTRUNK_VERSION=0.6.1
JMBE_VERSION=1.0.9
COMPONENT_ROOT="$PROJECT_ROOT/build/components"
LICENSE_ROOT="$PROJECT_ROOT/build/p25-licenses"
DOWNLOAD_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-sdrtrunk.XXXXXX")
trap 'rm -rf "$DOWNLOAD_ROOT"' EXIT INT TERM

mkdir -p "$COMPONENT_ROOT"
mkdir -p "$LICENSE_ROOT"

component_ready() {
  test -d "$1/bin" && test -d "$1/lib"
}

fetch_component() {
  project=$1
  version=$2
  asset=$3
  destination=$4
  if component_ready "$destination"; then
    printf 'Using cached %s component for %s.\n' "$project" "$(basename "$destination")"
    return
  fi
  archive="$DOWNLOAD_ROOT/$asset"
  source="https://github.com/DSheirer/$project/releases/download/v$version/$asset"
  curl --fail --location --silent --show-error --connect-timeout 15 --max-time 600 \
    "$source" --output "$archive"
  unpack="$DOWNLOAD_ROOT/unpack-$(printf '%s' "$asset" | tr '.-' '__')"
  mkdir -p "$unpack"
  unzip -q "$archive" -d "$unpack"
  root=$(find "$unpack" -mindepth 1 -maxdepth 1 -type d | head -n 1)
  if [ -z "$root" ]; then
    printf 'No component directory found in %s\n' "$asset" >&2
    exit 1
  fi
  rm -rf "$destination"
  mkdir -p "$(dirname "$destination")"
  mv "$root" "$destination"
  (cd "$DOWNLOAD_ROOT" && shasum -a 256 "$asset") >> "$LICENSE_ROOT/COMPONENT_SHA256SUMS.txt"
}

touch "$LICENSE_ROOT/COMPONENT_SHA256SUMS.txt"

fetch_component sdrtrunk "$SDRTRUNK_VERSION" "sdr-trunk-osx-aarch64-v$SDRTRUNK_VERSION.zip" "$COMPONENT_ROOT/sdrtrunk/darwin-arm64"
fetch_component sdrtrunk "$SDRTRUNK_VERSION" "sdr-trunk-osx-x86_64-v$SDRTRUNK_VERSION.zip" "$COMPONENT_ROOT/sdrtrunk/darwin-amd64"
fetch_component sdrtrunk "$SDRTRUNK_VERSION" "sdr-trunk-linux-aarch64-v$SDRTRUNK_VERSION.zip" "$COMPONENT_ROOT/sdrtrunk/linux-arm64"
fetch_component sdrtrunk "$SDRTRUNK_VERSION" "sdr-trunk-linux-x86_64-v$SDRTRUNK_VERSION.zip" "$COMPONENT_ROOT/sdrtrunk/linux-amd64"
fetch_component sdrtrunk "$SDRTRUNK_VERSION" "sdr-trunk-windows-x86_64-v$SDRTRUNK_VERSION.zip" "$COMPONENT_ROOT/sdrtrunk/windows-amd64"

fetch_component jmbe "$JMBE_VERSION" "jmbe-creator-osx-aarch64-v$JMBE_VERSION.zip" "$COMPONENT_ROOT/jmbe-creator/darwin-arm64"
fetch_component jmbe "$JMBE_VERSION" "jmbe-creator-osx-x86_64-v$JMBE_VERSION.zip" "$COMPONENT_ROOT/jmbe-creator/darwin-amd64"
fetch_component jmbe "$JMBE_VERSION" "jmbe-creator-linux-aarch64-v$JMBE_VERSION.zip" "$COMPONENT_ROOT/jmbe-creator/linux-arm64"
fetch_component jmbe "$JMBE_VERSION" "jmbe-creator-linux-x86_64-v$JMBE_VERSION.zip" "$COMPONENT_ROOT/jmbe-creator/linux-amd64"
fetch_component jmbe "$JMBE_VERSION" "jmbe-creator-windows-x86_64-v$JMBE_VERSION.zip" "$COMPONENT_ROOT/jmbe-creator/windows-amd64"

restore_or_fetch() {
  destination=$1
  installed_name=$2
  source=$3
  if [ -s "$destination" ]; then return; fi
  installed="/Applications/GP-SDR.app/Contents/Resources/Documentation/$installed_name"
  if [ -s "$installed" ]; then
    cp "$installed" "$destination"
    return
  fi
  curl --fail --location --silent --show-error --connect-timeout 15 --max-time 300 \
    "$source" --output "$destination"
}

restore_or_fetch "$LICENSE_ROOT/SDRTrunk-v$SDRTRUNK_VERSION-source.tar.gz" \
  "SDRTrunk-v$SDRTRUNK_VERSION-source.tar.gz" \
  "https://github.com/DSheirer/sdrtrunk/archive/refs/tags/v$SDRTRUNK_VERSION.tar.gz"
restore_or_fetch "$LICENSE_ROOT/SDRTrunk-LICENSE" "SDRTrunk-LICENSE" \
  "https://raw.githubusercontent.com/DSheirer/sdrtrunk/v$SDRTRUNK_VERSION/LICENSE"
restore_or_fetch "$LICENSE_ROOT/JMBE-v$JMBE_VERSION-source.tar.gz" \
  "JMBE-v$JMBE_VERSION-source.tar.gz" \
  "https://github.com/DSheirer/jmbe/archive/refs/tags/v$JMBE_VERSION.tar.gz"
restore_or_fetch "$LICENSE_ROOT/JMBE-LICENSE" "JMBE-LICENSE" \
  "https://raw.githubusercontent.com/DSheirer/jmbe/v$JMBE_VERSION/LICENSE"

printf 'SDRTrunk v%s and JMBE Creator v%s components are ready.\n' "$SDRTRUNK_VERSION" "$JMBE_VERSION"
