"""Build causal text probes from the pinned LiveKit English validation rows."""

import argparse
import hashlib
import json
import math
from pathlib import Path


DATASET_REVISION = "ca9d98a9686b920a2d8c9eb984224ba9be74e4dd"
BENCHMARK_REVISION = "6594d8b3b8af385b15f116dde310ce45af92d646"


def export(source, destination):
    raw = source.read_bytes()
    payload = json.loads(raw)
    if payload["partial"] or len(payload["rows"]) != 100:
        raise ValueError("expected 100 complete viewer rows")
    cases, excluded = [], []
    for row_index, entry in enumerate(payload["rows"]):
        if entry["row_idx"] != row_index or entry["truncated_cells"]:
            raise ValueError("row selection or truncation differs")
        row = entry["row"]
        if row["language"] != "en" or not row["audio"]:
            raise ValueError("expected English rows with revision-bearing asset URLs")
        if any(f"/{DATASET_REVISION}/" not in audio["src"] for audio in row["audio"]):
            raise ValueError("viewer asset revision differs from the pinned revision")
        for message in row["messages"]:
            if not isinstance(message["role"], str) or not isinstance(message["content"], str):
                raise ValueError("invalid history message")
        for word in row["words"]:
            if not math.isfinite(word["end"]) or not isinstance(word["word"], str):
                raise ValueError("invalid timed word")
        for span_index, span in enumerate(row["silence_spans"]):
            if not math.isfinite(span["start"]) or not math.isfinite(span["end"]) or span["end"] < span["start"]:
                raise ValueError("invalid silence span")
            if span["end"] - span["start"] < 0.2 - 1e-6:
                continue
            complete = span_index == len(row["silence_spans"]) - 1
            visible_until = span["start"] + 0.2 - 0.5
            text = " ".join(word["word"].strip() for word in row["words"]
                            if word["end"] <= visible_until + 1e-6 and word["word"].strip())
            case = {"name": f"row_{row_index}_pause_{span_index}", "language": "en",
                    "history": row["messages"], "text": text, "complete": complete,
                    "turn_id": row["id"], "row_index": row_index, "span_index": span_index,
                    "score_time_s": span["start"] + 0.2, "visible_until_s": visible_until}
            if text:
                cases.append(case)
            else:
                excluded.append({key: value for key, value in case.items() if key not in {"history", "text"}})
    corpus = {
        "schema": 1, "kind": "independent_dataset", "source": "livekit/eot-bench-data",
        "revision": DATASET_REVISION, "split": "en/validation",
        "license": "Pinned dataset card: CC-BY-4.0; benchmark README instead says Apache-2.0. Preserve this conflict.",
        "sampling": "Positional rows 0..99; one score 200ms into each original silence span lasting at least 200ms; 500ms transcript lag.",
        "label_policy": "Original final silence span is complete; earlier spans are hold. Final-span identity is assigned before duration filtering.",
        "holdout_status": "Published validation split; model training overlap and conversation independence are unverified. Multiple pauses share turns.",
        "benchmark_revision": BENCHMARK_REVISION,
        "benchmark_source": "eot_harness/io.py:11,31-46,80-102",
        "source_sha256": hashlib.sha256(raw).hexdigest(), "source_rows": 100,
        "no_audio_downloaded": True, "no_text_exclusions": excluded, "cases": cases,
    }
    destination.write_text(json.dumps(corpus, indent=2, ensure_ascii=True, allow_nan=False) + "\n")
    print(json.dumps({"cases": len(cases), "hold": sum(not case["complete"] for case in cases),
                      "eot": sum(case["complete"] for case in cases), "no_text": len(excluded)}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    export(args.source, args.destination)
