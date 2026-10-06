# Pipecat Comparison Harness

This harness measures feature extraction, inference on identical features, and
the complete audio-to-probability call. It does not measure SIP, VAD, STT,
transcript waiting, thread-pool scheduling, or assistant response latency.

`reference.py` loads the actual local Pipecat analyzer methods. It skips only the
root package's distribution-version banner and creates an explicit CPU session
with the source's thread/optimization settings for compatibility with ORT 1.16.
The feature-only stage uses the caller's actual nested padding function and
vendored feature function. Full prediction calls `_predict_endpoint` unchanged.

Nine inputs cover the six feature fixtures, two speech WAVs in the reference
checkout, and a halfway-cut question. WAV conversion happens before timing.
Audio and feature artifacts, source/model hashes, dependency versions, individual
timing rounds, and model metadata differences are retained in the output folder.

## Reproduce

Use an isolated Python environment with `numpy==1.26.4`, `onnxruntime==1.16.0`,
`onnx==1.17.0`, `soxr==1.0.0`, `loguru==0.7.3`, and `pydantic==2.10.6` to match
the measured baseline. Select the same ONNX model for both implementations.
Native Go dependencies must be configured as for assistant-api tests.

From the repository root, with `PYTHON`, `PIPECAT_CHECKOUT`, `MODEL`, and `RESULTS`
set to the local environment, checkout, model, and output paths:

```sh
OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 VECLIB_MAXIMUM_THREADS=1 PYTHONDONTWRITEBYTECODE=1 \
  "$PYTHON" api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/reference.py \
  "$PIPECAT_CHECKOUT" "$MODEL" "$RESULTS" --rounds 7 --iterations 10

PIPECAT_TURN_MODEL_PATH="$MODEL" PIPECAT_BENCHMARK_REFERENCE="$RESULTS/reference.json" \
  go test -tags=integration -run '^TestPipecatNativeParity$' -v \
  ./api/assistant-api/internal/end_of_speech/internal/pipecat

GOMAXPROCS=1 PIPECAT_TURN_MODEL_PATH="$MODEL" PIPECAT_BENCHMARK_REFERENCE="$RESULTS/reference.json" \
  go test -tags=integration -run '^$' -bench '^BenchmarkPipecatNative$' \
  -benchmem -benchtime=10x -count=7 -cpu=1 \
  ./api/assistant-api/internal/end_of_speech/internal/pipecat > "$RESULTS/go.bench"

"$PYTHON" api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/summarize.py \
  "$RESULTS/reference.json" "$RESULTS/go.bench" "$RESULTS/summary.json"
```

Run the timed processes sequentially without race instrumentation or concurrent
CPU-heavy jobs. Compare round medians and retain outliers. Go allocation counts
cover the Go heap, not native ONNX allocations. Constructor cost is excluded.

The Pipecat checkout currently declares a newer ORT version than this controlled
baseline. Cross-version results must be labeled separately. Exact feature bit
identity, corpus-level semantic accuracy, and full event-loop parity are not
implied by these measurements.

## Labeled Quality Evaluation

`quality_corpus.py` samples 256 row indices without replacement using seed
20260909 from `pipecat-ai/smart-turn-data-v3.1-test`, revision
`2a9377baf2bbc73ba176c4505fe4adf988288fe9`. Its physical split is named `train`,
although the publisher describes this dataset as testing data. Holdout status
for the tested v3.2 model is not independently established.

Use Python 3.11 with the dependencies above and `soundfile==0.13.1`.
The downloader saves the selection before inference, checks revision and audio
hashes, resumes cached downloads, and fails without replacing missing rows.
Audio stays in the requested output directory; do not commit downloaded audio.

Each recording has four conditions: clean, 250 ms appended silence, seeded
20 dB Gaussian noise, and an 8 kHz mu-law round trip. The manifest specifies
the transformations. All conditions retain the original completion label.
Appending silence can discard earlier context at the model's eight-second
boundary. Noise or bandwidth loss can remove information relevant to the label.

Before changing the baseline, compile its integration test binary using
`go test -tags=integration -c -o "$BASELINE_BINARY"` with the Pipecat package
path. This preserves the baseline for a sequential comparison after changes.
With `QUALITY` pointing to a temporary directory and `BASELINE_BINARY` to that
absolute binary path:

