#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${1:-1.0.12-dev}
BUNDLE_VERSION=$(printf '%s' "$VERSION" | sed 's/[^0-9.].*$//')
if [ -z "$BUNDLE_VERSION" ]; then BUNDLE_VERSION=1.0.12; fi
BUILD_ROOT="$PROJECT_ROOT/build/release"
DIST_ROOT="$PROJECT_ROOT/dist"
SERVER_ROOT="$PROJECT_ROOT/server"
APP_BUILD_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-apps.XXXXXX")
CONFLICT_LIST=$(mktemp "${TMPDIR:-/tmp}/gpsdr-conflicts.XXXXXX")
trap 'rm -rf "$APP_BUILD_ROOT"; rm -f "$CONFLICT_LIST"' EXIT INT TERM

find "$PROJECT_ROOT" \
  \( -path "$PROJECT_ROOT/.git" -o -path "$PROJECT_ROOT/build" -o -path "$PROJECT_ROOT/dist" \) -prune -o \
  -type f \( -name '* 2.*' -o -name '*.icloud-conflict*' \) -print > "$CONFLICT_LIST"
if [ -s "$CONFLICT_LIST" ]; then
  echo "Release stopped: cloud-conflict source copies would make the build ambiguous." >&2
  cat "$CONFLICT_LIST" >&2
  echo "Move these copies outside the repository, review them, and run the build again." >&2
  exit 1
fi

rm -rf "$BUILD_ROOT" "$DIST_ROOT" 2>/dev/null || true
mkdir -p "$BUILD_ROOT/bin" "$DIST_ROOT"
# Finder can recreate .DS_Store between removal of a directory and its parent.
# Remove any remaining real build content while tolerating that harmless race.
find "$BUILD_ROOT" "$DIST_ROOT" -mindepth 1 ! -name '.DS_Store' -exec rm -rf {} +
find "$BUILD_ROOT" "$DIST_ROOT" -name '.DS_Store' -delete 2>/dev/null || true
mkdir -p "$BUILD_ROOT/bin" "$DIST_ROOT"
chmod +x "$PROJECT_ROOT/Scripts/fetch_p25_stack.sh"
"$PROJECT_ROOT/Scripts/fetch_p25_stack.sh"

build_go() {
  target_os=$1
  target_arch=$2
  output=$3
  (cd "$SERVER_ROOT" && CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath \
    -ldflags "-s -w -X gpsdr.local/gpsdr/internal/app.Version=$VERSION" -o "$output" .)
}

build_go darwin arm64 "$BUILD_ROOT/bin/gpsdr-server-darwin-arm64"
build_go darwin amd64 "$BUILD_ROOT/bin/gpsdr-server-darwin-amd64"
build_go linux arm64 "$BUILD_ROOT/bin/gpsdr-server-linux-arm64"
build_go linux amd64 "$BUILD_ROOT/bin/gpsdr-server-linux-amd64"
build_go windows amd64 "$BUILD_ROOT/bin/gpsdr-server-windows-amd64.exe"

build_shell() {
  architecture=$1
  output=$2
  log_file="$BUILD_ROOT/native-shell-$architecture.log"
  installed_shell="$HOME/Applications/GP-SDR.app/Contents/MacOS/GP-SDR"
  if [ -f "$installed_shell" ] && git -C "$PROJECT_ROOT" diff --quiet -- macos/GPSDRApp.m macos/GPSDRApp.swift && lipo "$installed_shell" -verify_arch "$architecture" >/dev/null 2>&1; then
    lipo "$installed_shell" -thin "$architecture" -output "$output"
    return
  fi
  if xcrun clang -fobjc-arc -O2 -target "$architecture-apple-macos13.0" \
    -framework AppKit -framework CoreLocation -framework WebKit "$PROJECT_ROOT/macos/GPSDRApp.m" -o "$output" 2>"$log_file"; then
    return
  fi
  if [ -f "$installed_shell" ] && lipo "$installed_shell" -verify_arch "$architecture" >/dev/null 2>&1; then
    echo "Native shell toolchain could not build $architecture; reusing the unchanged installed GP-SDR shell."
    lipo "$installed_shell" -thin "$architecture" -output "$output"
    return
  fi
  cat "$log_file" >&2
  return 1
}

