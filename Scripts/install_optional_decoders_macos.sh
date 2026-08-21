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

brew install cmake pkgconf libsndfile pulseaudio codec2 ncurses sox dump1090-fa rtl_433

GPSDR_BUILD_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-decoders.XXXXXX")
trap 'rm -rf "$GPSDR_BUILD_ROOT"' EXIT INT TERM

# Reproducible, reviewed upstream revisions for the v1.1 installer.
MULTIMON_REV=de0585926542687155852db502a9d2861e9acf95
ACARSDEC_REV=0b7ba27b18b30f8b703ceea9feb5bc107dfbc42e
AISCATCHER_REV=6023dd4f4e6d900db6064e416ba00fd2aa92ecd5
MBELIB_REV=30dc79074ca022366a27d705b8023011d9600339
DSDFME_REV=198f0eacb5ef3873fab23186640c90789152894c

clone_pinned() {
  repository=$1
  revision=$2
  destination=$3
  git init -q "$destination"
  git -C "$destination" remote add origin "$repository"
  git -C "$destination" fetch -q --depth 1 origin "$revision"
  git -C "$destination" checkout -q --detach FETCH_HEAD
}

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

clone_pinned https://github.com/EliasOenal/multimon-ng.git "$MULTIMON_REV" "$GPSDR_BUILD_ROOT/multimon-ng"
build_and_install "$GPSDR_BUILD_ROOT/multimon-ng"

clone_pinned https://github.com/f00b4r0/acarsdec.git "$ACARSDEC_REV" "$GPSDR_BUILD_ROOT/acarsdec"
build_and_install "$GPSDR_BUILD_ROOT/acarsdec"

clone_pinned https://github.com/jvde-github/AIS-catcher.git "$AISCATCHER_REV" "$GPSDR_BUILD_ROOT/AIS-catcher"
build_and_install "$GPSDR_BUILD_ROOT/AIS-catcher"

clone_pinned https://github.com/lwvmobile/mbelib.git "$MBELIB_REV" "$GPSDR_BUILD_ROOT/mbelib"
build_and_install "$GPSDR_BUILD_ROOT/mbelib"

clone_pinned https://github.com/lwvmobile/dsd-fme.git "$DSDFME_REV" "$GPSDR_BUILD_ROOT/dsd-fme"
build_and_install "$GPSDR_BUILD_ROOT/dsd-fme" \
  -DCMAKE_PREFIX_PATH="$GPSDR_BREW_PREFIX;$GPSDR_BREW_PREFIX/opt/ncurses" \
  -DCURSES_INCLUDE_PATH="$GPSDR_BREW_PREFIX/opt/ncurses/include" \
  -DCURSES_LIBRARY="$GPSDR_BREW_PREFIX/opt/ncurses/lib/libncursesw.dylib"

printf '\nInstalled decoder tools:\n'
for tool in dump1090 rtl_433 multimon-ng acarsdec AIS-catcher dsd-fme; do
  if command -v "$tool" >/dev/null 2>&1; then
    printf '  Ready: %s (%s)\n' "$tool" "$(command -v "$tool")"
  else
    printf '  Missing: %s\n' "$tool"
    exit 1
  fi
done

printf '\nOpen GP-SDR, go to Hardware, and press Refresh.\n'
