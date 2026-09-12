"""Generate native token-ID fixtures from an existing pinned asset, without downloads."""

import argparse
import hashlib
import json
from pathlib import Path

import tokenizers


ASSET_SHA256 = "6f0dc4b1306b1b17da46b55b696c8f54ba7cb05a1b91dc526fd143f31e882fcb"
ASSET_REVISION = "ebcab0c09c2b62d926e92180d364df3aaae68a09"
MULTILINGUAL_ASSET_SHA256 = "9c5ae00e602b8860cbd784ba82a8aa14e8feecec692e7076590d014d7b7fdafa"
MULTILINGUAL_ASSET_REVISION = "87e35fcb1e60a569bea70346191c4886ea92e281"
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
MULTILINGUAL_CASES = [
    ("chinese", "\u4f60\u597d\uff0c\u6211\u60f3\u9884\u8ba2\u660e\u5929\u7684\u673a\u7968\u3002"),
    ("japanese", "\u3059\u307f\u307e\u305b\u3093\u3001\u3082\u3046\u4e00\u5ea6\u304a\u9858\u3044\u3057\u307e\u3059\u3002"),
    ("korean", "\uc548\ub155\ud558\uc138\uc694, \uc608\uc57d\uc744 \ubcc0\uacbd\ud558\uace0 \uc2f6\uc5b4\uc694."),
    ("hindi", "\u0928\u092e\u0938\u094d\u0924\u0947, \u092e\u0941\u091d\u0947 \u092e\u0926\u0926 \u091a\u093e\u0939\u093f\u090f\u0964"),
    ("arabic", "\u0645\u0631\u062d\u0628\u0627\u064b\u060c \u0623\u0631\u064a\u062f \u062a\u063a\u064a\u064a\u0631 \u0627\u0644\u062d\u062c\u0632."),
    ("russian", "\u041f\u043e\u0434\u043e\u0436\u0434\u0438\u0442\u0435, \u044f \u0435\u0449\u0451 \u043d\u0435 \u0437\u0430\u043a\u043e\u043d\u0447\u0438\u043b."),
    ("spanish", "\u00bfPuedes repetirlo? A\u00fan no he terminado."),
    ("multilingual_chat", "<|im_start|>user\n\u4f60\u597d, necesito ayuda.  \u0927\u0928\u094d\u092f\u0935\u093e\u0926!<|im_end|>\n<|im_start|>assistant\n\u0645\u0631\u062d\u0628\u0627\u064b"),
    ("mixed_case_contractions", "I'M I'd WE'Re we'VE He'LL CAN'T John'S"),
    ("punctuation_newlines", "hi!\r\n\n  next?\n\t\r\n done  "),
]


def generate(asset_path, output, variant="english"):
    if tokenizers.__version__ != "0.22.2":
        raise RuntimeError("generation requires tokenizers==0.22.2 from LiveKit's uv.lock")
    asset_bytes = asset_path.read_bytes()
    asset_sha256 = MULTILINGUAL_ASSET_SHA256 if variant == "multilingual" else ASSET_SHA256
    asset_revision = MULTILINGUAL_ASSET_REVISION if variant == "multilingual" else ASSET_REVISION
    assert hashlib.sha256(asset_bytes).hexdigest() == asset_sha256
    asset = json.loads(asset_bytes)
    assert asset["normalizer"] == ({"type": "NFC"} if variant == "multilingual" else {
        "type": "Sequence",
        "normalizers": [
            {"type": "Replace", "pattern": {"Regex": r"\s+"}, "content": " "},
            {"type": "Strip", "strip_left": True, "strip_right": True},
        ],
    })
    assert asset["pre_tokenizer"] == ({
        "type": "Sequence",
        "pretokenizers": [
            {
                "type": "Split",
                "pattern": {"Regex": r"(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+"},
                "behavior": "Isolated",
                "invert": False,
            },
            {"type": "ByteLevel", "add_prefix_space": False, "trim_offsets": False, "use_regex": False},
        ],
    } if variant == "multilingual" else {
        "type": "Sequence",
        "pretokenizers": [
            {"type": "Digits", "individual_digits": True},
            {"type": "ByteLevel", "add_prefix_space": False, "trim_offsets": True, "use_regex": True},
        ],
    })
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
    for name, text in CASES + (MULTILINGUAL_CASES if variant == "multilingual" else []):
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
            "asset_sha256": asset_sha256,
            "asset_revision": asset_revision,
            "asset_url": f"https://huggingface.co/livekit/turn-detector/resolve/{asset_revision}/tokenizer.json",
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
    if variant == "multilingual":
        byte_values = {char: value for value, char in byte_chars.items()}
        reference["pretokenizer_cases"] = [
            {
                "name": case["name"],
                "text": case["text"],
                "boundaries": [
                    bytes(byte_values[char] for char in piece).decode("utf-8")
                    for piece, _ in native.pre_tokenizer.pre_tokenize_str(case["text"])
                ],
            }
            for case in cases
        ]
    output.mkdir(parents=True, exist_ok=True)
    (output / "tokenizer.json").write_bytes(subset_bytes)
    (output / "reference.json").write_text(json.dumps(reference, ensure_ascii=True, indent=2) + "\n")
    print(json.dumps(reference["provenance"], indent=2))
    print(f"Generated {len(cases)} bounded cases")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("asset", type=Path)
    parser.add_argument("--variant", choices=("english", "multilingual"), default="english")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    output = args.output or (Path(__file__).parent / "multilingual" if args.variant == "multilingual" else Path(__file__).parent)
    generate(args.asset, output, args.variant)
