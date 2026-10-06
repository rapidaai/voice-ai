"""Generate feature fixtures from an explicit local Pipecat checkout; requires NumPy."""

import argparse
import ast
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import platform
import struct
import subprocess
import sys

import numpy as np


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("checkout", type=Path)
args = parser.parse_args()
checkout = args.checkout.resolve()
source_dir = Path("src/pipecat/audio/turn/smart_turn")
feature_path = checkout / source_dir / "_whisper_features.py"
caller_path = checkout / source_dir / "local_smart_turn_v3.py"
spec = importlib.util.spec_from_file_location("whisper_reference", feature_path)
features = importlib.util.module_from_spec(spec)
spec.loader.exec_module(features)

# Execute only the caller's actual padding function, without ONNX or Pipecat imports.
caller = ast.parse(caller_path.read_text())
sample_rate = next(
    node for node in caller.body
    if isinstance(node, ast.Assign)
    and any(isinstance(target, ast.Name) and target.id == "_MODEL_SAMPLE_RATE"
            for target in node.targets)
)
analyzer = next(node for node in caller.body
                if isinstance(node, ast.ClassDef) and node.name == "LocalSmartTurnAnalyzerV3")
predict = next(node for node in analyzer.body
               if isinstance(node, ast.FunctionDef) and node.name == "_predict_endpoint")
prepare = next(node for node in predict.body
               if isinstance(node, ast.FunctionDef)
               and node.name == "truncate_audio_to_last_n_seconds")
namespace = {"np": np}
exec(compile(ast.Module(body=[sample_rate, prepare], type_ignores=[]),
             str(caller_path), "exec"), namespace)

short_time = np.arange(257) / 16000
pause_time = np.arange(64000) / 16000
long_time = np.arange(160137) / 16000
quiet_index = np.arange(128000, dtype=np.int64)
paused = 0.3 * np.sin(2 * np.pi * 173 * pause_time)
paused[36000:] = 0
cases = {
    "silence": np.zeros(16000, dtype=np.float32),
    "short": (0.11 + 0.25 * np.sin(2 * np.pi * 431 * short_time)).astype(np.float32),
    "pause": paused.astype(np.float32),
    "long": (0.2 * np.sin(2 * np.pi * (97 * long_time + 83 * long_time**2))
             + 0.1 * np.cos(2 * np.pi * 1801 * long_time)).astype(np.float32),
    "quiet_dc": (0.25 + ((quiet_index * 73 % 251) - 125) / 2**24).astype(np.float32),
    "constant": np.full(128000, 0.25, dtype=np.float32),
}
output_dir = Path(__file__).resolve().parent
metadata = {
    "format": "gzip: little-endian uint32 audio count, uint32 feature count, float32 audio, float32 row-major features",
    "reference_commit": subprocess.check_output(
        ["git", "-C", str(checkout), "rev-parse", "HEAD"], text=True).strip(),
    "source_sha256": {
        str(path.relative_to(checkout)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in (feature_path, caller_path)
    },
    "source_dirty": bool(subprocess.check_output(
        ["git", "-C", str(checkout), "status", "--porcelain", "--",
         str(feature_path.relative_to(checkout)), str(caller_path.relative_to(checkout))],
        text=True).strip()),
    "python": platform.python_version(),
    "python_executable": sys.executable,
    "numpy": np.__version__,
    "sample_rate": namespace["_MODEL_SAMPLE_RATE"],
    "feature_shape": [80, 800],
    "scope": "16 kHz mono float32; actual caller padding and feature function, no resampling or ONNX inference",
    "comparison": "Go test enforces 2e-5 absolute tolerance on these fixtures, not all possible inputs. Go matches NumPy 1.26 float32 reduction order and scalar promotion; NumPy 2 scalar promotion and FFT rounding can differ. No bit identity claim.",
    "regenerate": "PYTHONDONTWRITEBYTECODE=1 OPENBLAS_NUM_THREADS=1 python3 api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/features/generate.py /path/to/pipecat",
    "cases": [],
}
for name, audio in cases.items():
    prepared = namespace[prepare.name](audio)
    expected = features.compute_whisper_log_mel_features(prepared)
    assert expected.dtype == np.float32 and expected.shape == (80, 800)
    payload = (struct.pack("<II", audio.size, expected.size)
               + audio.astype("<f4").tobytes() + expected.astype("<f4").tobytes())
    compressed = gzip.compress(payload, compresslevel=9, mtime=0)
    filename = f"{name}.f32.gz"
    (output_dir / filename).write_bytes(compressed)
    metadata["cases"].append({
        "name": name,
        "file": filename,
        "audio_samples": int(audio.size),
        "sha256": hashlib.sha256(compressed).hexdigest(),
    })
(output_dir / "reference.json").write_text(json.dumps(metadata, indent=2) + "\n")
