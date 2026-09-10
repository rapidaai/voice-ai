# LiveKit Native Comparison

## Task Acceptance And Scope

The parent task is **Standard LiveKit EOS alignment**. Preserve manual model
selection, selected local model assets, current thresholds and options, and
packet contracts. Parent acceptance includes intentional cancellation, stale
completion suppression, and session release. Allowed parent scope is the
LiveKit package only; no factory, VAD, Pipecat, options, or UI edits here.

This worker owns only new `testdata/benchmark/**`, `turn_detector_parity_test.go`,
and `turn_detector_bench_test.go`. Production constants, errors, and the
`Predict(text)` signature are untouched. The separately owned `PredictContext`
implementation is exercised, not changed. Rollback is removal of these new
harness files. Preserve all other workers' edits.

Acceptance for this harness:

- Execute the actual local Python reference methods with explicitly selected
  local model/tokenizer assets. Capture source SHA, asset digests, runtime
  versions/build digests, exact prompts, IDs, probabilities, and decisions.
- Separate prompt parity, token parity on identical prompts, raw inference,
  `Predict`, and the model pipeline. Preserve the configured threshold, using
  `p >= threshold` for Rapida decisions, not publisher per-language thresholds.
- Report full-corpus labeled diagnostics separately from implementation parity.
  Do not infer real-world accuracy from either this curated corpus or parity.
- Run quality first, then only three latency cases in seven serial rounds with
  alternating Python-first / Go-first order. No experiment runs before tests
  finish and the operator approves them.

Risks: stale completion, cancellation races, session release during inference,
and tokenizer drift. Native model benchmarks do not verify the EOS packet or
timer lifecycle. The parent race/integration tests remain necessary. Other
risks include mismatched model/tokenizer revisions, source changes during a
run, thermal drift, and non-equivalent wrapper/scheduling costs.

## Reference Boundary

`reference.py` compiles unchanged AST class bodies from the actual local
`base.py`, `english.py`, and `multilingual.py`, plus the original model constants.
It executes the original `EOUModelBase.predict_end_of_turn`, `_format_chat_ctx`,
and runner `run` methods. It does not import unrelated LiveKit services or
replace text preparation with a hand-written approximation.

Initialization is intentionally replaced: explicit local ONNX/tokenizer files,
the publisher's pinned chat template, CPU execution, one intra-op thread, one
inter-op thread, sequential execution, and all graph optimizations. The
reference's adaptive thread count and dynamic-block setting are not used.
`PreTrainedTokenizerFast` loads the exact local tokenizer JSON; it does not
silently substitute the tokenizer from a publisher model revision.

The local executor directly calls the original runner. The Python `async_eot`
stage includes the original async method and `asyncio.wait_for`, on a reused
event loop. It does **not** include LiveKit process IPC, worker scheduling, or
network inference. Its blocking local executor cannot emulate worker timeout
preemption. The Go `production_context` stage formats history, creates a
deadline with `context.WithTimeout(b.Context(), predictionTimeout)`, calls
`PredictContext`, and cancels the context.

Other stages are `format`, `tokenize`, `inference`, `predict`, and `pipeline`.
`predict` consumes the identical Python prompt. `pipeline` includes each
implementation's own formatting and `Predict` path without the async wrapper.
These are **model pipeline** measurements, never full voice latency. Python
format timing includes copying selected messages; Go also selects history.
Python `runner_json` timing is retained separately with no Go equivalent.

## Local Provenance

Inspected checkout commit: `ab6e8a1caf0b87ae019173aae28f26babb510776`.
Source hashes are recomputed on every run, including `models.py` and `uv.lock`.

| File | SHA-256 |
| --- | --- |
| `base.py` | `90bdb5b65caff10f29e75e92790634548ffdd9fe9c215b358e22e4e189d27db5` |
| `english.py` | `541a6f3b58eddbbb50d172f72c165081ef9fecd0d3366dbf76631b199c649954` |
| `multilingual.py` | `a03e0434f5fe7f238a63c11c5df0246bf408e295909e24adb07428f024af5e24` |
| `model_q8.onnx` | `78eb599253570f9e488337b44d05cba559740b1a6c355356a95869e2626ba73f` |
| `model_q8_multilingual.onnx` | `70f9870a0e10236d0cae2886b9281b587ebcc25a0b3aac1e85822c652fa5277f` |
| `tokenizer.json` | `6f0dc4b1306b1b17da46b55b696c8f54ba7cb05a1b91dc526fd143f31e882fcb` |

