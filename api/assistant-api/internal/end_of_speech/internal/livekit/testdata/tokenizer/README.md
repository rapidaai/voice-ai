# Native Tokenizer Fixtures

## English

These 29 bounded cases use token IDs and merge rules from the production tokenizer
pinned in `docker/assistant-api/native-deps.lock`:

- Repository: `livekit/turn-detector`
- Revision: `ebcab0c09c2b62d926e92180d364df3aaae68a09`
- Asset SHA-256: `6f0dc4b1306b1b17da46b55b696c8f54ba7cb05a1b91dc526fd143f31e882fcb`
- Declared preparation: replace every whitespace run with one space, then strip
  both ends of each non-special segment after added-token extraction.
- Declared pretokenization: `Digits(individual_digits=true)`, then
  `ByteLevel(add_prefix_space=false, trim_offsets=true, use_regex=true)`.

`generate.py` verifies the supplied local asset digest and requires native
`tokenizers==0.22.2`, matching LiveKit agents' `uv.lock`. It obtains every expected
ID by running the complete asset with `add_special_tokens=False`, then checks the
reduced asset with the same native engine. `reference.json` records the native
version, source revision/digest, subset digest, tokens, and expected IDs.

The subset retains original IDs, relative merge order, and all declared stages.
Candidate merges from both raw and prepared text remain present, so omitted
stages are detectable. The source has 49,152 vocabulary entries and 48,900 merges;
the subset has 459 entries and 204 merges, including added special-token IDs.

The original 25 inputs are retained. Four additional cases cover whitespace-only
segments between specials, stripping next to specials, all 25 Go `IsSpace` code
points, and adjacent control/format characters that must not become spaces.

Regenerate and verify with an existing asset and native `tokenizers==0.22.2`:

```sh
python3 generate.py /path/to/pinned/tokenizer.json
python3 verify_native.py /path/to/pinned/tokenizer.json
```

Ordinary Go tests require neither the full asset nor Python. These native outputs
supersede the earlier boundary-only expectations, which omitted preparation.
For example, `foo  bar` and `foo\n\nbar` both produce `[15236, 2753]`.

## Multilingual

`multilingual/` contains 39 native token-ID and raw pretokenizer boundary cases
from revision `87e35fcb1e60a569bea70346191c4886ea92e281`. The source asset SHA-256 is
`9c5ae00e602b8860cbd784ba82a8aa14e8feecec692e7076590d014d7b7fdafa`.
This asset applies NFC composition, followed by the pinned isolated regex split
and `ByteLevel(add_prefix_space=false, trim_offsets=false, use_regex=false)`.

The subset retains 825 vocabulary entries and 555 merges. Native verification
checks both full and reduced assets, including Unicode composition, mixed-case
contractions, multilingual chat, whitespace, and special-token boundaries.

```sh
python3 generate.py /path/to/pinned/multilingual/tokenizer.json --variant multilingual
python3 verify_native.py /path/to/pinned/multilingual/tokenizer.json --variant multilingual
```

These fixtures do not change the deployed model or tokenizer selection.

## Limits

Evidence is limited to exact native ID equality on the bounded corpus. It does
not establish exhaustive Unicode/tokenizer equivalence or model-inference parity.
The reduced vocabulary is sufficient only for the committed corpus.
