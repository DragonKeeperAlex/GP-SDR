# GP-SDR Android preview

This standalone Android shell serves the shared GP-SDR engine locally. Mapper, tuner controls, spectrum/waterfall settings, storage policy, channel import/export, and RTL-TCP receivers retain the desktop behavior. A foreground receiver service prevents sleep while it is active.

## Storage

Open **Storage** before starting and select the removable SD card. Small settings and Mapper indexes remain internal; `IQ`, `Recordings`, and `Exports` are redirected to the app-owned SD directory. The card must stay mounted during capture; GP-SDR does not silently fall back to internal storage. Existing captures are never moved or deleted when changing selection. Android removes app-owned SD data on uninstall, so export anything important.

## Receiver support

Tap **Start**, then **USB** and grant Android permission for each HackRF. The preview includes a batched receive-only HackRF USB transport (8–20 MS/s, frequency, PPM, LNA/VGA, amplifier, and antenna-power controls) and RTL-TCP networking. Direct RTL-SDR USB and the native P25 decoder are not included in this preview; use RTL-TCP and the desktop GP-SDR P25 service until the Android native P25 port is completed. The UI reports these limits instead of showing non-working controls.

## Performance and build

Performance offers Auto, Eco, Balanced, and Detail modes. They change graph rendering only; sample rate and Mapper concurrency remain independent. Auto chooses Eco below 4 GB RAM.

With Android SDK API 35, NDK 27.2, Gradle 8.13, and Go 1.26+, generate `app/libs/gpsdr-engine.aar` with `gomobile bind` from `server/mobilebridge`, then run `gradle -p android assembleDebug`. The APK is `android/app/build/outputs/apk/debug/app-debug.apk`.