`metadata.json` pins publisher commits and template/config/model digests for
the revisions declared by this reference checkout. Both local model digests
differ from those declared publisher revisions. The supplied single tokenizer
matches the English revision, not the multilingual revision's tokenizer.
English's publisher template uses special role tokens, while multilingual uses
role names plus newlines. These facts are recorded as pairing warnings, not
silently fixed. Code-path alignment and deployment-asset compatibility are
separate questions. A selected-local-assets run is not a stock publisher release.

Go checks model/tokenizer/library digests and verifies the actually loaded ORT
library using `lsof` on macOS or `/proc/self/maps` on Linux. Python checks that
its ORT version matches the selected Go library. The Python extension and Go
shared library are separately hashed builds of the **same release**, not a
claim that both language bindings execute the same binary bytes.

## Setup And Tests

The inspected Python environment initially had NumPy 1.26.4 and ORT 1.16.0,
but no Transformers or Tokenizers. Pinned tokenizer dependencies were installed
in a separate temporary target, leaving that environment unchanged. No model
downloads are performed by the reference or experiment runner.

From the repository root:

```sh
export PKG=./api/assistant-api/internal/end_of_speech/internal/livekit
export BENCH="$PKG/testdata/benchmark"
export PYTHON=/private/tmp/rapida-pipecat-benchmark/bin/python
export PYTHONPATH=/private/tmp/rapida-livekit-benchmark-deps
export CHECKOUT=/Users/prashant.srivastav/Documents/codes/lexatic/wip/livekit/agents
export ASSETS=/Users/prashant.srivastav/Documents/codes/lexatic/infra/azure/assistant-models/models/livekit_turn
export ORT_LIBRARY=/Users/prashant.srivastav/Documents/codes/lexatic/onnxruntime-osx-arm64-1.16.0/lib/libonnxruntime.1.16.0.dylib
export GOCACHE=/private/tmp/voice-ai-gocache
export LIVEKIT_TURN_MODEL_PATH="$ASSETS/model_q8.onnx"
export LIVEKIT_TURN_MULTI_MODEL_PATH="$ASSETS/model_q8_multilingual.onnx"
export LIVEKIT_TURN_TOKENIZER_PATH="$ASSETS/tokenizer.json"

# Only needed to recreate the separate dependency target.
"$PYTHON" -m pip install --target /private/tmp/rapida-livekit-benchmark-deps \
  transformers==4.57.6 tokenizers==0.22.2 jinja2==3.1.6 numpy==1.26.4

LIVEKIT_REFERENCE_CHECKOUT="$CHECKOUT" LIVEKIT_REFERENCE_TOKENIZER="$ASSETS/tokenizer.json" \
  "$PYTHON" -B -m unittest discover -s "$BENCH" -p 'test_*.py' -v

env GOCACHE=/private/tmp/voice-ai-gocache go test -race -tags=integration -count=1 ./api/assistant-api/internal/end_of_speech/...

PATHS="$PKG/turn_detector_parity_test.go,$PKG/turn_detector_bench_test.go,$BENCH/README.md,$BENCH/corpus.json,$BENCH/metadata.py,$BENCH/metadata.json,$BENCH/reference.py,$BENCH/report.py,$BENCH/run.py,$BENCH/test_harness.py"
env GOCACHE=/private/tmp/voice-ai-gocache just agent-finalize "$PATHS"
```

Unset `LIVEKIT_BENCHMARK_REFERENCE` and `LIVEKIT_BENCHMARK_OUTPUT` during parent
tests. Native parity is opt-in; the harness unit tests need no model inference.
The source/tokenizer Python test exercises all 29 diagnostic inputs under both
reference classes, without invoking ONNX. Existing native CGO include/link flags
must point at the selected runtime. A different ORT release requires explicitly
matching both Python and Go; there is no automatic fallback.

## Approved Experiments

Freeze source, finish the required tests, then build once without `-race`.
Retain the build command, source commit/diff, binary, and all output files.
Do not run alongside another quality/latency experiment or CPU-heavy workload.
The runner uses a local exclusive lock to prevent overlap with itself.

```sh
go test -tags=integration -c -o /private/tmp/rapida-livekit-benchmark.test "$PKG"

# Run one command to completion before starting the other. Outputs must be new directories.
"$PYTHON" -B "$BENCH/run.py" --approve-experiments \
  --checkout "$CHECKOUT" --model-type en --model "$ASSETS/model_q8.onnx" \
  --tokenizer "$ASSETS/tokenizer.json" --ort-library "$ORT_LIBRARY" --threshold 0.0289 \
  --go-binary /private/tmp/rapida-livekit-benchmark.test \
  --output /private/tmp/rapida-livekit-en-comparison --rounds 7 --iterations 10

"$PYTHON" -B "$BENCH/run.py" --approve-experiments \
  --checkout "$CHECKOUT" --model-type multilingual --model "$ASSETS/model_q8_multilingual.onnx" \
  --tokenizer "$ASSETS/tokenizer.json" --ort-library "$ORT_LIBRARY" --threshold 0.0289 \
  --go-binary /private/tmp/rapida-livekit-benchmark.test \
  --output /private/tmp/rapida-livekit-multilingual-comparison --rounds 7 --iterations 10
```

