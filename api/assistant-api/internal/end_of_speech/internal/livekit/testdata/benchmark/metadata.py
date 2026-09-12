"""Fetch small publisher metadata only. Never download or select model assets."""

import argparse
import hashlib
import json
from pathlib import Path
from urllib.request import urlopen


def fetch(revision):
    root = "https://huggingface.co"
    with urlopen(f"{root}/api/models/livekit/turn-detector/revision/{revision}?blobs=true", timeout=30) as response:
        info = json.load(response)
    commit = info["sha"]
    url = f"{root}/livekit/turn-detector/resolve/{commit}/tokenizer_config.json"
    with urlopen(url, timeout=30) as response:
        raw = response.read()
    config = json.loads(raw)
    files = {item["rfilename"]: item for item in info["siblings"]}
    return {
        "requested_revision": revision,
        "commit": commit,
        "config_url": url,
        "config_sha256": hashlib.sha256(raw).hexdigest(),
        "chat_template": config["chat_template"],
        "tokenizer_class": config["tokenizer_class"],
        "tokenizer_git_blob": files["tokenizer.json"]["blobId"],
        "model_sha256": files["onnx/model_q8.onnx"]["lfs"]["sha256"],
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--en-revision", required=True)
    parser.add_argument("--multilingual-revision", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    result = {"en": fetch(args.en_revision), "multilingual": fetch(args.multilingual_revision)}
    args.output.write_text(json.dumps(result, indent=2, ensure_ascii=True) + "\n")
