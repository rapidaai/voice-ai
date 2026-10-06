"""Pin NumPy 1.26 waveform-scaling output for deterministic float32 inputs."""

import hashlib
import json
from pathlib import Path

import numpy as np


if np.__version__ != "1.26.4":
    raise RuntimeError("use NumPy 1.26.4 to match the controlled Pipecat reference")

reference = {"numpy": np.__version__, "seed": 1337, "cases": []}
for pattern, lengths in (
    ("pcm", (0, 1, 2, 7, 8, 9, 15, 16, 17, 63, 64, 65, 120, 127, 128, 129, 135,
             255, 256, 257, 511, 512, 513, 4095, 4096, 4097, 8184, 8191, 8192, 8193,
             16383, 16384, 16385, 127999, 128000, 128001)),
    ("quiet_dc", (7, 129, 8191, 128000)),
    ("cancellation", (129, 8191, 128000)),
    ("constant", (128000,)),
    ("negative_zero", (128000,)),
):
    for length in lengths:
        samples = np.empty(length, dtype=np.float32)
        state = reference["seed"]
        for index in range(length):
            state = (1664525 * state + 1013904223) & 0xffffffff
            value = (state >> 16) - 32768
            if pattern == "pcm":
                samples[index] = np.float32(value) / np.float32(32768)
            elif pattern == "quiet_dc":
                samples[index] = np.float32(np.float32(value) / np.float32(1 << 30) + np.float32(0.125))
            elif pattern == "cancellation":
                samples[index] = (np.float32(1), np.float32(1e-7), np.float32(-1), np.float32(1e-7))[index % 4]
            elif pattern == "constant":
                samples[index] = np.float32(0.25)
            else:
                samples[index] = np.float32(-0.0)
        if length:
            output = (samples - samples.mean()) / np.sqrt(samples.var() + 1e-7)
        else:
            output = samples
        reference["cases"].append({"pattern": pattern, "samples": length,
                                   "input_sha256": hashlib.sha256(samples.astype('<f4').tobytes()).hexdigest(),
                                   "output_sha256": hashlib.sha256(output.astype('<f4').tobytes()).hexdigest()})

Path(__file__).with_name("reference.json").write_text(json.dumps(reference, indent=2) + "\n")
