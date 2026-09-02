# Experimental standalone Android preview

The repository contains a standalone Android shell that runs the shared GP-SDR engine locally. **The 1.5.0-rc9 downloadable packages are for macOS, Windows, and Linux; Android remains an experimental source-build preview.** It is separate from opening a desktop host’s console in an Android browser.

## What is available

| Capability | Preview boundary |
| --- | --- |
| HackRF USB | Batched receive-only transport; 8–20 MS/s plus frequency, PPM, gain, amplifier, and antenna-power controls. Physical performance still needs setup-specific testing. |
| RTL-SDR | RTL-TCP networking; direct RTL-SDR USB is not included. |
| P25 | Native Android P25 is not included; use the desktop GP-SDR service for P25. |
| Interface | Shared Mapper/tuner, display settings, storage policy, channel import/export, and remote receiver support. |
| Transmit | Android HackRF transport is receive-only. |

## First run with a built preview

1. Before starting, open **Storage** and select removable SD storage. IQ, Recordings, and Exports use the app-owned SD directory; small settings and indexes stay internal.
2. Keep the card mounted. GP-SDR does not silently fall back to internal storage if it disappears. Changing storage does not move or delete existing captures.
3. Tap **Start**, then **USB**, and grant Android USB permission for each HackRF you intend to use.
4. Verify one known receive signal before starting a wide Mapper job. Monitor USB throughput, dropped data, power, and device temperature for your actual setup.
5. Stop reception before unplugging hardware or removing storage. The foreground receiver service keeps the device awake while active.

Android removes app-owned SD data when uninstalling the app. Export or back up important recordings before uninstalling or replacing the installation in a way that clears its data.

## Rendering performance

Choose Auto, Eco, Balanced, or Detail. Auto selects Eco on devices below 4 GB RAM. These modes change graph rendering only; receiver sample rate and Mapper concurrency remain separate settings. Lower those independently if capture or analysis cannot keep up.

## Building

The source instructions specify Android SDK API 35, NDK 27.2, Gradle 8.13, and Go 1.26+. Generate `android/app/libs/gpsdr-engine.aar` using `gomobile bind` from `server/mobilebridge`, then assemble the Android debug app. The repository also provides `Scripts/build-android.sh` as the build entry point. Consult those checked-in instructions for the exact toolchain and required setup; this wiki update did not build or test an APK.

The expected debug output is `android/app/build/outputs/apk/debug/app-debug.apk`. Direct USB availability, power, permissions, and decoder parity must be verified on the physical Android device; successful compilation is not RF acceptance.

Source: [Android README](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/android/README.md), [build script](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Scripts/build-android.sh), [release scope](https://github.com/DragonKeeperAlex/GP-SDR/blob/715de3b/Docs/RELEASE_NOTES_1.5.0-rc9.md).
