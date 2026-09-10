"""Download a fixed-seed labeled sample and create controlled audio conditions."""

import argparse
import audioop
import hashlib
import io
import json
from pathlib import Path
import platform
import random
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

import numpy as np
import soundfile as sf
import soxr


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("output", type=Path)
parser.add_argument("--count", type=int, default=256)
parser.add_argument("--seed", type=int, default=20260909)
args = parser.parse_args()
dataset = "pipecat-ai/smart-turn-data-v3.1-test"
revision = "2a9377baf2bbc73ba176c4505fe4adf988288fe9"
if args.count < 1:
    parser.error("count must be positive")
if sys.byteorder != "little":
    parser.error("audioop PCM conversion requires a little-endian host")
args.output.mkdir(parents=True, exist_ok=True)
output = args.output.resolve()
with urllib.request.urlopen(f"https://huggingface.co/api/datasets/{dataset}", timeout=60) as response:
    metadata = json.load(response)
if metadata["sha"] != revision:
    raise RuntimeError("dataset revision changed; review and explicitly update the pinned revision")
row_count = metadata["cardData"]["dataset_info"]["splits"][0]["num_examples"]
if args.count > row_count:
    parser.error("count exceeds dataset size")
selected_rows = sorted(random.Random(args.seed).sample(range(row_count), args.count))
manifest = {
    "dataset": dataset, "revision": revision, "config": "default", "split": "train", "seed": args.seed,
    "selection": "uniform random row indices without replacement, selected before inference",
    "selected_rows": selected_rows, "python": platform.python_version(), "rng": "random.Random.sample",
    "shards": sorted(entry["rfilename"] for entry in metadata["siblings"] if entry["rfilename"].endswith(".parquet")),
    "population_rows": row_count, "source_count": args.count,
    "label": "endpoint_bool: true means complete; false means incomplete",
    "numpy": np.__version__, "soundfile": sf.__version__, "soxr": soxr.__version__,
    "conditions": {
        "clean": "decoded mono PCM, resampled to 16 kHz with soxr HQ",
        "silence_250ms": "append 4000 zero samples at 16 kHz; possible leading context loss beyond 8 seconds; original label retained",
        "noise_20db": "seeded Gaussian white noise at 20 dB clip-wide RMS SNR; no clipping; zero-power input receives zero noise",
        "mulaw_8khz": "soxr HQ 8 kHz, PCM16, G.711 mu-law encode/decode, soxr HQ 16 kHz",
    },
    "cases": [],
}
(output / "selection.json").write_text(json.dumps({key: value for key, value in manifest.items() if key != "cases"}, indent=2) + "\n")
for position, row_index in enumerate(selected_rows):
    cached_path = output / f"row_{row_index}.json"
    audio_path = output / f"row_{row_index}.flac"
    if cached_path.exists() and audio_path.exists():
        cached = json.loads(cached_path.read_text())
        if cached["revision"] != revision or cached["row_index"] != row_index:
            raise RuntimeError(f"cached row identity mismatch: {row_index}")
        row = cached["row"]
        audio_bytes = audio_path.read_bytes()
        if hashlib.sha256(audio_bytes).hexdigest() != cached["audio_sha256"]:
            raise RuntimeError(f"cached audio checksum mismatch: {row_index}")
    else:
        query = urllib.parse.urlencode({"dataset": dataset, "config": "default", "split": "train",
                                       "offset": row_index, "length": 1})
        time.sleep(1)
        for attempt in range(6):
            try:
                with urllib.request.urlopen(f"https://datasets-server.huggingface.co/rows?{query}", timeout=60) as response:
                    page = json.load(response)
                if page["num_rows_total"] != row_count or len(page["rows"]) != 1:
                    raise RuntimeError("dataset row count changed or requested row missing")
                entry = page["rows"][0]
                if entry["row_idx"] != row_index or entry["truncated_cells"]:
                    raise RuntimeError("wrong row returned or truncated row")
                row = entry["row"]
                audio_url = row["audio"][0]["src"]
                if f"/--/{revision}/--/" not in audio_url:
                    raise RuntimeError("audio URL does not match pinned dataset revision")
                with urllib.request.urlopen(audio_url, timeout=60) as response:
                    audio_bytes = response.read()
                break
            except urllib.error.HTTPError as error:
                if error.code not in (429, 500, 502, 503, 504) or attempt == 5:
                    raise
                delay = max(60, int(error.headers.get("Retry-After", "60")))
                print(f"HTTP {error.code} at row {row_index}; retrying in {delay}s", flush=True)
                time.sleep(delay)
            except (OSError, TimeoutError):
                if attempt == 5:
                    raise
                time.sleep(2 ** attempt)
        row.pop("audio")
        audio_path.write_bytes(audio_bytes)
        cached_path.write_text(json.dumps({"revision": revision, "row_index": row_index, "row": row,
                                          "audio_sha256": hashlib.sha256(audio_bytes).hexdigest()}, indent=2) + "\n")
    if type(row["endpoint_bool"]) is not bool:
        raise RuntimeError(f"missing boolean ground truth for row {row_index}")
    audio, sample_rate = sf.read(io.BytesIO(audio_bytes), dtype="float32", always_2d=True)
    channels = audio.shape[1]
    audio = audio.mean(axis=1)
    if sample_rate != 16000:
        audio = soxr.resample(audio, sample_rate, 16000, quality="HQ")
    audio = np.asarray(audio, dtype=np.float32)
    if not len(audio) or not np.isfinite(audio).all():
        raise RuntimeError(f"empty or non-finite audio at row {row_index}")
    noise = np.random.default_rng(np.random.SeedSequence([args.seed, row_index])).standard_normal(len(audio))
    audio_rms = np.sqrt(np.mean(audio.astype(np.float64) ** 2))
    noise *= audio_rms / (10 * np.sqrt(np.mean(noise ** 2)))
    telephone = soxr.resample(audio, 16000, 8000, quality="HQ")
    telephone = np.rint(np.clip(telephone * 32768, -32768, 32767)).astype("<i2")
    telephone = audioop.ulaw2lin(audioop.lin2ulaw(telephone.tobytes(), 2), 2)
    telephone = np.frombuffer(telephone, dtype="<i2").astype(np.float32) / 32768
    telephone = soxr.resample(telephone, 8000, 16000, quality="HQ")
    for condition, samples in (
        ("clean", audio),
        ("silence_250ms", np.concatenate((audio, np.zeros(4000, dtype=np.float32)))),
        ("noise_20db", (audio + noise).astype(np.float32)),
        ("mulaw_8khz", telephone.astype(np.float32)),
    ):
        name = f"row_{row_index}_{condition}"
        audio_file = output / f"{name}.audio.f32"
        samples = samples.astype("<f4")
        samples.tofile(audio_file)
        manifest["cases"].append({
            "name": name, "source_id": row["id"], "row_index": row_index,
            "condition": condition, "expected_complete": row["endpoint_bool"],
            "language": row["language"], "synthetic": row["synthetic"],
            "midfiller": row["midfiller"], "endfiller": row["endfiller"], "source_dataset": row["dataset"],
            "source_sha256": hashlib.sha256(audio_bytes).hexdigest(), "source_sample_rate": sample_rate,
            "source_channels": channels, "audio_file": str(audio_file), "audio_samples": len(samples),
            "source_samples_16khz": len(audio), "silence_discards_context": condition == "silence_250ms" and len(audio) > 124000,
            "audio_sha256": hashlib.sha256(samples.tobytes()).hexdigest(),
        })
    if (position + 1) % 16 == 0 or position + 1 == len(selected_rows):
        print(f"Downloaded {position + 1}/{len(selected_rows)} source recordings", flush=True)
with urllib.request.urlopen(f"https://huggingface.co/api/datasets/{dataset}", timeout=60) as response:
    if json.load(response)["sha"] != revision:
        raise RuntimeError("dataset changed during download")
(output / "corpus.json").write_text(json.dumps(manifest, indent=2) + "\n")
print(f"Corpus: {output / 'corpus.json'}; cases={len(manifest['cases'])}")
