#!/usr/bin/env python3
"""Verify hardware IQ reaches the installed OP25 demodulator/frame assembler.

Stop GP-SDR P25 first. Runs the installed engine's real flowgraph briefly with
the saved receiver config. Checks a bounded IQ sample and advancing assembler
input streams, without requiring valid P25 frames. Never saves IQ or audio.
"""
import argparse
import contextlib
import importlib.util
import json
import pathlib
import socket
import sys
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", required=True, type=pathlib.Path)
    parser.add_argument("--device-name", required=True, help="One device name from the saved config; tests it independently")
    parser.add_argument("--apps", type=pathlib.Path, default=pathlib.Path("/usr/local/lib/gp-sdr/op25/apps"))
    parser.add_argument("--seconds", type=float, default=2)
    args = parser.parse_args()
    if not 1 <= args.seconds <= 8:
        parser.error("observation must be between 1 and 8 seconds")
    config = json.loads(args.config.read_text())
    if any("c5cb" in d.get("args", "").lower() for d in config["devices"]):
        parser.error("excluded HackRF c5cb must not be tested")
    config["devices"] = [d for d in config["devices"] if d["name"] == args.device_name]
    config["channels"] = [ch for ch in config["channels"] if ch["device"] == args.device_name]
    if len(config["devices"]) != 1 or len(config["channels"]) != 1:
        parser.error("select exactly one configured device and channel")
    if any("hackrf" in d.get("args", "").lower() and d.get("rate", 0) < 8_000_000 for d in config["devices"]):
        parser.error("HackRF input checks require at least 8 MS/s")
    with socket.socket() as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        probe.bind(("127.0.0.1", 8081))
    for path in [args.apps, *[args.apps / p for p in ("tx", "tdma", "corr", "util")]]:
        sys.path.insert(0, str(path))
    config.pop("terminal", None)
    config["audio"]["instances"] = []
    for channel in config["channels"]:
        channel.pop("raw_output", None)
        channel.pop("raw_input", None)
        channel["plot"] = ""
    with contextlib.redirect_stdout(sys.stderr):
        import numpy as np
        from gnuradio import gr

        spec = importlib.util.spec_from_file_location("gpsdr_input_probe_engine", args.apps / "multi_rx.py")
        engine = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(engine)
        tb = engine.rx_block(0, config)
        class StreamProbe(gr.sync_block):
            def __init__(self, dtype):
                gr.sync_block.__init__(self, name="GP-SDR bounded input check", in_sig=[dtype], out_sig=[dtype])
                self.count = 0
                self.sample = np.empty(0, dtype=dtype)

            def work(self, inputs, outputs):
                data = inputs[0]
                outputs[0][:] = data
                remaining = 8192 - len(self.sample)
                if remaining > 0:
                    self.sample = np.concatenate((self.sample, data[:remaining].copy()))
                self.count += len(data)
                return len(data)

        probes = []
        for channel in tb.channels:
            iq = StreamProbe(np.complex64)
            symbols = StreamProbe(np.uint8)
            # Inline probes forward the unchanged stream into the real
            # demodulator and frame assembler, including undecodable noise.
            tb.disconnect(channel.device.src, channel.demod)
            tb.disconnect(channel.demod, channel.decoder)
            tb.connect(channel.device.src, iq, channel.demod, symbols, channel.decoder)
            probes.append((channel, iq, symbols))
        if len(probes) != len(config["channels"]):
            raise RuntimeError("OP25 did not construct every configured channel")
        results = []
        tb.start()
        try:
            time.sleep(args.seconds)
            before = [(iq.count, symbols.count) for _, iq, symbols in probes]
            time.sleep(args.seconds)
            for index, (ch, iq, symbols) in enumerate(probes):
                samples = iq.sample.copy()
                source_count = iq.count
                decoder_count = symbols.count
                finite = bool(len(samples) and np.isfinite(samples).all())
                variance = float(np.var(samples)) if finite else 0.0
                result = {
                    "channel": ch.config["name"], "device": ch.device.name,
                    "actualRateHz": float(ch.device.sample_rate),
                    "iqSamplesChecked": len(samples), "iqVariance": variance,
                    "sourceSamplesAdvanced": source_count - before[index][0],
                    "decoderSymbolsAdvanced": decoder_count - before[index][1],
                    "decoderInputSymbols": decoder_count,
                }
                result["passed"] = finite and len(samples) == 8192 and variance > 0 and result["sourceSamplesAdvanced"] > 0 and result["decoderSymbolsAdvanced"] > 0
                results.append(result)
        finally:
            tb.stop()
            tb.wait()
            tb.kill()
    print(json.dumps({"receivers": results, "passed": bool(results) and all(r["passed"] for r in results)}, indent=2))
    return 0 if results and all(r["passed"] for r in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
