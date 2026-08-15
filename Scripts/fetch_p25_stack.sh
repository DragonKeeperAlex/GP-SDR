#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=0.9.8
BASE_URL="https://github.com/MattCheramie/GopherTrunk/releases/download/v$VERSION"
CACHE_ROOT="$PROJECT_ROOT/build/p25-cache/v$VERSION"
HELPER_ROOT="$PROJECT_ROOT/build/helpers"
LICENSE_ROOT="$PROJECT_ROOT/build/p25-licenses"
mkdir -p "$CACHE_ROOT" "$HELPER_ROOT" "$LICENSE_ROOT"

expected_hash() {
  case "$1" in
    gophertrunk-v0.9.8-darwin-amd64.tar.gz) printf '%s' 413220e6591ad606ac6f9cee702b367cb4f19cbd7a1d334ea7883f041d390014 ;;
    gophertrunk-v0.9.8-darwin-arm64.tar.gz) printf '%s' 0ea913ab233e86573b260d5a087eae6458e8327727ad341405ef1c152b2f4ec8 ;;
    gophertrunk-v0.9.8-linux-amd64.tar.gz) printf '%s' e63bc3c7f6e22be02c8cbce36d26c5beedb2456bbc2be00ceda9d870200b4673 ;;
    gophertrunk-v0.9.8-linux-arm64.tar.gz) printf '%s' bdefc7dc0bdca9134021b7d7639da8fb5cd80b2a537f73a57a7e9431671b68fb ;;
    gophertrunk-v0.9.8-windows-amd64.zip) printf '%s' b8e5e0cf7799b76ad4645c49eca417cbc2bf97ca87f10292ab931ad52df04db7 ;;
    *) return 1 ;;
  esac
}

verify_archive() {
  archive=$1
  expected=$(expected_hash "$(basename "$archive")")
  actual=$(shasum -a 256 "$archive" | awk '{print $1}')
  if [ "$actual" != "$expected" ]; then
    printf 'P25 engine checksum mismatch for %s\n' "$archive" >&2
    exit 1
  fi
}

install_archive() {
  target=$1
  archive_name=$2
  binary_name=$3
  archive="$CACHE_ROOT/$archive_name"
  if [ ! -f "$archive" ]; then
    curl -fL "$BASE_URL/$archive_name" -o "$archive"
  fi
  verify_archive "$archive"
  unpack=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-p25.XXXXXX")
  case "$archive" in
    *.zip) unzip -q "$archive" -d "$unpack" ;;
    *) tar -xzf "$archive" -C "$unpack" ;;
  esac
  source_binary=$(find "$unpack" -type f \( -name gophertrunk -o -name gophertrunk.exe \) -print | head -n 1)
  if [ -z "$source_binary" ]; then
    printf 'P25 engine binary missing from %s\n' "$archive" >&2
    exit 1
  fi
  mkdir -p "$HELPER_ROOT/$target"
  cp "$source_binary" "$HELPER_ROOT/$target/$binary_name"
  chmod 755 "$HELPER_ROOT/$target/$binary_name"
  rm -rf "$unpack"
}

install_archive darwin-arm64 "gophertrunk-v$VERSION-darwin-arm64.tar.gz" gophertrunk
install_archive darwin-amd64 "gophertrunk-v$VERSION-darwin-amd64.tar.gz" gophertrunk
install_archive linux-amd64 "gophertrunk-v$VERSION-linux-amd64.tar.gz" gophertrunk
install_archive linux-arm64 "gophertrunk-v$VERSION-linux-arm64.tar.gz" gophertrunk
install_archive windows-amd64 "gophertrunk-v$VERSION-windows-amd64.zip" gophertrunk.exe

curl -fsSL "https://raw.githubusercontent.com/MattCheramie/GopherTrunk/v$VERSION/LICENSE" -o "$LICENSE_ROOT/GopherTrunk-LICENSE"
curl -fsSL "https://raw.githubusercontent.com/MattCheramie/GopherTrunk/v$VERSION/THIRD_PARTY_LICENSES.md" -o "$LICENSE_ROOT/GopherTrunk-THIRD_PARTY_LICENSES.md"
printf 'Bundled GopherTrunk P25 stack v%s is ready.\n' "$VERSION"
