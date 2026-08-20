#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=0.9.9
COMMIT=09b00149d289d9db2867a6d166bc4d71ea912503
SOURCE_URL=https://github.com/MattCheramie/GopherTrunk.git
HELPER_ROOT="$PROJECT_ROOT/build/helpers"
LICENSE_ROOT="$PROJECT_ROOT/build/p25-licenses"
PATCH_FILE="$PROJECT_ROOT/third_party/patches/gophertrunk-opensourcesdrlab-offset-binary.patch"
SOURCE_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-gophertrunk.XXXXXX")
trap 'rm -rf "$SOURCE_ROOT"' EXIT INT TERM

mkdir -p "$HELPER_ROOT" "$LICENSE_ROOT"
git clone --quiet --depth 1 --branch "v$VERSION" "$SOURCE_URL" "$SOURCE_ROOT"
actual_commit=$(git -C "$SOURCE_ROOT" rev-parse HEAD)
if [ "$actual_commit" != "$COMMIT" ]; then
  printf 'GopherTrunk source commit mismatch: expected %s, got %s\n' "$COMMIT" "$actual_commit" >&2
  exit 1
fi
git -C "$SOURCE_ROOT" apply --check "$PATCH_FILE"
git -C "$SOURCE_ROOT" apply "$PATCH_FILE"
(cd "$SOURCE_ROOT" && go test ./internal/sdr/hackrf)

build_target() {
  target_os=$1
  target_arch=$2
  target_dir=$3
  binary_name=$4
  mkdir -p "$HELPER_ROOT/$target_dir"
  (cd "$SOURCE_ROOT" && CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath \
    -ldflags "-s -w -X github.com/MattCheramie/GopherTrunk/internal/version.Version=v$VERSION-gpsdr1 -X github.com/MattCheramie/GopherTrunk/internal/version.Commit=$COMMIT" \
    -o "$HELPER_ROOT/$target_dir/$binary_name" ./cmd/gophertrunk)
  chmod 755 "$HELPER_ROOT/$target_dir/$binary_name"
}

build_target darwin arm64 darwin-arm64 gophertrunk
build_target darwin amd64 darwin-amd64 gophertrunk
build_target linux amd64 linux-amd64 gophertrunk
build_target linux arm64 linux-arm64 gophertrunk
build_target windows amd64 windows-amd64 gophertrunk.exe

cp "$SOURCE_ROOT/LICENSE" "$LICENSE_ROOT/GopherTrunk-LICENSE"
cp "$SOURCE_ROOT/THIRD_PARTY_LICENSES.md" "$LICENSE_ROOT/GopherTrunk-THIRD_PARTY_LICENSES.md"
printf 'Bundled GopherTrunk P25 stack v%s with GP-SDR OpenSourceSDRLab compatibility patch is ready.\n' "$VERSION"
