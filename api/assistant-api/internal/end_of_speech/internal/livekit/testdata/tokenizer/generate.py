"""Generate native token-ID fixtures from an existing pinned asset, without downloads."""

import argparse
import hashlib
import json
from pathlib import Path

import tokenizers


ASSET_SHA256 = "6f0dc4b1306b1b17da46b55b696c8f54ba7cb05a1b91dc526fd143f31e882fcb"
ASSET_REVISION = "ebcab0c09c2b62d926e92180d364df3aaae68a09"
CASES = [
    ("empty", ""),
    ("repeated_space", "foo  bar"),
    ("three_spaces", "foo   bar"),
    ("leading_trailing_space", "  foo  bar  "),
    ("spaces_only", "     "),
    ("repeated_newline", "foo\n\nbar"),
    ("newline_spaces", "foo\n \n  bar"),
    ("trailing_newlines", "foo\n\n"),
    ("tabs_crlf", "foo\t\tbar\r\nnext"),
    ("ascii_digits", "1234567890"),
    ("word_digits", "abc123def 42.50"),
    ("space_before_digits", "foo  12 bar"),
    ("contractions", "I'm you're we've he'll she'd can't John's"),
    ("uppercase_contractions", "I'M YOU'RE CAN'T"),
    ("unicode_letters", "caf\u00e9 na\u00efve \u4f60\u597d \u041f\u0440\u0438\u0432\u0435\u0442"),
    ("unicode_combining", "cafe\u0301"),
    ("unicode_numbers", "A\u0661\u0662\u00b2\u2167\u00bdB"),
    ("unicode_space", "foo\u00a0\u00a0bar\u2003baz\u2028end"),
    ("unicode_symbols", "hi \U0001f642\U0001f680!"),
    ("punctuation", "hi... (yes?) -- ok!"),
    ("special_only", "<|im_start|><|im_end|><|endoftext|>"),
    ("special_adjacent", "foo<|im_start|>bar<|im_end|>"),
    ("special_digits", "12<|im_end|>34"),
    ("chat", "<|im_start|>user\nfoo  bar 123<|im_end|>\n<|im_start|>assistant\nI'm here"),
    ("special_prefix", "<repo_name>x<reponame>y<|user|><|assistant|><|pad|>"),
    ("special_whitespace", "<|im_start|> \t\n <|im_end|>"),
    ("special_strip", "  foo <|im_start|>  bar  <|im_end|>  "),
    ("all_whitespace", "foo\t\n\v\f\r \u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000bar"),
    ("non_whitespace_controls", "foo\u001c\u001e\u001f\u180e\u200b\ufeffbar"),
]


def generate(asset_path, output):
    if tokenizers.__version__ != "0.22.2":
        raise RuntimeError("generation requires tokenizers==0.22.2 from LiveKit's uv.lock")
    asset_bytes = asset_path.read_bytes()
    assert hashlib.sha256(asset_bytes).hexdigest() == ASSET_SHA256
    asset = json.loads(asset_bytes)
    assert asset["normalizer"] == {
        "type": "Sequence",
        "normalizers": [
            {"type": "Replace", "pattern": {"Regex": r"\s+"}, "content": " "},
            {"type": "Strip", "strip_left": True, "strip_right": True},
        ],
    }
    assert asset["pre_tokenizer"] == {
        "type": "Sequence",
        "pretokenizers": [
            {"type": "Digits", "individual_digits": True},
            {"type": "ByteLevel", "add_prefix_space": False, "trim_offsets": True, "use_regex": True},
        ],
    }
    vocab = asset["model"]["vocab"]
    merges = [tuple(pair) for pair in asset["model"]["merges"]]
    specials = {item["content"]: item["id"] for item in asset["added_tokens"] if item["special"]}
    source_vocab_size = len(vocab)
    vocab = vocab | specials
    printable = list(range(33, 127)) + list(range(161, 173)) + list(range(174, 256))
    missing = [value for value in range(256) if value not in printable]
    byte_chars = {value: chr(value) for value in printable}
    byte_chars.update({value: chr(256 + offset) for offset, value in enumerate(missing)})
    used_tokens = set(byte_chars.values()).intersection(vocab) | set(specials)
    native = tokenizers.Tokenizer.from_file(str(asset_path))
    cases = []
    byte_samples = []
    for name, text in CASES:
        encoding = native.encode(text, add_special_tokens=False)
        cases.append({"name": name, "text": text, "tokens": encoding.tokens, "ids": encoding.ids})
        used_tokens.update(encoding.tokens)
        byte_samples.append("".join(byte_chars[value] for value in text.encode("utf-8")))
        byte_samples.append("".join(encoding.tokens))

    # Keep candidate merges from raw and prepared text so missing stages remain detectable.
    selected_merges = [pair for pair in merges if any("".join(pair) in sample for sample in byte_samples)]
    for pair in selected_merges:
        used_tokens.update((*pair, "".join(pair)))
    subset = {
        **asset,
        "model": {
            **asset["model"],
            "vocab": {token: vocab[token] for token in sorted(used_tokens, key=vocab.get)},
            "merges": selected_merges,
        },
    }
    subset_bytes = (json.dumps(subset, ensure_ascii=True, indent=2) + "\n").encode()
    native_subset = tokenizers.Tokenizer.from_str(subset_bytes.decode())
    for case in cases:
        actual = native_subset.encode(case["text"], add_special_tokens=False).ids
        if actual != case["ids"]:
            raise AssertionError(f'subset mismatch for {case["name"]}: {actual} != {case["ids"]}')
    reference = {
        "provenance": {
            "asset_sha256": ASSET_SHA256,
            "asset_revision": ASSET_REVISION,
            "asset_url": f"https://huggingface.co/livekit/turn-detector/resolve/{ASSET_REVISION}/tokenizer.json",
            "native_tokenizers_version": tokenizers.__version__,
            "version_source": "LiveKit agents uv.lock",
            "native_hf_executed": True,
            "native_subset_verified": True,
            "add_special_tokens": False,
            "limitation": "Native ID equality for the bounded corpus, not exhaustive tokenizer or model-inference equivalence",
            "subset_sha256": hashlib.sha256(subset_bytes).hexdigest(),
            "source_vocab_size": source_vocab_size,
            "source_merge_count": len(merges),
            "subset_vocab_size": len(subset["model"]["vocab"]),
            "subset_merge_count": len(subset["model"]["merges"]),
        },
        "cases": cases,
    }
    output.mkdir(parents=True, exist_ok=True)
    (output / "tokenizer.json").write_bytes(subset_bytes)
    (output / "reference.json").write_text(json.dumps(reference, ensure_ascii=True, indent=2) + "\n")
    print(json.dumps(reference["provenance"], indent=2))
    print(f"Generated {len(cases)} bounded cases")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("asset", type=Path)
    parser.add_argument("--output", type=Path, default=Path(__file__).parent)
    args = parser.parse_args()
    generate(args.asset, args.output)