build_shell arm64 "$BUILD_ROOT/bin/GP-SDR-shell-arm64"
build_shell x86_64 "$BUILD_ROOT/bin/GP-SDR-shell-amd64"

mkdir -p "$PROJECT_ROOT/build/helpers/darwin-arm64" "$PROJECT_ROOT/build/helpers/darwin-amd64"
xcrun clang -fobjc-arc -O2 -target arm64-apple-macos13.0 -framework Foundation \
  "$PROJECT_ROOT/macos/GPSDRPreferences.m" \
  -o "$PROJECT_ROOT/build/helpers/darwin-arm64/gpsdr-mac-prefs"
xcrun clang -fobjc-arc -O2 -target x86_64-apple-macos13.0 -framework Foundation \
  "$PROJECT_ROOT/macos/GPSDRPreferences.m" \
  -o "$PROJECT_ROOT/build/helpers/darwin-amd64/gpsdr-mac-prefs"
xcrun clang++ -std=c++17 -O2 -target arm64-apple-macos13.0 \
  "$PROJECT_ROOT/helpers/soapy_capture/main.cpp" \
  -o "$PROJECT_ROOT/build/helpers/darwin-arm64/gpsdr-soapy"
xcrun clang++ -std=c++17 -O2 -target x86_64-apple-macos13.0 \
  "$PROJECT_ROOT/helpers/soapy_capture/main.cpp" \
  -o "$PROJECT_ROOT/build/helpers/darwin-amd64/gpsdr-soapy"

lipo -create "$BUILD_ROOT/bin/gpsdr-server-darwin-arm64" \
  "$BUILD_ROOT/bin/gpsdr-server-darwin-amd64" \
  -output "$BUILD_ROOT/bin/gpsdr-server-darwin-universal"
lipo -create "$BUILD_ROOT/bin/GP-SDR-shell-arm64" \
  "$BUILD_ROOT/bin/GP-SDR-shell-amd64" \
  -output "$BUILD_ROOT/bin/GP-SDR-shell-universal"
lipo -create "$PROJECT_ROOT/build/helpers/darwin-arm64/gpsdr-mac-prefs" \
  "$PROJECT_ROOT/build/helpers/darwin-amd64/gpsdr-mac-prefs" \
  -output "$BUILD_ROOT/bin/gpsdr-mac-prefs-darwin-universal"
if [ -f "$PROJECT_ROOT/build/helpers/darwin-arm64/gpsdr-soapy" ] && \
   [ -f "$PROJECT_ROOT/build/helpers/darwin-amd64/gpsdr-soapy" ]; then
  lipo -create "$PROJECT_ROOT/build/helpers/darwin-arm64/gpsdr-soapy" \
    "$PROJECT_ROOT/build/helpers/darwin-amd64/gpsdr-soapy" \
    -output "$BUILD_ROOT/bin/gpsdr-soapy-darwin-universal"
fi

