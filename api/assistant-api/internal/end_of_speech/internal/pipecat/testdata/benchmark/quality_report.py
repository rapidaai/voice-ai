"""Compare labeled completion accuracy and paired prediction parity."""

import argparse
import hashlib
import json
import math
from pathlib import Path
import random


def classification_metrics(cases, predictions):
    counts = {"true_complete": 0, "true_incomplete": 0, "false_complete": 0, "false_incomplete": 0}
    squared_error = 0.0
    for case in cases:
        prediction = predictions[case["name"]]
        expected = case["expected_complete"]
        if type(expected) is not bool:
            raise ValueError("ground truth must be boolean")
        if prediction["prediction"] == 1:
            counts["true_complete" if expected else "false_complete"] += 1
        else:
            counts["false_incomplete" if expected else "true_incomplete"] += 1
        squared_error += (prediction["probability"] - int(expected)) ** 2
    result = {"count": len(cases), **counts, "brier_score": squared_error / len(cases) if cases else None}
    for name, numerator, denominator in (
        ("accuracy", counts["true_complete"] + counts["true_incomplete"], len(cases)),
        ("premature_completion_rate", counts["false_complete"], counts["false_complete"] + counts["true_incomplete"]),
        ("missed_completion_rate", counts["false_incomplete"], counts["false_incomplete"] + counts["true_complete"]),
    ):
        if not denominator:
            result[name] = {"value": None, "numerator": numerator, "denominator": 0, "wilson_95": None}
            continue
        fraction = numerator / denominator
        correction = 1 + 1.96 ** 2 / denominator
        center = (fraction + 1.96 ** 2 / (2 * denominator)) / correction
        radius = 1.96 * math.sqrt(fraction * (1 - fraction) / denominator + 1.96 ** 2 / (4 * denominator ** 2)) / correction
        result[name] = {"value": fraction, "numerator": numerator, "denominator": denominator,
                        "wilson_95": [max(0.0, center - radius), min(1.0, center + radius)]}
    premature = result["premature_completion_rate"]["value"]
    missed = result["missed_completion_rate"]["value"]
    result["balanced_accuracy"] = 1 - (premature + missed) / 2 if premature is not None and missed is not None else None
    return result


