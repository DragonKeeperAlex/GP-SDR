#!/bin/sh
set -eu

if [ "$(uname -s)" != "Darwin" ]; then
  printf 'This helper is for macOS. Use the platform-specific How to links in GP-SDR.\n' >&2
  exit 1
fi

if ! command -v brew >/dev/null 2>&1; then
  printf 'Homebrew is required. Install it from https://brew.sh and run this helper again.\n' >&2
  exit 1
fi

GPSDR_BREW_PREFIX=$(brew --prefix)
if [ ! -w "$GPSDR_BREW_PREFIX/bin" ]; then
  printf 'The Homebrew bin directory is not writable: %s/bin\n' "$GPSDR_BREW_PREFIX" >&2
  exit 1
fi

brew install cmake pkgconf libsndfile pulseaudio codec2 ncurses sox dump1090-fa

GPSDR_BUILD_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-decoders.XXXXXX")
trap 'rm -rf "$GPSDR_BUILD_ROOT"' EXIT INT TERM

build_and_install() {
  source_directory=$1
  shift
  cmake -S "$source_directory" -B "$source_directory/build" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="$GPSDR_BREW_PREFIX" \
    -DCMAKE_PREFIX_PATH="$GPSDR_BREW_PREFIX" "$@"
  cmake --build "$source_directory/build" --parallel
  cmake --install "$source_directory/build"
}

git clone --depth 1 https://github.com/EliasOenal/multimon-ng.git "$GPSDR_BUILD_ROOT/multimon-ng"
build_and_install "$GPSDR_BUILD_ROOT/multimon-ng"

git clone --depth 1 https://github.com/f00b4r0/acarsdec.git "$GPSDR_BUILD_ROOT/acarsdec"
build_and_install "$GPSDR_BUILD_ROOT/acarsdec"

git clone --depth 1 https://github.com/jvde-github/AIS-catcher.git "$GPSDR_BUILD_ROOT/AIS-catcher"
build_and_install "$GPSDR_BUILD_ROOT/AIS-catcher"

git clone --branch ambe_tones --depth 1 https://github.com/lwvmobile/mbelib.git "$GPSDR_BUILD_ROOT/mbelib"
build_and_install "$GPSDR_BUILD_ROOT/mbelib"

git clone https://github.com/lwvmobile/dsd-fme.git "$GPSDR_BUILD_ROOT/dsd-fme"
build_and_install "$GPSDR_BUILD_ROOT/dsd-fme" \
  -DCMAKE_PREFIX_PATH="$GPSDR_BREW_PREFIX;$GPSDR_BREW_PREFIX/opt/ncurses" \
  -DCURSES_INCLUDE_PATH="$GPSDR_BREW_PREFIX/opt/ncurses/include" \
  -DCURSES_LIBRARY="$GPSDR_BREW_PREFIX/opt/ncurses/lib/libncursesw.dylib"

printf '\nInstalled decoder tools:\n'
for tool in dump1090 multimon-ng acarsdec AIS-catcher dsd-fme; do
  if command -v "$tool" >/dev/null 2>&1; then
    printf '  Ready: %s (%s)\n' "$tool" "$(command -v "$tool")"
  else
    printf '  Missing: %s\n' "$tool"
    exit 1
  fi
done

printf '\nOpen GP-SDR, go to Hardware, and press Refresh.\n'