make_mac_app() {
  architecture=$1
  shell_binary=$2
  server_binary=$3
  app_root="$APP_BUILD_ROOT/GP-SDR-$architecture/GP-SDR.app"
  mkdir -p "$app_root/Contents/MacOS" "$app_root/Contents/Resources/bin" "$app_root/Contents/Resources/Documentation"
  cp "$PROJECT_ROOT/packaging/macos/Info.plist" "$app_root/Contents/Info.plist"
  cp "$PROJECT_ROOT/packaging/macos/GP-SDR.icns" "$app_root/Contents/Resources/GP-SDR.icns"
  /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $BUNDLE_VERSION" "$app_root/Contents/Info.plist"
  cp "$shell_binary" "$app_root/Contents/MacOS/GP-SDR"
  cp "$server_binary" "$app_root/Contents/Resources/bin/gpsdr-server"
  cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$app_root/Contents/Resources/Documentation/"
  cp "$PROJECT_ROOT/build/p25-licenses/"* "$app_root/Contents/Resources/Documentation/"
  helper_tag=$architecture
  if [ "$architecture" = "x86_64" ]; then helper_tag=amd64; fi
  if [ "$architecture" = "universal" ]; then
    helper_path="$BUILD_ROOT/bin/gpsdr-soapy-darwin-universal"
    prefs_helper_path="$BUILD_ROOT/bin/gpsdr-mac-prefs-darwin-universal"
    cp -R "$PROJECT_ROOT/build/components/sdrtrunk/darwin-arm64" "$app_root/Contents/Resources/sdrtrunk-arm64"
    cp -R "$PROJECT_ROOT/build/components/sdrtrunk/darwin-amd64" "$app_root/Contents/Resources/sdrtrunk-amd64"
    cp -R "$PROJECT_ROOT/build/components/jmbe-creator/darwin-arm64" "$app_root/Contents/Resources/jmbe-creator-arm64"
    cp -R "$PROJECT_ROOT/build/components/jmbe-creator/darwin-amd64" "$app_root/Contents/Resources/jmbe-creator-amd64"
  else
    helper_path="$PROJECT_ROOT/build/helpers/darwin-$helper_tag/gpsdr-soapy"
    prefs_helper_path="$PROJECT_ROOT/build/helpers/darwin-$helper_tag/gpsdr-mac-prefs"
    cp -R "$PROJECT_ROOT/build/components/sdrtrunk/darwin-$helper_tag" "$app_root/Contents/Resources/sdrtrunk-$helper_tag"
    cp -R "$PROJECT_ROOT/build/components/jmbe-creator/darwin-$helper_tag" "$app_root/Contents/Resources/jmbe-creator-$helper_tag"
  fi
  if [ -f "$helper_path" ]; then
    cp "$helper_path" "$app_root/Contents/Resources/bin/gpsdr-soapy"
    chmod 755 "$app_root/Contents/Resources/bin/gpsdr-soapy"
  fi
  cp "$prefs_helper_path" "$app_root/Contents/Resources/bin/gpsdr-mac-prefs"
  chmod 755 "$app_root/Contents/Resources/bin/gpsdr-mac-prefs"
  chmod 755 "$app_root/Contents/MacOS/GP-SDR" "$app_root/Contents/Resources/bin/gpsdr-server"
  find "$app_root/Contents/Resources" -path '*/bin/*' -type f -exec chmod 755 {} \;
  xattr -cr "$app_root"
  codesign --force --sign - "$app_root/Contents/MacOS/GP-SDR"
  codesign --force --sign - "$app_root/Contents/Resources/bin/gpsdr-server"
  if [ -f "$app_root/Contents/Resources/bin/gpsdr-soapy" ]; then
    codesign --force --sign - "$app_root/Contents/Resources/bin/gpsdr-soapy"
  fi
  codesign --force --sign - "$app_root/Contents/Resources/bin/gpsdr-mac-prefs"
  codesign --force --deep --sign - "$app_root"
  ditto --norsrc -c -k --keepParent "$app_root" "$DIST_ROOT/GP-SDR-$VERSION-macos-$architecture.zip"
}

make_mac_app arm64 "$BUILD_ROOT/bin/GP-SDR-shell-arm64" "$BUILD_ROOT/bin/gpsdr-server-darwin-arm64"
make_mac_app x86_64 "$BUILD_ROOT/bin/GP-SDR-shell-amd64" "$BUILD_ROOT/bin/gpsdr-server-darwin-amd64"
make_mac_app universal "$BUILD_ROOT/bin/GP-SDR-shell-universal" "$BUILD_ROOT/bin/gpsdr-server-darwin-universal"

