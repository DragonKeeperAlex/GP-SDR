#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
sdk_dir="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$repo_dir/build/android-sdk}}"
gradle_bin="${GRADLE_BIN:-$repo_dir/build/android-tools/gradle-8.13/bin/gradle}"
gomobile_bin="${GOMOBILE_BIN:-$repo_dir/build/android-tools/gomobile}"
if [[ ! -x "$gomobile_bin" ]]; then
  echo "Install Go and gomobile, then set GOMOBILE_BIN to its path." >&2; exit 1
fi
if [[ ! -x "$gradle_bin" ]]; then
  echo "Install Gradle 8.13, then set GRADLE_BIN to its path." >&2; exit 1
fi
mkdir -p "$repo_dir/android/app/libs"
pushd "$repo_dir/server/mobilebridge" >/dev/null
PATH="$(dirname "$gomobile_bin"):$PATH" ANDROID_HOME="$sdk_dir" "$gomobile_bin" bind -target=android/arm64,android/arm -androidapi=28 -javapkg=org.gpsdr.engine -o "$repo_dir/android/app/libs/gpsdr-engine.aar" .
popd >/dev/null
ANDROID_HOME="$sdk_dir" "$gradle_bin" -p "$repo_dir/android" assembleDebug
echo "APK: $repo_dir/android/app/build/outputs/apk/debug/app-debug.apk"
