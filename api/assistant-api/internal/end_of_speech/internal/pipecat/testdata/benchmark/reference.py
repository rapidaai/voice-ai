"""Benchmark the local Pipecat analyzer with explicit audio, model, and runtime provenance."""

import argparse
import ast
import gzip
import hashlib
import importlib
import json
from pathlib import Path
import platform
import struct
import subprocess
import sys
import time
import types
import wave

import numpy as np
import onnx
import onnxruntime as ort
import soxr


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("checkout", type=Path)
parser.add_argument("model", type=Path)
parser.add_argument("output", type=Path)
parser.add_argument("--rounds", type=int, default=7)
parser.add_argument("--iterations", type=int, default=10)
parser.add_argument("--corpus", type=Path)
parser.add_argument("--skip-timing", action="store_true")
args = parser.parse_args()
if args.rounds < 1 or args.iterations < 1:
    parser.error("rounds and iterations must be positive")
checkout = args.checkout.resolve()
args.output.mkdir(parents=True, exist_ok=True)
output = args.output.resolve()
source_dir = checkout / "src/pipecat/audio/turn/smart_turn"
caller_path = source_dir / "local_smart_turn_v3.py"

# Skip only the root package's distribution-version banner; analyzer modules are unmodified.
package = types.ModuleType("pipecat")
package.__path__ = [str(checkout / "src/pipecat")]
sys.modules["pipecat"] = package
caller = importlib.import_module("pipecat.audio.turn.smart_turn.local_smart_turn_v3")
from loguru import logger

logger.remove()
# ORT 1.16's Python wheel requires an explicit provider; retain the source's CPU session settings.
analyzer = caller.LocalSmartTurnAnalyzerV3.__new__(caller.LocalSmartTurnAnalyzerV3)
caller.BaseSmartTurn.__init__(analyzer)
analyzer._log_data = False
session_options = ort.SessionOptions()
session_options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
session_options.inter_op_num_threads = 1
session_options.intra_op_num_threads = 1
session_options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
analyzer._session = ort.InferenceSession(str(args.model.resolve()), sess_options=session_options,
                                       providers=["CPUExecutionProvider"])
analyzer.set_sample_rate(16000)

# Reuse the caller's nested padding function for the feature-only timing boundary.
source = ast.parse(caller_path.read_text())
analyzer_node = next(node for node in source.body if isinstance(node, ast.ClassDef)
                     and node.name == "LocalSmartTurnAnalyzerV3")
predict_node = next(node for node in analyzer_node.body if isinstance(node, ast.FunctionDef)
                    and node.name == "_predict_endpoint")
padding_node = next(node for node in predict_node.body if isinstance(node, ast.FunctionDef)
                    and node.name == "truncate_audio_to_last_n_seconds")
padding_scope = {"np": np, "_MODEL_SAMPLE_RATE": 16000}
exec(compile(ast.Module(body=[padding_node], type_ignores=[]), str(caller_path), "exec"), padding_scope)
prepare_audio = padding_scope[padding_node.name]

cases = {}
expected_audio_hashes = {}
audio_sources = []
if args.corpus is not None:
    corpus = json.loads(args.corpus.read_text())
    for fixture in corpus["cases"]:
        if not fixture["name"] or fixture["name"] in cases:
            raise ValueError("corpus names must be nonempty and unique")
        payload = Path(fixture["audio_file"]).read_bytes()
        if hashlib.sha256(payload).hexdigest() != fixture["audio_sha256"]:
            raise ValueError(f"audio checksum mismatch: {fixture['name']}")
        audio = np.frombuffer(payload, dtype="<f4").copy()
        if not len(audio) or len(audio) != fixture["audio_samples"] or not np.isfinite(audio).all():
            raise ValueError(f"invalid corpus audio: {fixture['name']}")
        cases[fixture["name"]] = audio
        expected_audio_hashes[fixture["name"]] = fixture["audio_sha256"]
    if not cases:
        raise ValueError("corpus contains no cases")
