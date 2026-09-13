"""Check the bounded fixture corpus with LiveKit's pinned native tokenizer."""

import argparse
import hashlib
import json
from pathlib import Path

import tokenizers


if tokenizers.__version__ != "0.22.2":
    raise RuntimeError("verification requires tokenizers==0.22.2 from LiveKit's uv.lock")

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("asset", type=Path)
parser.add_argument("--variant", choices=("english", "multilingual"), default="english")
parser.add_argument("--output", type=Path, help="fixture directory used by generate.py")
args = parser.parse_args()
output = args.output or (Path(__file__).parent / "multilingual" if args.variant == "multilingual" else Path(__file__).parent)
reference = json.loads((output / "reference.json").read_text())
asset_path = args.asset
if hashlib.sha256(asset_path.read_bytes()).hexdigest() != reference["provenance"]["asset_sha256"]:
    raise RuntimeError("tokenizer asset digest does not match the pinned fixture source")

subset_path = output / "tokenizer.json"
if hashlib.sha256(subset_path.read_bytes()).hexdigest() != reference["provenance"]["subset_sha256"]:
    raise RuntimeError("tokenizer subset digest does not match the fixture provenance")

for label, path in [("full asset", asset_path), ("subset", subset_path)]:
    tokenizer = tokenizers.Tokenizer.from_file(str(path))
    for case in reference["cases"]:
        actual = tokenizer.encode(case["text"], add_special_tokens=False).ids
        if actual != case["ids"]:
            raise AssertionError(f'{label} {case["name"]}: expected {case["ids"]}, got {actual}')
    for case in reference.get("pretokenizer_cases", []):
        actual = [
            tokenizer.decoder.decode([piece])
            for piece, _ in tokenizer.pre_tokenizer.pre_tokenize_str(case["text"])
        ]
        if actual != case["boundaries"]:
            raise AssertionError(f'{label} {case["name"]}: expected {case["boundaries"]!r}, got {actual!r}')

print(f'Native tokenizers {tokenizers.__version__}: {len(reference["cases"])} cases passed on full asset and subset')
if "pretokenizer_cases" in reference:
    print(f'{len(reference["pretokenizer_cases"])} raw pretokenizer boundary cases passed on full asset and subset')