def evaluate_quality(corpus_path, reference_path, baseline_path, fft_path):
    corpus = json.loads(corpus_path.read_text())
    reference = json.loads(reference_path.read_text())
    baseline = json.loads(baseline_path.read_text())
    fft = json.loads(fft_path.read_text())
    if reference["corpus_sha256"] != hashlib.sha256(corpus_path.read_bytes()).hexdigest():
        raise ValueError("reference corpus checksum mismatch")
    reference_digest = hashlib.sha256(reference_path.read_bytes()).hexdigest()
    for report in (baseline, fft):
        if report["reference_sha256"] != reference_digest:
            raise ValueError("Go report reference checksum mismatch")
    indexed = {}
    for name, report in (("corpus", corpus), ("pipecat", reference), ("rapida", baseline), ("fft", fft)):
        indexed[name] = {case["name"]: case for case in report["cases"]}
        if not indexed[name] or len(indexed[name]) != len(report["cases"]):
            raise ValueError(f"empty or duplicate case names: {name}")
        if set(indexed[name]) != set(indexed["corpus"]):
            raise ValueError(f"case set mismatch: {name}")
        if name != "corpus":
            for prediction in report["cases"]:
                probability = prediction["probability"]
                if not math.isfinite(probability) or not 0 <= probability <= 1:
                    raise ValueError(f"invalid probability: {name}")
                if prediction["prediction"] != int(probability > 0.5):
                    raise ValueError(f"decision does not match threshold: {name}")
                if name != "pipecat" and (not math.isfinite(prediction["max_feature_error"]) or prediction["max_feature_error"] < 0):
                    raise ValueError(f"invalid feature error: {name}")
    conditions = sorted(corpus["conditions"])
    selected_rows = corpus["selected_rows"]
    if len(set(selected_rows)) != len(selected_rows) or len(selected_rows) != corpus["source_count"]:
        raise ValueError("selected row indices must be unique and match source count")
    if {case["row_index"] for case in corpus["cases"]} != set(selected_rows):
        raise ValueError("actual rows differ from declared selection")
    sources = {}
    for case in corpus["cases"]:
        if type(case["expected_complete"]) is not bool:
            raise ValueError("ground truth must be boolean")
        if case["condition"] not in conditions:
            raise ValueError("unknown audio condition")
        source = sources.setdefault(case["source_id"], [])
        source.append(case)
    if len(sources) != corpus["source_count"]:
        raise ValueError("source count mismatch or duplicate source identifiers")
    for cases in sources.values():
        if sorted(case["condition"] for case in cases) != conditions:
            raise ValueError("source does not have exactly one case per condition")
        if len({case["expected_complete"] for case in cases}) != 1:
            raise ValueError("source labels differ across conditions")
        if len({case["row_index"] for case in cases}) != 1 or len({case["source_sha256"] for case in cases}) != 1:
            raise ValueError("source identity differs across conditions")
    for case in corpus["cases"]:
        if indexed["pipecat"][case["name"]]["audio_sha256"] != case["audio_sha256"]:
            raise ValueError("reference audio digest differs from corpus")
    results = {
        "dataset": corpus["dataset"], "revision": corpus["revision"], "seed": corpus["seed"],
        "source_count": len(sources), "case_count": len(corpus["cases"]),
        "unique_source_audio": len({case["source_sha256"] for case in corpus["cases"]}),
        "conditions": {}, "clean_groups": {}, "pairwise": {}, "errors": [],
        "artifact_sha256": {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in
                            (("corpus", corpus_path), ("reference", reference_path), ("rapida", baseline_path), ("fft", fft_path))},
        "inference_provenance": {key: reference[key] for key in
                                 ("reference_commit", "source_sha256", "source_dirty", "model_sha256", "onnxruntime", "numpy", "providers")},
        "limitations": [
            "Publisher's test dataset, physically named train split; independent holdout status for v3.2 is unverified.",
            "Ground truth is taken from the dataset, not independently relabeled.",
            "Audio variants retain original labels; noise and bandwidth loss can remove relevant cues.",
            "Each condition has one observation per source; pooled conditions are not independent recordings.",
            "Wilson intervals describe this sample and assume independent recordings, not deployment representativeness.",
            "Speaker/conversation cluster IDs are unavailable; dependence between source recordings is not controlled.",
            "No runtime VAD gating, transcript waiting, live interruption, or completion-delay evaluation.",
        ],
    }
    for condition in conditions:
        cases = [case for case in corpus["cases"] if case["condition"] == condition]
        results["conditions"][condition] = {name: classification_metrics(cases, indexed[name]) for name in ("pipecat", "rapida", "fft")}
    results["silence_context_groups"] = {}
    for discards_context in (False, True):
        cases = [case for case in corpus["cases"] if case["condition"] == "silence_250ms" and case["silence_discards_context"] == discards_context]
        results["silence_context_groups"][str(discards_context)] = {name: classification_metrics(cases, indexed[name]) for name in ("pipecat", "rapida", "fft")}
    for group in ("language", "synthetic", "source_dataset", "midfiller", "endfiller"):
        results["clean_groups"][group] = {}
        for value in sorted({str(case[group]) for case in corpus["cases"]}):
            cases = [case for case in corpus["cases"] if case["condition"] == "clean" and str(case[group]) == value]
            results["clean_groups"][group][value] = {name: classification_metrics(cases, indexed[name]) for name in ("pipecat", "rapida", "fft")}
    for left, right in (("pipecat", "rapida"), ("pipecat", "fft"), ("rapida", "fft")):
        differences = [abs(indexed[left][name]["probability"] - indexed[right][name]["probability"]) for name in indexed[left]]
        disagreements = [name for name in indexed[left] if indexed[left][name]["prediction"] != indexed[right][name]["prediction"]]
        regressions = [name for name in disagreements if indexed[left][name]["prediction"] == int(indexed["corpus"][name]["expected_complete"])]
        results["pairwise"][f"{left}_vs_{right}"] = {
            "count": len(differences), "decision_disagreements": disagreements,
            "correct_to_wrong": regressions, "wrong_to_correct": [name for name in disagreements if name not in regressions],
            "max_probability_error": max(differences), "mean_probability_error": sum(differences) / len(differences),
            "worst_probability_case": max(indexed[left], key=lambda name: abs(indexed[left][name]["probability"] - indexed[right][name]["probability"])),
            "signed_probability_differences": {name: indexed[right][name]["probability"] - indexed[left][name]["probability"] for name in indexed[left]},
        }
        paired = results["pairwise"][f"{left}_vs_{right}"]
        source_ids = list(sources)
        source_deltas = {condition: [] for condition in conditions}
        discordant_sources = 0
        for cases in sources.values():
            discordant_sources += any(case["name"] in disagreements for case in cases)
            for case in cases:
                name = case["name"]
                expected = int(case["expected_complete"])
                source_deltas[case["condition"]].append(int(indexed[right][name]["prediction"] == expected) - int(indexed[left][name]["prediction"] == expected))
        paired["discordant_source_count"] = discordant_sources
        paired["zero_discordance_one_sided_95_upper"] = 1 - 0.05 ** (1 / len(source_ids)) if discordant_sources == 0 else None
        random_generator = random.Random(corpus["seed"])
        bootstrap_deltas = {condition: [] for condition in conditions}
        for _ in range(2000):
            selected = random_generator.choices(range(len(source_ids)), k=len(source_ids))
            for condition in conditions:
                bootstrap_deltas[condition].append(sum(source_deltas[condition][index] for index in selected) / len(selected))
        paired["accuracy_difference_by_condition"] = {}
        for condition in conditions:
            samples = sorted(bootstrap_deltas[condition])
            paired["accuracy_difference_by_condition"][condition] = {
                "right_minus_left": sum(source_deltas[condition]) / len(source_ids),
                "source_bootstrap_95": [samples[49], samples[1949]],
            }
        paired["bootstrap_note"] = "2000 fixed-seed source resamples keep all conditions paired; zero observed flips collapse the bootstrap interval and do not prove equivalence."
    results["max_feature_error"] = {name: max(case["max_feature_error"] for case in indexed[name].values()) for name in ("rapida", "fft")}
    results["feature_tolerance_failures"] = {name: [case["name"] for case in indexed[name].values() if case["max_feature_error"] > 2e-5] for name in ("rapida", "fft")}
    results["probability_tolerance_failures"] = {name: [case["name"] for case in indexed[name].values()
                                                      if abs(case["probability"] - indexed["pipecat"][case["name"]]["probability"]) > 1e-4]
                                                 for name in ("rapida", "fft")}
    results["pipecat_parity_passed"] = {name: not (results["feature_tolerance_failures"][name] or results["probability_tolerance_failures"][name]
                                                  or results["pairwise"][f"pipecat_vs_{name}"]["decision_disagreements"])
                                          for name in ("rapida", "fft")}
    results["closest_to_threshold"] = sorted(
        ({"name": case["name"], "probability": case["probability"], "margin": abs(case["probability"] - 0.5)} for case in reference["cases"]),
        key=lambda case: case["margin"],
    )[:10]
    for case in corpus["cases"]:
        wrong = [name for name in ("pipecat", "rapida", "fft") if indexed[name][case["name"]]["prediction"] != int(case["expected_complete"])]
        if wrong:
            results["errors"].append({"name": case["name"], "source_id": case["source_id"], "condition": case["condition"],
                                      "expected_complete": case["expected_complete"], "language": case["language"], "wrong": wrong,
                                      "probabilities": {name: indexed[name][case["name"]]["probability"] for name in ("pipecat", "rapida", "fft")}})
    return results


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("corpus", "reference", "baseline", "fft", "output"):
        parser.add_argument(name, type=Path)
    args = parser.parse_args()
    report = evaluate_quality(args.corpus, args.reference, args.baseline, args.fft)
    args.output.write_text(json.dumps(report, indent=2, allow_nan=False) + "\n")
    for condition, implementations in report["conditions"].items():
        for implementation, metrics in implementations.items():
            print(f"{condition:16} {implementation:8} n={metrics['count']} accuracy={metrics['accuracy']['value']:.4f} "
                  f"false_complete={metrics['false_complete']} false_incomplete={metrics['false_incomplete']}")
    for comparison, metrics in report["pairwise"].items():
        print(f"{comparison}: disagreements={len(metrics['decision_disagreements'])}/{metrics['count']} "
              f"max_probability_error={metrics['max_probability_error']:.9g}")