make_deb() {
  architecture=$1
  binary=$2
  deb_root="$BUILD_ROOT/deb-$architecture"
  package_root="$deb_root/root"
  control_root="$deb_root/control"
  mkdir -p "$package_root/usr/bin" "$package_root/usr/lib/gp-sdr" "$package_root/lib/systemd/system" "$package_root/usr/share/doc/gp-sdr" "$control_root"
  cp "$binary" "$package_root/usr/bin/gp-sdr"
  chmod 755 "$package_root/usr/bin/gp-sdr"
  cp -R "$PROJECT_ROOT/build/components/sdrtrunk/linux-$architecture" "$package_root/usr/lib/gp-sdr/sdrtrunk"
  cp -R "$PROJECT_ROOT/build/components/jmbe-creator/linux-$architecture" "$package_root/usr/lib/gp-sdr/jmbe-creator"
  cp "$PROJECT_ROOT/packaging/linux/gp-sdr.service" "$package_root/lib/systemd/system/gp-sdr.service"
  cp "$PROJECT_ROOT/README.md" "$package_root/usr/share/doc/gp-sdr/README.md"
  cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$package_root/usr/share/doc/gp-sdr/"
  cp "$PROJECT_ROOT/build/p25-licenses/"* "$package_root/usr/share/doc/gp-sdr/"
  find "$package_root/usr/lib/gp-sdr" -path '*/bin/*' -type f -exec chmod 755 {} \;
  linux_helper="$PROJECT_ROOT/build/helpers/linux-$architecture/gpsdr-soapy"
  if [ -f "$linux_helper" ]; then
    cp "$linux_helper" "$package_root/usr/bin/gpsdr-soapy"
    chmod 755 "$package_root/usr/bin/gpsdr-soapy"
  fi
  sed -e "s/@VERSION@/$VERSION/g" -e "s/@ARCH@/$architecture/g" "$PROJECT_ROOT/packaging/linux/control.in" > "$control_root/control"
  package_file="$DIST_ROOT/gp-sdr_${VERSION}_${architecture}.deb"
  if command -v dpkg-deb >/dev/null 2>&1; then
    mkdir -p "$package_root/DEBIAN"
    cp "$control_root/control" "$package_root/DEBIAN/control"
    dpkg-deb --build --root-owner-group "$package_root" "$package_file"
  else
    printf '2.0\n' > "$deb_root/debian-binary"
    COPYFILE_DISABLE=1 tar -C "$control_root" -czf "$deb_root/control.tar.gz" control
    COPYFILE_DISABLE=1 tar -C "$package_root" -czf "$deb_root/data.tar.gz" .
    (cd "$deb_root" && ar -qS "$package_file" debian-binary control.tar.gz data.tar.gz)
  fi
}

make_deb amd64 "$BUILD_ROOT/bin/gpsdr-server-linux-amd64"
make_deb arm64 "$BUILD_ROOT/bin/gpsdr-server-linux-arm64"

windows_root="$BUILD_ROOT/GP-SDR-windows-x86_64"
mkdir -p "$windows_root"
cp "$BUILD_ROOT/bin/gpsdr-server-windows-amd64.exe" "$windows_root/GP-SDR.exe"
cp -R "$PROJECT_ROOT/build/components/sdrtrunk/windows-amd64" "$windows_root/sdrtrunk"
cp -R "$PROJECT_ROOT/build/components/jmbe-creator/windows-amd64" "$windows_root/jmbe-creator"
cp "$PROJECT_ROOT/packaging/windows/README.txt" "$windows_root/README.txt"
cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$windows_root/"
cp "$PROJECT_ROOT/build/p25-licenses/"* "$windows_root/"
if [ -f "$PROJECT_ROOT/build/helpers/windows-amd64/gpsdr-soapy.exe" ]; then
  cp "$PROJECT_ROOT/build/helpers/windows-amd64/gpsdr-soapy.exe" "$windows_root/gpsdr-soapy.exe"
fi
(cd "$windows_root" && zip -q -r "$DIST_ROOT/GP-SDR-$VERSION-windows-x86_64.zip" .)

(cd "$DIST_ROOT" && {
  : > SHA256SUMS.txt
  for artifact in ./*; do
    [ "$artifact" = "./SHA256SUMS.txt" ] && continue
    shasum -a 256 "$artifact" >> SHA256SUMS.txt
  done
})
printf 'Release artifacts created in %s\n' "$DIST_ROOT"
