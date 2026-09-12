"""Summarize repeated Python and Go measurements without hiding individual rounds."""

import argparse
import json
from pathlib import Path
import re
import statistics


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("reference", type=Path)
parser.add_argument("go_bench", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
reference = json.loads(args.reference.read_text())
go_results = {}
pattern = re.compile(r"^BenchmarkPipecatNative/([^/]+)/([^\s-]+)(?:-\d+)?\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op$")
for line in args.go_bench.read_text().splitlines():
    match = pattern.fullmatch(line)
    if not match:
        continue
    name, stage, elapsed, allocated, allocations = match.groups()
    result = go_results.setdefault((name, stage), {"rounds_ns": [], "bytes_per_op": [], "allocs_per_op": []})
    result["rounds_ns"].append(float(elapsed))
    result["bytes_per_op"].append(int(allocated))
    result["allocs_per_op"].append(int(allocations))

summary = {"reference_commit": reference["reference_commit"], "model_sha256": reference["model_sha256"],
           "onnxruntime": reference["onnxruntime"], "cases": []}
print("| Case | Stage | Pipecat median ms | Rapida median ms | Rapida / Pipecat | Go B/op |")
print("|---|---|---:|---:|---:|---:|")
for case in reference["cases"]:
    stages = {}
    for stage, python_rounds in case["timings_ns"].items():
        go = go_results[(case["name"], stage)]
        if len(go["rounds_ns"]) != reference["rounds"]:
            raise RuntimeError(f"missing Go rounds for {case['name']}/{stage}")
        python_median = statistics.median(python_rounds)
        go_median = statistics.median(go["rounds_ns"])
        stages[stage] = {"python_rounds_ns": python_rounds, **go,
                         "python_median_ms": python_median / 1e6, "go_median_ms": go_median / 1e6,
                         "ratio": go_median / python_median}
        print(f"| {case['name']} | {stage} | {python_median / 1e6:.3f} | {go_median / 1e6:.3f} | {go_median / python_median:.2f}x | {statistics.median(go['bytes_per_op']):.0f} |")
    summary["cases"].append({"name": case["name"], "stages": stages})
args.output.write_text(json.dumps(summary, indent=2) + "\n")
