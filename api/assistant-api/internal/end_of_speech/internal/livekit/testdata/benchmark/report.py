"""Report parity, labeled diagnostics, and serial latency rounds separately."""

import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import statistics


def quality(rows):
    labeled = [row for row in rows if row[0] is not None]
    tp = sum(label and decision for label, decision in labeled)
    tn = sum(not label and not decision for label, decision in labeled)
    fp = sum(not label and decision for label, decision in labeled)
    fn = sum(label and not decision for label, decision in labeled)
    return {"labeled": len(labeled), "excluded": len(rows) - len(labeled),
            "true_complete": tp, "true_incomplete": tn, "false_complete": fp, "missed_complete": fn,
            "label_agreement": (tp + tn) / len(labeled) if labeled else None,
            "false_completion_rate": fp / (fp + tn) if fp + tn else None,
            "missed_completion_rate": fn / (fn + tp) if fn + tp else None}


def compare(reference, native):
    if native.get("schema") != 1 or native["model_type"] != reference["model_type"] or native["threshold"] != reference["threshold"]:
        raise ValueError("native report schema, model, or threshold differs")
    expected = {case["name"]: case for case in reference["cases"]}
    actual = {case["name"]: case for case in native["cases"]}
    if len(expected) != len(reference["cases"]) or len(actual) != len(native["cases"]) or expected.keys() != actual.keys():
        raise ValueError("missing, duplicate, or extra predictions")
    kind = reference["corpus"]["kind"]
    if kind not in {"curated_diagnostic", "independent_dataset"}:
        raise ValueError("unknown corpus kind")
    groups = {}
    differences = []
    for name, case in expected.items():
        row = actual[name]
        for stage in ("inference", "predict", "pipeline"):
            result = row["results"][stage]
            p = result.get("probability")
            if type(p) not in (float, int) or not math.isfinite(p) or not 0 <= p <= 1:
                raise ValueError(f"invalid or missing {stage} probability for {name}")
            if type(result.get("decision")) is not bool or result["decision"] != (p >= reference["threshold"]):
                raise ValueError(f"invalid {stage} decision for {name}")
        differences.append({"name": name, "prompt_match": row["prompt_match"], "token_match": row["token_match"],
                            **{stage: {"absolute_error": abs(row["results"][stage]["probability"] - case["probability"]),
                                       "decision_match": row["results"][stage]["decision"] == case["decision"]}
                               for stage in ("inference", "predict", "pipeline")}})
        for language in ("all", case["language"]):
            group = groups.setdefault(language, {"python": [], "go_pipeline": []})
            group["python"].append((case["complete"], case["decision"]))
            group["go_pipeline"].append((case["complete"], row["results"]["pipeline"]["decision"]))
    parity_ok = all(row["prompt_match"] and row["token_match"] and
                    all(row[stage]["absolute_error"] <= 1e-5 and row[stage]["decision_match"]
                        for stage in ("inference", "predict", "pipeline")) for row in differences)
    return {"model_type": reference["model_type"], "threshold": reference["threshold"],
            "publisher_model_match": reference["publisher_model_match"],
            "publisher_tokenizer_match": reference["publisher_tokenizer_match"],
            "corpus_kind": kind, "corpus_provenance": reference["corpus"],
            "quality_metric": "diagnostic_label_agreement" if kind == "curated_diagnostic" else "dataset_label_accuracy",
            "limitation": "Parity is implementation agreement, not real-world accuracy. Curated labels are diagnostic. Independent dataset status and holdout claims require external review. Non-English cases are stress inputs for the English model.",
            "parity_pass": parity_ok, "parity": differences,
            "quality_by_language": {language: {name: quality(rows) for name, rows in group.items()} for language, group in groups.items()}}


def latency(reference, text):
    pattern = re.compile(r"^BenchmarkLiveKitNative/([^/]+)/([^/]+)/([a-z_]+)(?:-\d+)?\s+\d+\s+([\d.]+) ns/op(?:\s+.*)?$")
    go_samples, python_samples = {}, {}
    names = {case["name"] for case in reference["cases"]}
    for line in text.splitlines():
        match = pattern.match(line.strip())
        if not match:
            continue
        model, name, stage, value = match.groups()
        if model != reference["model_type"] or name not in names:
            raise ValueError("benchmark model or case differs from the reference")
        go_samples.setdefault((name, stage), []).append(float(value))
    for sample in reference["timing"]["samples"]:
        python_samples.setdefault((sample["case"], sample["stage"]), []).append(sample["ns_per_op"])
    result = []
    for name in sorted(names):
        stages = [(stage, stage) for stage in ("format", "tokenize", "inference", "predict", "pipeline")]
        if (name, "async_eot") in python_samples or (name, "production_context") in go_samples:
            stages.append(("production_context", "async_eot"))
        for stage, python_stage in stages:
            py = python_samples.get((name, python_stage), [])
            go = go_samples.get((name, stage), [])
            rounds = reference["timing"]["rounds"]
            if rounds < 2 or len(py) != rounds or len(go) != rounds:
                raise ValueError(f"{name}/{stage} requires matching repeated Python and Go rounds, at least two")
            if any(not math.isfinite(value) or value <= 0 for value in py + go):
                raise ValueError("latency samples must be finite and positive")
            result.append({"case": name, "stage": stage, "python_stage": python_stage, "python_ns_per_op": py, "go_ns_per_op": go,
                           "python_median_ns": statistics.median(py), "go_median_ns": statistics.median(go),
                           "go_over_python_median": statistics.median(go) / statistics.median(py)})
    return {"samples": result, "interpretation": "Descriptive round medians only, not significance or request-tail percentiles. Pipeline inputs may differ when prompt parity fails. Python formatting includes copying the selected context; Go includes history selection. production_context includes Go deadline creation/cancellation; async_eot includes a reused Python loop and local executor, not IPC or production scheduling. Neither is full voice latency."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("reference", type=Path)
    parser.add_argument("native", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--go-bench", type=Path)
    args = parser.parse_args()
    raw = args.reference.read_bytes()
    reference, native = json.loads(raw), json.loads(args.native.read_text())
    if native["reference_sha256"] != hashlib.sha256(raw).hexdigest():
        raise ValueError("native output came from different reference bytes")
    result = compare(reference, native)
    if args.go_bench:
        result["latency"] = latency(reference, args.go_bench.read_text())
    args.output.write_text(json.dumps(result, ensure_ascii=True, indent=2, allow_nan=False) + "\n")
    print(json.dumps({"parity_pass": result["parity_pass"], "quality_metric": result["quality_metric"]}))
