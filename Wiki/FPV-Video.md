# FPV video

GP-SDR includes a dedicated low-latency receiver for conventional analog NTSC and PAL FPV video. It is separate from Mapper and does not create Mapper jobs or retained IQ captures.

## Start a receiver

1. Connect a receiver and open **FPV video**.
2. Pick the receiver. GP-SDR lists local receivers that expose a continuous IQ source.
3. Choose a Raceband or Band A preset, or enter a frequency manually.
4. Select NTSC or PAL and start with **16 MS/s**.
5. Start video. The status changes to **Receiving** while GP-SDR waits for video synchronization.

The receiver must cover the chosen frequency. RTL-SDR models normally cannot tune to the 5.8 GHz FPV band, but they remain usable for analog video within their reported range. HackRF and suitably configured PlutoSDR hardware can cover 5.8 GHz.

## Receiver controls

Controls follow the selected hardware. HackRF exposes LNA, VGA, RF amplifier, and DC removal. Other receivers show only controls that apply to them. Begin with the RF amplifier off; excessive gain can erase synchronization and produce a worse picture.

Use **Fullscreen** after stable frames appear. PAL is common on some imported cameras; switch standards if the picture never synchronizes or has the wrong geometry.

## Decoder component

The native macOS build uses the separately installed GPL-licensed FPV Viewer decoder. GP-SDR reports a clear setup error if that component is unavailable. Digital DJI, Walksnail, and HDZero links are not decoded by the analog receiver.

## PlutoSDR USB and Ethernet

When the driver advertises both paths, Hardware shows a **Connection** selector. Pick USB or Ethernet/network before assigning the Pluto to Tuner, Mapper, Spectrum Analyzer, or FPV. GP-SDR preserves the exact libiio URI reported by SoapySDR for its normal receive and transmit streams.
