# Receive-only image media

Image media remains separate from packet decoders and from the low-latency
analog FPV receiver. These workflows produce image artifacts with provenance;
they do not create packet `DecoderMessage` records or protocol claims from RF
energy alone.

## IMG-01: NOAA APT / weather images

Initial scope is receive-only NOAA APT demodulation from demodulated PCM
produced by a supported VHF SDR capture or live IQ stream. A bounded first-pass
PCM-to-PNG extractor now exists in `internal/app/image_media.go`; it does not
yet claim live SDR integration, satellite identification, or physical image
acceptance. The implementation must identify the satellite pass
and channel, demodulate the APT audio/two-channel line structure, emit a PNG
and a sidecar manifest, and preserve the original capture when processing
fails. The manifest must include frequency, sample rate, receiver, antenna,
capture interval, processing version, image dimensions, line count, and a
quality summary.

Software acceptance requires a checked-in or generated deterministic fixture
with a known image hash, correct line count/dimensions, bounded processing time,
malformed-input rejection, and no packet-decoder event. Physical acceptance
requires a live pass and comparison against a reference image; it is not closed
by fixture decoding alone.

## IMG-02: digital SSTV

Initial scope is receive-only digital SSTV, with the first supported mode
chosen explicitly before implementation. Do not label an arbitrary modem or
audio recording as digital SSTV. The selected mode, symbol/raster format,
frequency/audio bandwidth, and compatible reference transmitter must be named
in the implementation and fixture manifest.

The output is an image artifact plus sidecar provenance and decode quality. The
workflow must reject incomplete or corrupted frames, retain partial output as
an explicitly incomplete artifact when useful, and keep image results separate
from packet decoder messages. Software acceptance requires a deterministic
known-image fixture and negative/noise fixtures. Physical acceptance requires
one received live frame and a visual comparison with the transmitted image.

## Shared image-media contract

- Receive-only by default; no transmitter control is added.
- No Mapper job is created solely to render an image.
- Original IQ/audio evidence is retained according to the selected capture
  policy until image processing succeeds or the user explicitly discards it.
- Image artifacts are written atomically and never replace an existing result.
- Every image records source, receiver, frequency, timing, decoder/mode,
  software revision, processing status, and failure reason when applicable.
- A generated PNG, changing frame counter, or successful process start is not
  image acceptance. Record lock/sync, complete frame/image output, and visual
  comparison separately.
