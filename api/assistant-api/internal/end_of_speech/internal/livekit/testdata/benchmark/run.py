"""Run full-corpus quality, then three-case latency in serial alternating rounds."""

import argparse
import copy
import fcntl
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

from reference import digest
from report import compare, latency


def run(args):
    if not args.approve_experiments:
        raise ValueError("tests must finish first; pass --approve-experiments only after approval")
    if args.rounds < 2 or args.iterations < 1:
        raise ValueError("requires at least two rounds and a positive iteration count")
    args.output.mkdir(parents=True, exist_ok=False)
    scripts = Path(__file__).parent.resolve()
    package = scripts.parent.parent
    paths = sorted(path for path in package.iterdir() if path.suffix in {".go", ".h", ".c"})
    paths += sorted(path for path in scripts.iterdir() if path.suffix in {".py", ".json"})
    before = {str(path): digest(path) for path in paths}
    env = dict(os.environ, OMP_NUM_THREADS="1", OPENBLAS_NUM_THREADS="1", VECLIB_MAXIMUM_THREADS="1",
               TOKENIZERS_PARALLELISM="false", GOMAXPROCS="1", HF_HUB_OFFLINE="1", TRANSFORMERS_OFFLINE="1")
    base = [sys.executable, "-B", str(scripts / "reference.py"),
            "--checkout", str(args.checkout), "--model", str(args.model), "--tokenizer", str(args.tokenizer),
            "--ort-library", str(args.ort_library), "--model-type", args.model_type,
            "--threshold", str(args.threshold), "--corpus", str(args.corpus), "--metadata", str(args.metadata)]

    def command(argv, log, environment=env, allow_failure=False):
        with log.open("w") as stream:
            completed = subprocess.run(argv, env=environment, stdout=stream, stderr=subprocess.STDOUT)
        if completed.returncode and not allow_failure:
            raise RuntimeError(f"command failed ({completed.returncode}); see {log}")
        return completed.returncode

    quality_dir = args.output / "quality"
    command(base + ["--output", str(quality_dir)], args.output / "python-quality.log")
    reference_path = quality_dir / "reference.json"
    native_path = quality_dir / "go.json"
    go_env = dict(env, LIVEKIT_BENCHMARK_REFERENCE=str(reference_path), LIVEKIT_BENCHMARK_OUTPUT=str(native_path))
    status = command([str(args.go_binary), "-test.run=^TestLiveKitNativeParity$", "-test.count=1", "-test.v"],
                     args.output / "go-quality.log", go_env, allow_failure=True)
    raw = reference_path.read_bytes()
    reference, native = json.loads(raw), json.loads(native_path.read_text())
    if native["reference_sha256"] != hashlib.sha256(raw).hexdigest():
        raise ValueError("quality reference digest differs")
    summary = compare(reference, native)
    summary["go_quality_exit_code"] = status
    (args.output / "quality.json").write_text(json.dumps(summary, indent=2, ensure_ascii=True) + "\n")
    names = ("complete_en", "repetition_incomplete", "long_left_truncation")
    if not set(names) <= {case["name"] for case in reference["cases"]}:
        raise ValueError("latency corpus requires the three documented representative case names")
    selection = [value for name in names for value in ("--case", name)]
    bench = f"^BenchmarkLiveKitNative$/{args.model_type}$/^({'|'.join(names)})$/"
    rounds = []
    python_reference = None
    go_text = []
    for index in range(args.rounds):
        directory = args.output / f"round-{index + 1:02d}"
        directory.mkdir()
        order = ("python", "go") if index % 2 == 0 else ("go", "python")
        for implementation in order:
            if implementation == "python":
                command(base + ["--output", str(directory), "--rounds", "1", "--iterations", str(args.iterations)] + selection,
                        directory / "python.log")
            else:
                command([str(args.go_binary), "-test.run=^$", f"-test.bench={bench}", "-test.benchmem",
                         f"-test.benchtime={args.iterations}x", "-test.count=1", "-test.cpu=1"], directory / "go.bench", go_env)
        current = json.loads((directory / "reference.json").read_text())
        for field in ("model_sha256", "tokenizer_sha256", "ort_library_sha256", "python_ort_binary_sha256", "source_sha256", "harness_sha256", "metadata_sha256", "corpus_sha256", "threshold"):
            if current[field] != reference[field]:
                raise ValueError(f"reference drift during experiment: {field}")
        if python_reference is None:
            python_reference = copy.deepcopy(current)
            python_reference["timing"]["samples"] = []
            python_reference["timing"]["rounds"] = args.rounds
        python_reference["timing"]["samples"].extend({**sample, "round": index} for sample in current["timing"]["samples"])
        go_text.append((directory / "go.bench").read_text())
        rounds.append({"round": index + 1, "order": order})
    after = {str(path): digest(path) for path in paths}
    if before != after:
        raise ValueError("Go sources changed during measurements; discard this run and rebuild")
    result = latency(python_reference, "\n".join(go_text))
    result.update(round_order=rounds, go_binary_sha256=digest(args.go_binary), source_sha256=before,
                  parity_pass=summary["parity_pass"], quality_report="quality.json",
                  build_note="Record the build command and freeze source before compiling; runtime source hashes alone do not certify binary contents")
    (args.output / "latency.json").write_text(json.dumps(result, indent=2, ensure_ascii=True) + "\n")
    print(json.dumps({"output": str(args.output), "parity_pass": summary["parity_pass"], "rounds": args.rounds}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("checkout", "model", "tokenizer", "ort-library", "go-binary", "output"):
        parser.add_argument(f"--{name}", required=True, type=lambda value: Path(value).resolve())
    parser.add_argument("--model-type", required=True, choices=("en", "multilingual"))
    parser.add_argument("--threshold", required=True, type=float)
    parser.add_argument("--corpus", type=Path, default=Path(__file__).with_name("corpus.json"))
    parser.add_argument("--metadata", type=Path, default=Path(__file__).with_name("metadata.json"))
    parser.add_argument("--rounds", type=int, default=7)
    parser.add_argument("--iterations", type=int, default=10)
    parser.add_argument("--approve-experiments", action="store_true")
    args = parser.parse_args()
    with Path("/private/tmp/rapida-livekit-benchmark.lock" if sys.platform == "darwin" else "/tmp/rapida-livekit-benchmark.lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        run(args)
