#!/bin/sh
set -eu

PROJECT_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION=${1:-1.0.0-dev}
BUNDLE_VERSION=$(printf '%s' "$VERSION" | sed 's/[^0-9.].*$//')
if [ -z "$BUNDLE_VERSION" ]; then BUNDLE_VERSION=1.0.0; fi
BUILD_ROOT="$PROJECT_ROOT/build/release"
DIST_ROOT="$PROJECT_ROOT/dist"
SERVER_ROOT="$PROJECT_ROOT/server"
APP_BUILD_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/gpsdr-apps.XXXXXX")
trap 'rm -rf "$APP_BUILD_ROOT"' EXIT INT TERM

rm -rf "$BUILD_ROOT" "$DIST_ROOT"
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

xcrun swiftc -swift-version 5 -parse-as-library -O -target arm64-apple-macos13.0 \
  -framework AppKit -framework WebKit "$PROJECT_ROOT/macos/GPSDRApp.swift" \
  -o "$BUILD_ROOT/bin/GP-SDR-shell-arm64"
xcrun swiftc -swift-version 5 -parse-as-library -O -target x86_64-apple-macos13.0 \
  -framework AppKit -framework WebKit "$PROJECT_ROOT/macos/GPSDRApp.swift" \
  -o "$BUILD_ROOT/bin/GP-SDR-shell-amd64"

mkdir -p "$PROJECT_ROOT/build/helpers/darwin-arm64" "$PROJECT_ROOT/build/helpers/darwin-amd64"
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
lipo -create "$PROJECT_ROOT/build/helpers/darwin-arm64/gophertrunk" \
  "$PROJECT_ROOT/build/helpers/darwin-amd64/gophertrunk" \
  -output "$BUILD_ROOT/bin/gophertrunk-darwin-universal"

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
  p25_binary=$4
  app_root="$APP_BUILD_ROOT/GP-SDR-$architecture/GP-SDR.app"
  mkdir -p "$app_root/Contents/MacOS" "$app_root/Contents/Resources/bin" "$app_root/Contents/Resources/Documentation"
  cp "$PROJECT_ROOT/packaging/macos/Info.plist" "$app_root/Contents/Info.plist"
  cp "$PROJECT_ROOT/packaging/macos/GP-SDR.icns" "$app_root/Contents/Resources/GP-SDR.icns"
  /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $BUNDLE_VERSION" "$app_root/Contents/Info.plist"
  cp "$shell_binary" "$app_root/Contents/MacOS/GP-SDR"
  cp "$server_binary" "$app_root/Contents/Resources/bin/gpsdr-server"
  cp "$p25_binary" "$app_root/Contents/Resources/bin/gophertrunk"
  cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$app_root/Contents/Resources/Documentation/"
  cp "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-LICENSE" "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-THIRD_PARTY_LICENSES.md" "$app_root/Contents/Resources/Documentation/"
  helper_tag=$architecture
  if [ "$architecture" = "x86_64" ]; then helper_tag=amd64; fi
  if [ "$architecture" = "universal" ]; then
    helper_path="$BUILD_ROOT/bin/gpsdr-soapy-darwin-universal"
  else
    helper_path="$PROJECT_ROOT/build/helpers/darwin-$helper_tag/gpsdr-soapy"
  fi
  if [ -f "$helper_path" ]; then
    cp "$helper_path" "$app_root/Contents/Resources/bin/gpsdr-soapy"
    chmod 755 "$app_root/Contents/Resources/bin/gpsdr-soapy"
  fi
  chmod 755 "$app_root/Contents/MacOS/GP-SDR" "$app_root/Contents/Resources/bin/gpsdr-server" "$app_root/Contents/Resources/bin/gophertrunk"
  xattr -cr "$app_root"
  codesign --force --deep --sign - "$app_root"
  ditto --norsrc -c -k --keepParent "$app_root" "$DIST_ROOT/GP-SDR-$VERSION-macos-$architecture.zip"
}

make_mac_app arm64 "$BUILD_ROOT/bin/GP-SDR-shell-arm64" "$BUILD_ROOT/bin/gpsdr-server-darwin-arm64" "$PROJECT_ROOT/build/helpers/darwin-arm64/gophertrunk"
make_mac_app x86_64 "$BUILD_ROOT/bin/GP-SDR-shell-amd64" "$BUILD_ROOT/bin/gpsdr-server-darwin-amd64" "$PROJECT_ROOT/build/helpers/darwin-amd64/gophertrunk"
make_mac_app universal "$BUILD_ROOT/bin/GP-SDR-shell-universal" "$BUILD_ROOT/bin/gpsdr-server-darwin-universal" "$BUILD_ROOT/bin/gophertrunk-darwin-universal"

make_deb() {
  architecture=$1
  binary=$2
  deb_root="$BUILD_ROOT/deb-$architecture"
  package_root="$deb_root/root"
  control_root="$deb_root/control"
  mkdir -p "$package_root/usr/bin" "$package_root/lib/systemd/system" "$package_root/usr/share/doc/gp-sdr" "$control_root"
  cp "$binary" "$package_root/usr/bin/gp-sdr"
  chmod 755 "$package_root/usr/bin/gp-sdr"
  cp "$PROJECT_ROOT/build/helpers/linux-$architecture/gophertrunk" "$package_root/usr/bin/gophertrunk"
  chmod 755 "$package_root/usr/bin/gophertrunk"
  cp "$PROJECT_ROOT/packaging/linux/gp-sdr.service" "$package_root/lib/systemd/system/gp-sdr.service"
  cp "$PROJECT_ROOT/README.md" "$package_root/usr/share/doc/gp-sdr/README.md"
  cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$package_root/usr/share/doc/gp-sdr/"
  cp "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-LICENSE" "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-THIRD_PARTY_LICENSES.md" "$package_root/usr/share/doc/gp-sdr/"
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
cp "$PROJECT_ROOT/build/helpers/windows-amd64/gophertrunk.exe" "$windows_root/gophertrunk.exe"
cp "$PROJECT_ROOT/packaging/windows/README.txt" "$windows_root/README.txt"
cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/NOTICE" "$PROJECT_ROOT/THIRD_PARTY.md" "$windows_root/"
cp "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-LICENSE" "$PROJECT_ROOT/build/p25-licenses/GopherTrunk-THIRD_PARTY_LICENSES.md" "$windows_root/"
if [ -f "$PROJECT_ROOT/build/helpers/windows-amd64/gpsdr-soapy.exe" ]; then
  cp "$PROJECT_ROOT/build/helpers/windows-amd64/gpsdr-soapy.exe" "$windows_root/gpsdr-soapy.exe"
fi
(cd "$windows_root" && zip -q -r "$DIST_ROOT/GP-SDR-$VERSION-windows-x86_64.zip" .)

(cd "$DIST_ROOT" && shasum -a 256 ./* > SHA256SUMS.txt)
printf 'Release artifacts created in %s\n' "$DIST_ROOT"