Selection and threshold are mandatory inputs; these examples preserve current
selection paths and the current 0.0289 default. No language-based model switch
or threshold tuning occurs. Quality uses all 29 cases. Latency uses only
`complete_en`, `repetition_incomplete`, and `long_left_truncation`, with three
warmups per stage and ten measured iterations per round. Seven rounds alternate
Python/Go order, starting with Python. Startup, model loading, hashing, and Go
compilation are excluded from timed stages. Every stage and process is serial.

Outputs include `quality/reference.json`, `quality/go.json`, `quality.json`,
`round-01` through `round-07` raw artifacts/logs, and `latency.json`. The Go test
exports discrepancies even when parity assertions fail. The driver retains
that failure status, reports it, and can still time the selected paths; this
does not waive parity failure. Missing/invalid predictions abort reporting.
Source drift during an experiment fails the run. Execution-time source hashes
do not by themselves certify that a prebuilt binary used those source bytes.

To regenerate a quality report separately:

```sh
"$PYTHON" -B "$BENCH/report.py" "$RESULTS/quality/reference.json" \
  "$RESULTS/quality/go.json" "$RESULTS/quality-summary.json"
```

Round medians and raw samples are descriptive. They are not significance tests
or request p95/p99. Constructor cost is excluded; Go allocation counts cover
the Go heap only. An unequal-prompt pipeline ratio must not be described as
equivalent-work speedup. Async and deadline-bearing stages expose wrapper costs
but are not end-to-end equivalence claims.

## Quality Interpretation

Labels were hand-authored before model inference. The corpus includes complete
and incomplete conjunctions, repetitions, short replies, history filtering and
truncation, Unicode, digits, and multiple languages. `repetition_incomplete` is
the exact user sample: "I I just I I just just wanted to talk to you and then".
It is labeled incomplete because the final conjunction leaves the utterance
unfinished. Capture the current Go quality baseline before changing English
formatting, then compare both implementations on these unchanged local assets.

Curated results are named `diagnostic_label_agreement`. Null labels exclude
punctuation-only, whitespace-only, and literal-special-token probes from label
metrics, but retain them for parity. False completion rate uses incomplete
labels as its denominator; missed completion rate uses complete labels. Report
per-language counts; non-English inputs are stress cases for the English model.

An independent corpus can be passed using `--corpus`; retain the three latency
case names for `run.py`, or use `reference.py` and `report.py` for quality only.
Its manifest must explicitly declare `kind: independent_dataset`, `source`,
`revision`, `split`, `license`, `sampling`, `label_policy`, and `holdout_status`,
with the same case schema. Dataset rows are not bundled; the separately
evaluated published English subset is described below.
The report calls those results `dataset_label_accuracy`, conditional on the
provided provenance; it does not verify holdout independence or claim general
real-world accuracy. Do not tune thresholds on the evaluation corpus.

## Published Text Subset

`dataset.py` exports a pinned English validation subset without downloading
audio. It checks revision-bearing asset URL strings before discarding those
URLs. It preserves original silence-span labels, applies 500 ms transcript lag,
and selects one score 200 ms into each pause lasting at least 200 ms. These are
snapshot probes, not full streaming-harness or live-call metrics.

```sh
curl -fsSL 'https://datasets-server.huggingface.co/rows?dataset=livekit%2Feot-bench-data&config=en&split=validation&offset=0&length=100' -o /private/tmp/rapida-livekit-eot-rows.json
"$PYTHON" -B "$BENCH/dataset.py" /private/tmp/rapida-livekit-eot-rows.json /private/tmp/rapida-livekit-eot-corpus.json
"$PYTHON" -B -m unittest discover -s "$BENCH" -p 'test_*.py'
```

Pass the generated corpus to `reference.py --corpus` for quality-only runs,
then run `TestLiveKitNativeParity` and `report.py` against that reference.
`run.py` intentionally requires the three curated latency case names, so it is
not used for this external quality-only subset. See `RESULTS.md` for measured
outcomes, source dates, exclusions, and the dataset license discrepancy.
