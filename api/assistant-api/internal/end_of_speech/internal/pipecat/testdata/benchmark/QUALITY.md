# Pipecat Quality Evaluation

Baseline and follow-up measured September 9, 2026. The numerical-fix section
evaluates the updated worktree; baseline sections retain the original evidence.
The tested 400-point Gonum FFT is now integrated into production code.

## Production FFT Verification

The production FFT binary, without overlays, passes all 1024 frozen quality
cases. Probabilities and completion decisions exactly match both actual Pipecat
and the corrected DFT baseline. Maximum feature difference from Pipecat is
3.725290298e-9. Aggregate accuracy remains unchanged; the model's own errors
remain. This is reference parity, not a claim of perfect turn detection.

Each reusable scratch owns its FFT state and buffers. The unused direct-DFT
tables and legacy radix-2 implementation are removed. The existing Gonum and
regexp2 dependency versions are unchanged; `go.mod` marks their direct use.
Full EOS native integration and race suites pass. The production FFT timing
comparison, including retained host-contention outliers, is in `RESULTS.md`.

Artifacts in `/private/tmp/rapida-pipecat-numerics/` include
`production-fft-quality.json`, `production-fft-quality.log`, and
`production-quality-summary.json`. The report's `rapida` series is the
corrected DFT baseline and its `fft` series is the production FFT binary.

## Numerical Fix

The waveform-scaling correction resolves the parity failures below on the same
frozen corpus. No model, threshold, packet, VAD, dispatch, or timeout changes
were made. The initial FFT follow-up used a temporary overlay; the production
verification above repeats it after integration.

| Check Against Pipecat | Before | Corrected Go | Corrected FFT Overlay |
|---|---:|---:|---:|
| Completion-decision differences | 2/1024 | 0/1024 | 0/1024 |
| Probability errors above 1e-4 | 53/1024 | 0/1024 | 0/1024 |
| Maximum probability difference | 0.360198796 | 0 | 0 |
| Maximum feature difference | 8.893199265e-6 | 5.960464478e-8 | 3.725290298e-9 |

The old code accumulated waveform mean and variance in float64. Pipecat's
NumPy 1.26 path instead combines 8192-sample buffered reductions, 128-sample
pairwise leaves, and eight float32 accumulation lanes. The scalar epsilon is
added in float64 before the square root and the divisor is then cast to float32.
Both rounding details affect the model inputs.

The fix preserves this ordering in a bounded inline tree without recursion,
new helpers, or waveform copies. All 16 diagnostic waveforms and all 1024
quality predictions now match the controlled reference exactly. This pins
NumPy 1.26 arithmetic; it does not claim bitwise equivalence to NumPy 2, whose
Python-scalar promotion differs.

Regression coverage includes 45 deterministic input/output bit-digest fixtures
and every chunk length from 1 through 8192. The new fixtures fail against the
old scaling implementation. Broad EOS integration tests pass, including native
LiveKit English and multilingual models when the existing multilingual model
is configured through `LIVEKIT_TURN_MULTI_MODEL_PATH`.

Seven 100-call runs measured waveform scaling plus its benchmark input copy at
a median 0.414 ms per 128000 samples, with zero Go heap allocations. Seven
10-call rounds measured corrected DFT features at 88.625 ms and prediction at
176.881 ms. A later FFT-overlay run measured 14.322 ms and 125.667 ms;
other host workloads were active, so those separate runs are not a clean
paired speedup estimate. Original controlled timing remains in `RESULTS.md`.

Post-fix artifacts are under `/private/tmp/rapida-pipecat-numerics/`:
`fixed-quality.json`, `fixed-fft-quality.json`, `fixed-summary.json`,
`fixed-quality.log`, `fixed-fft-quality.log`, `fixed.log`,
`regression-before.log`, and the benchmark text files. The diagnostic log's
`scaled_differences` and probability fields measure the fix; its earlier
manual mean/variance calculations intentionally retain the old formula.

## Baseline Outcome

- FFT versus current Rapida: all 1024 probabilities are exactly equal, with no
  changed completion decisions or observed quality regressions.
- Both Go implementations versus actual Pipecat: 1022/1024 matching decisions.
  Probability differences exceed the preset 1e-4 tolerance on 53 cases.
- Full Pipecat parity fails. Equal aggregate accuracy is not exact alignment.
- A 16-case diagnostic reproduced Pipecat probabilities exactly when Go consumed
  Pipecat feature tensors. Differences return when Go computes the features.
  This isolates the observed gap to feature computation, not the FFT change.

Do not treat this evaluation as merge approval for exact Pipecat alignment.
Do not relax the probability tolerance to hide the failures.

## Baseline Corpus And Conditions

- Dataset: `pipecat-ai/smart-turn-data-v3.1-test`.
- Revision: `2a9377baf2bbc73ba176c4505fe4adf988288fe9`.
- Config `default`, physical split `train`, population 31473 rows.
- Selection: 256 uniform random row indices without replacement, Python 3.11.14
  `random.Random(20260909).sample`, frozen before inference. No replacements.
- Labels: 134 complete, 122 incomplete, supplied by `endpoint_bool`.
- Sources: 58 human, 198 synthetic, spanning 23 languages.
- Four conditions per source: clean, 250 ms appended silence, 20 dB Gaussian
  noise, and an 8 kHz G.711 mu-law round trip. See the corpus script for exact
  resampling, noise, and codec operations.

