"""Check the bounded fixture corpus with LiveKit's pinned native tokenizer."""

import hashlib
import json
from pathlib import Path
import sys

import tokenizers


if tokenizers.__version__ != "0.22.2":
    raise RuntimeError("verification requires tokenizers==0.22.2 from LiveKit's uv.lock")

reference = json.loads(Path(__file__).with_name("reference.json").read_text())
asset_path = Path(sys.argv[1])
if hashlib.sha256(asset_path.read_bytes()).hexdigest() != reference["provenance"]["asset_sha256"]:
    raise RuntimeError("tokenizer asset digest does not match the pinned fixture source")

subset_path = Path(__file__).with_name("tokenizer.json")
if hashlib.sha256(subset_path.read_bytes()).hexdigest() != reference["provenance"]["subset_sha256"]:
    raise RuntimeError("tokenizer subset digest does not match the fixture provenance")

for label, path in [("full asset", asset_path), ("subset", subset_path)]:
    tokenizer = tokenizers.Tokenizer.from_file(str(path))
    for case in reference["cases"]:
        actual = tokenizer.encode(case["text"], add_special_tokens=False).ids
        if actual != case["ids"]:
            raise AssertionError(f'{label} {case["name"]}: expected {case["ids"]}, got {actual}')

print(f'Native tokenizers {tokenizers.__version__}: {len(reference["cases"])} cases passed on full asset and subset')