else:
    fixture_dir = Path(__file__).resolve().parent.parent / "features"
    for name in ("silence", "short", "pause", "long", "quiet_dc", "constant"):
        payload = gzip.decompress((fixture_dir / f"{name}.f32.gz").read_bytes())
        sample_count, _ = struct.unpack_from("<II", payload)
        cases[name] = np.frombuffer(payload, dtype="<f4", count=sample_count, offset=8).copy()
    for name, relative in (
        ("speech_question", "scripts/release-evals/assets/capital_question.wav"),
        ("speech_watch", "scripts/provider-watch/assets/speech-16k.wav"),
    ):
        path = checkout / relative
        with wave.open(str(path), "rb") as reader:
            if reader.getsampwidth() != 2 or reader.getcomptype() != "NONE":
                raise RuntimeError(f"expected PCM16 WAV: {path}")
            audio = np.frombuffer(reader.readframes(reader.getnframes()), dtype="<i2").astype(np.float32) / 32768
            channels, sample_rate = reader.getnchannels(), reader.getframerate()
            if channels != 1:
                audio = audio.reshape(-1, channels).mean(axis=1)
            if sample_rate != 16000:
                audio = soxr.resample(audio, sample_rate, 16000, quality="HQ")
        cases[name] = audio.astype(np.float32)
        audio_sources.append({"name": name, "path": relative, "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                              "sample_rate": sample_rate, "channels": channels})
    cases["speech_cut"] = cases["speech_question"][:len(cases["speech_question"]) // 2]

model = onnx.load(str(args.model))
bundled_path = source_dir / "data/smart-turn-v3.2-cpu.onnx"
bundled = onnx.load(str(bundled_path))
model_ir, bundled_ir = model.ir_version, bundled.ir_version
model_differences = [field.name for field in model.DESCRIPTOR.fields
                     if getattr(model, field.name) != getattr(bundled, field.name)]
model.ir_version = bundled.ir_version = 0
source_files = [caller_path, source_dir / "_whisper_features.py", source_dir / "base_smart_turn.py"]
report = {
    "reference_commit": subprocess.check_output(["git", "-C", str(checkout), "rev-parse", "HEAD"], text=True).strip(),
    "source_sha256": {str(path.relative_to(checkout)): hashlib.sha256(path.read_bytes()).hexdigest() for path in source_files},
    "source_dirty": bool(subprocess.check_output(["git", "-C", str(checkout), "status", "--porcelain", "--", *map(str, source_files)], text=True).strip()),
    "model_sha256": hashlib.sha256(args.model.read_bytes()).hexdigest(),
    "model_ir": model_ir,
    "bundled_model_sha256": hashlib.sha256(bundled_path.read_bytes()).hexdigest(),
    "bundled_model_ir": bundled_ir,
    "models_equal_except_ir": model == bundled,
    "model_differing_fields": model_differences,
    "model_graph_equal": model.graph == bundled.graph,
    "python": platform.python_version(), "numpy": np.__version__, "onnxruntime": ort.__version__,
    "providers": analyzer._session.get_providers(),
    "platform": platform.platform(), "machine": platform.machine(),
    "rounds": args.rounds, "iterations": args.iterations, "warmup_iterations": 3,
    "intra_threads": 1, "inter_threads": 1,
    "audio_sources": audio_sources,
    "corpus_sha256": hashlib.sha256(args.corpus.read_bytes()).hexdigest() if args.corpus is not None else None,
    "timing_enabled": not args.skip_timing,
    "scope": "16 kHz mono input; actual local analyzer methods; explicit CPU session with source settings; excludes async scheduling and transcript wait",
    "cases": [],
}
bench_lines = []
for name, audio in cases.items():
    audio_file, features_file = output / f"{name}.audio.f32", output / f"{name}.features.f32"
    features = caller.compute_whisper_log_mel_features(prepare_audio(audio))
    prediction = analyzer._predict_endpoint(audio)
    feature_probability = analyzer._session.run(None, {"input_features": features[np.newaxis]})[0][0].item()
    if abs(feature_probability - prediction["probability"]) > 1e-7:
        raise AssertionError("isolated feature path differs from actual analyzer input")
    audio.astype("<f4").tofile(audio_file)
    features.astype("<f4").tofile(features_file)
    audio_sha256 = hashlib.sha256(audio_file.read_bytes()).hexdigest()
    features_sha256 = hashlib.sha256(features_file.read_bytes()).hexdigest()
    if args.corpus is not None and audio_sha256 != expected_audio_hashes[name]:
        raise ValueError(f"copied audio differs from corpus: {name}")
    measurements = {}
    for stage in (() if args.skip_timing else ("features", "inference", "predict")):
        if stage == "features":
            operation = lambda: caller.compute_whisper_log_mel_features(prepare_audio(audio))
        elif stage == "inference":
            input_features = features[np.newaxis]
            operation = lambda: analyzer._session.run(None, {"input_features": input_features})
        else:
            operation = lambda: analyzer._predict_endpoint(audio)
        for _ in range(3):
            operation()
        measurements[stage] = []
        for _ in range(args.rounds):
            started = time.perf_counter_ns()
            for _ in range(args.iterations):
                operation()
            ns_per_operation = (time.perf_counter_ns() - started) / args.iterations
            measurements[stage].append(ns_per_operation)
            bench_lines.append(f"BenchmarkPipecatNative/{name}/{stage} {args.iterations} {ns_per_operation:.0f} ns/op")
    report["cases"].append({"name": name, "audio_file": str(audio_file), "features_file": str(features_file),
                             "audio_sha256": audio_sha256, "features_sha256": features_sha256,
                             "audio_samples": len(audio), "probability": prediction["probability"],
                             "prediction": prediction["prediction"], "timings_ns": measurements})
    print(f"{name}: samples={len(audio)} probability={prediction['probability']:.9f}", flush=True)
(output / "reference.json").write_text(json.dumps(report, indent=2) + "\n")
(output / "python.bench").write_text("\n".join(bench_lines) + "\n")
print(f"Results: {output}; ORT={ort.__version__}; equal models except IR={report['models_equal_except_ir']}")