All implementations consume identical hashed 16 kHz mono float32 waveforms.
The Go loader verifies audio, feature, and model digests. Report generation
rejects missing or duplicate results, unselected rows, and inconsistent source
identity across conditions. Original labels are retained after transformations.

## Baseline Quality At Threshold 0.5

Complete means probability strictly greater than 0.5. Each row below contains
256 recordings. All three implementations have these aggregate counts, although
the noisy condition contains two case-level disagreements.

| Condition | Accuracy | Premature Completions / Incomplete | Missed Completions / Complete |
|---|---:|---:|---:|
| Clean | 241/256, 94.14% | 9/122, 7.38% | 6/134, 4.48% |
| Added 250 ms silence | 239/256, 93.36% | 10/122, 8.20% | 7/134, 5.22% |
| 20 dB Gaussian noise | 232/256, 90.62% | 13/122, 10.66% | 11/134, 8.21% |
| 8 kHz mu-law round trip | 240/256, 93.75% | 14/122, 11.48% | 2/134, 1.49% |

Clean accuracy has a Wilson 95% interval of 90.56%-96.42%. The corresponding
premature-completion interval is 3.93%-13.43%, and missed-completion interval is
2.07%-9.42%. These intervals assume independent source recordings and do not
establish production representativeness.

Human-only clean accuracy is 56/58 (96.55%, interval 88.27%-99.05%), with two
premature completions among 27 incomplete recordings. The sample is too small
for strong claims about human speech, individual languages, or accents.

For 116 recordings, appended silence also discards leading context under the
model's eight-second window. Those cases score 109/116; the remaining 140 score
130/140. This condition cannot be interpreted as a pure silence effect.

## Baseline Pairwise Differences

| Comparison | Decision Differences | Maximum Probability Difference |
|---|---:|---:|
| Current Rapida versus FFT overlay | 0/1024 | 0 |
| Pipecat versus current Rapida | 2/1024 | 0.360198796 |
| Pipecat versus FFT overlay | 2/1024 | 0.360198796 |

Both disagreements have incomplete dataset labels:

| Case | Language | Pipecat | Both Go Versions | Effect Versus Pipecat |
|---|---|---:|---:|---|
| `row_15456_noise_20db` | Spanish | 0.188876688 | 0.549075484 | Additional premature completion |
| `row_19772_noise_20db` | Danish | 0.568494201 | 0.500000000 | Corrected premature completion |

The equal error totals conceal one correctness regression and one correction.
The noisy-condition paired accuracy difference is zero, with a source-bootstrap
95% interval of -1.17 to +1.17 percentage points (2000 fixed-seed resamples).

No FFT-versus-Rapida differences were observed, but that does not prove universal
equivalence. With 256 independent sources, zero discordant sources gives an
approximately 1.16% one-sided 95% upper bound. The 1024 variants are not 1024
independent recordings.

Maximum feature error against Pipecat is 8.893199265e-6 for both Go versions,
within the existing 2e-5 tolerance. Nevertheless, probability tolerance fails on
53 cases. Small feature errors are not sufficient evidence of prediction parity
with this model. Further work must isolate audio-statistics and spectrogram
rounding differences; this evaluation does not establish which substage causes
each discrepancy.

## Baseline Runtime And Provenance

- Actual local Pipecat checkout commit:
  `1f8a513dd79c31f54bbd5b197fa26aefe242deca`. Inspected predictor sources are clean.
- Selected model SHA-256:
  `c02d673e1d0b7c1acfb3323b5792bb7b37aa7746538145dbf1e9a01dcecd99b7`.
- Same model file, ONNX Runtime 1.16.0 CPU, intra/inter threads 1; NumPy 1.26.4.
- Go 1.25.13, Apple M1 Pro. Both Go binaries link ORT 1.16.0.
- The Pipecat predictor methods are unchanged; their constructor is bypassed to
  create a controlled CPU session compatible with ORT 1.16. This is not a claim
  about the checkout's default newer runtime configuration.
- Rapida baseline is the dirty worktree, not merely HEAD. File and binary hashes
  are retained; the FFT overlay changes only feature extraction implementation.

Artifacts are under `/private/tmp/rapida-pipecat-quality/`: `corpus/corpus.json`,
`corpus/selection.json`, `reference/reference.json`, `rapida.json`, `fft.json`,
`summary.json`, `diagnostic.log`, `source-sha256.txt`, `go-linkage.txt`,
`go-build-info.txt`, and `fft-prototype.patch`. Raw audio and full-precision
probabilities remain there rather than being committed to the repository.

Reproduction commands and dependencies are in `README.md`. The actual Go runs
used separately compiled test binaries with `GOMAXPROCS=1` and
`-test.run '^TestPipecatQualityPredictions$' -test.v -test.count=1`, pointing
`PIPECAT_BENCHMARK_REFERENCE` and `PIPECAT_QUALITY_OUTPUT` at these artifacts.
Both quality test runs exited 1 because the Pipecat parity assertions failed,
while still exporting all 1024 results. This failure is retained intentionally.

## Limits

This is a sample of the publisher's public test corpus, not an independently
verified holdout for v3.2. Dataset labels were not independently relabeled.
Speaker/conversation clusters are unavailable; between-recording dependence is
not controlled. Most samples are synthetic. Noise and bandwidth loss can remove
information needed to interpret inherited labels.

No real SIP transport, VAD boundary selection, STT finalization, transcript
deadline, live interruption, or end-to-end completion delay is measured. The
original reported call audio was unavailable and was not reproduced here.