```sh
"$PYTHON" api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/quality_corpus.py \
  "$QUALITY/corpus" --count 256 --seed 20260909

OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1 VECLIB_MAXIMUM_THREADS=1 PYTHONDONTWRITEBYTECODE=1 \
  "$PYTHON" api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/reference.py \
  "$PIPECAT_CHECKOUT" "$MODEL" "$QUALITY/reference" \
  --corpus "$QUALITY/corpus/corpus.json" --skip-timing

PIPECAT_TURN_MODEL_PATH="$MODEL" PIPECAT_BENCHMARK_REFERENCE="$QUALITY/reference/reference.json" \
  PIPECAT_QUALITY_OUTPUT="$QUALITY/rapida.json" \
  "$BASELINE_BINARY" -test.run '^TestPipecatQualityPredictions$' -test.count=1

PIPECAT_TURN_MODEL_PATH="$MODEL" PIPECAT_BENCHMARK_REFERENCE="$QUALITY/reference/reference.json" \
  PIPECAT_QUALITY_OUTPUT="$QUALITY/fft.json" \
  go test -tags=integration -run '^TestPipecatQualityPredictions$' -count=1 \
  ./api/assistant-api/internal/end_of_speech/internal/pipecat

"$PYTHON" api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark/quality_report.py \
  "$QUALITY/corpus/corpus.json" "$QUALITY/reference/reference.json" \
  "$QUALITY/rapida.json" "$QUALITY/fft.json" "$QUALITY/summary.json"

"$PYTHON" -m unittest discover \
  -s api/assistant-api/internal/end_of_speech/internal/pipecat/testdata/benchmark \
  -p 'test_quality_report.py' -v
```

Record Go source and binary hashes, build commands, and native ORT linkage
alongside results. The Go exporter verifies the model hash and checks feature
error <= 2e-5, probability error <= 1e-4, and exact completion-decision agreement.
It exports results even when a parity assertion fails, so differences remain
available for diagnosis. Missing predictions make the report fail.

Quality is reported per condition, not by counting variants as independent
recordings. False completion rate uses incomplete labels as its denominator;
missed completion rate uses complete labels. Rates include Wilson 95% intervals.
Paired accuracy differences use 2000 fixed-seed source bootstrap resamples,
keeping conditions together. A zero-width bootstrap interval with no observed
disagreements is not proof of equivalence; the report also gives a one-sided
95% upper bound for zero discordant source recordings under independence.

Language, source, and synthetic groups are descriptive small-sample diagnostics.
This tests the audio model, not VAD gating, final transcript handling, live
interruption behavior, or end-to-end completion delay.

## Waveform Arithmetic Regression

`../waveform_scaling/generate.py` pins deterministic input/output bit digests
against NumPy 1.26.4. The Go waveform tests need no Python installation or public
audio download. They cover reduction-lane, pairwise-block, buffer, and model-window
boundaries, quiet DC input, cancellation, constant input, and signed zero.

The production arithmetic matches the controlled NumPy 1.26 reference. NumPy 2
changes Python-scalar promotion, so it is not a bitwise-equivalent reference.
Retain the fixed runtime versions when regenerating fixtures or comparing
model probabilities. `QUALITY.md` records both the original failures and the fix.

## Configuration Compatibility

Pipecat exposes completion threshold, transcript safety wait (`fallback_timeout`),
and received silence limit (`extended_timeout`). Their defaults are 0.5, 500 ms,
and 3000 ms. Its obsolete `quick_timeout` control is removed, not reassigned to
a different budget. Existing explicit silence limits, including 2000 ms, remain
unchanged when the UI loads saved configuration.

LiveKit exposes minimum delay (`quick_timeout`), maximum delay
(`extended_timeout`), completion threshold, model, and maximum history turns.
The defaults remain 250 ms, 3000 ms, 0.0289, `en`, and six turns. Its old
`fallback_timeout` field is an alias for the minimum delay, not an independent
control.

UI hydration carries supported legacy timeout aliases into the active keys
before removing obsolete controls. Canonical numeric values take precedence.
Stored key strings and backend alias handling are unchanged; no database
migration runs. Assistant option names are registered in `internal/options/audio.go`.
Go integration tests feed the actual provider JSON defaults to both constructors.
