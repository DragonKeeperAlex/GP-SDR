#!/usr/bin/env python3
"""Bounded, receive-only OP25 isolation checks. Stop GP-SDR P25 first.

Reuses a saved generated config without editing profiles. Collects decoded
telemetry and PCM counts, never plays or saves call audio. These checks cannot
establish intelligibility or a verified live control channel when none decodes.
"""
import argparse
import copy
import datetime
import json
import pathlib
import signal
import socket
import subprocess
import time
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", required=True, type=pathlib.Path)
    parser.add_argument("--output-directory", required=True, type=pathlib.Path)
    parser.add_argument("--working-directory", required=True, type=pathlib.Path)
    parser.add_argument("--seconds", type=int, default=42)
    parser.add_argument("--indices", type=int, nargs="+", default=[2, 1, 0])
    args = parser.parse_args()
    if not 5 <= args.seconds <= 60:
        parser.error("each check must be between 5 and 60 seconds")
    base = json.loads(args.config.read_text())
    # Refuse to contend with a live decoder rather than create ambiguous evidence.
    with socket.socket() as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        probe.bind(("127.0.0.1", 8081))
    args.output_directory.mkdir(parents=True, exist_ok=True, mode=0o700)
    results = []
    for index in args.indices:
        device = base["devices"][index]
        if "c5cb" in device.get("args", "").lower():
            raise ValueError("excluded HackRF c5cb must not be tested")
        channels = [c for c in base["channels"] if c["device"] == device["name"]]
        if len(channels) != 1:
            raise ValueError("baseline requires exactly one channel per device")
        config = copy.deepcopy(base)
        config["devices"] = [copy.deepcopy(device)]
        config["channels"] = [copy.deepcopy(channels[0])]
        config["channels"][0]["destination"] = "udp://127.0.0.1:23456"
        config["channels"][0]["meta_stream_name"] = "stream_0"
        config["audio"]["instances"] = []
        folder = args.output_directory / f"receiver-{index}"
        folder.mkdir(mode=0o700)
        config_path = folder / "config.json"
        config_path.write_text(json.dumps(config, indent=2))
        config_path.chmod(0o600)
        result = {"configurationIndex": index, "name": device["name"],
                  "rateHz": device["rate"], "gains": device["gains"],
                  "startedAt": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  "telemetryReplies": 0, "controlDecoded": False,
                  "unencryptedAssignments": 0, "pcmPackets": 0, "pcmSamples": 0}
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as audio, (folder / "decoder.log").open("w") as log:
            audio.bind(("127.0.0.1", 23456))
            audio.setblocking(False)
            process = subprocess.Popen(["/usr/local/bin/multi_rx.py", "-c", str(config_path.resolve()), "-v", "2"],
                                       cwd=args.working_directory, stdout=log, stderr=log,
                                       stdin=subprocess.DEVNULL)
            start = time.monotonic()
            try:
                while time.monotonic() - start < args.seconds and process.poll() is None:
                    request = urllib.request.Request("http://127.0.0.1:8081/",
                        data=b'[{"command":"update","arg1":0,"arg2":0}]',
                        headers={"Content-Type": "application/json"})
                    try:
                        with urllib.request.urlopen(request, timeout=0.4) as response:
                            updates = json.load(response)
                        result["telemetryReplies"] += 1
                        (folder / "last-telemetry.json").write_text(json.dumps(updates))
                        for update in updates:
                            for key, state in update.items():
                                if not key.isdigit() or not isinstance(state, dict):
                                    continue
                                if update.get("json_type") == "trunk_update":
                                    last = state.get("last_tsbk") or 0
                                    if state.get("network_active") == 1 and 0 <= time.time() - last < 10:
                                        result["controlDecoded"] = True
                                        result["controlChannelHz"] = state.get("rxchan")
                                elif update.get("json_type") == "channel_update":
                                    if state.get("tgid") and not state.get("encrypted"):
                                        result["unencryptedAssignments"] += 1
                    except (OSError, ValueError):
                        pass
                    # Drain frequently enough to count transport without saving media.
                    until = time.monotonic() + 0.2
                    while time.monotonic() < until:
                        try:
                            packet = audio.recv(65536)
                            if len(packet) > 2 and len(packet) % 2 == 0:
                                result["pcmPackets"] += 1
                                result["pcmSamples"] += len(packet) // 2
                        except BlockingIOError:
                            time.sleep(0.01)
                result["unexpectedExit"] = process.poll()
            finally:
                if process.poll() is None:
                    process.send_signal(signal.SIGINT)
                try:
                    process.wait(timeout=4)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            result["elapsedSeconds"] = round(time.monotonic() - start, 2)
        results.append(result)
        (args.output_directory / "results.json").write_text(json.dumps(results, indent=2))
        print(json.dumps(result), flush=True)


if __name__ == "__main__":
    main()
