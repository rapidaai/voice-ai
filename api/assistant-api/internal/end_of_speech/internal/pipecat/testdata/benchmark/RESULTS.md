# Pipecat Verification And Benchmark

Measured September 9, 2026. The baseline sections retain the original DFT
measurements. Production now uses a 400-point FFT; the follow-up quality
evaluation and waveform arithmetic correction are recorded in `QUALITY.md`.

## Production FFT Follow-Up

The final production binary was compared with a binary captured immediately
before FFT integration, including the waveform arithmetic fix in both.
Seven sequential pairs alternated which binary ran first. Each stage used
three warmups and ten measured calls on the spoken-question fixture, with
`GOMAXPROCS=1`, no race instrumentation, and the same model/runtime/audio.

| Stage | Corrected DFT Median | Production FFT Median |
|---|---:|---:|
| Feature extraction | 116.059 ms | 14.388 ms |
| Full audio-to-probability prediction | 242.749 ms | 132.670 ms |

All seven pairs favored FFT. The difference between prediction medians is about 110 ms
on this shared host; it is not a promised live-call improvement. Host contention
was substantial: one DFT round averaged 955.699 ms and one FFT round 185.355 ms.
All rounds are retained. Do not compare these timings directly with the older
Python baseline below or interpret them as p95/p99 service latency.

Feature extraction still reports zero Go heap allocations; full prediction
still reports 72 bytes and seven Go allocations. Native allocations are not
included. Production FFT passes the 1024-case quality comparison with zero
probability or decision differences from the controlled Pipecat reference.
Full EOS native integration and race suites pass.

The additional `CGO_ENABLED=0` full-package run fails four observability tests
that construct native detectors when model files exist. Those tests pass in
the native suite; this report does not claim the full no-cgo suite passes.

Artifacts are in `/private/tmp/rapida-pipecat-numerics/`:
`production-benchmark-summary.json` contains every round and both binary hashes,
`production-*-round-*.bench` contains raw output, and
`production_fft_benchmark.py` records the exact alternating commands.

## Conditions

- Apple M1 Pro, macOS 26.6.2, Go 1.25.13, Python 3.11.14, NumPy 1.26.4.
- Both sides use ONNX Runtime 1.16.0, CPU execution, one intra/inter-op thread,
  full graph optimization, and the identical Rapida model file.
- Pipecat source revision: `1f8a513dd79c31f54bbd5b197fa26aefe242deca`; inspected files clean.
- Three warmups, seven rounds of ten calls per stage; medians below are medians
  of round means. Python and Go timed processes ran sequentially without `-race`.
- Go uses production reusable feature scratch. Construction and file loading are
  excluded. This is a shared development machine, not an isolated load-test host.

## Baseline Timing

Milliseconds per call. Each input is padded/truncated to the model's eight-second window.

| Input | Pipecat Features | Rapida Features | Pipecat Inference | Rapida Inference | Pipecat Total | Rapida Total |
|---|---:|---:|---:|---:|---:|---:|
| Silence | 2.218 | 88.072 | 87.364 | 87.085 | 90.091 | 175.587 |
| Short signal | 2.226 | 88.583 | 87.263 | 87.162 | 89.631 | 178.169 |
| Signal with pause | 2.229 | 89.582 | 87.424 | 87.572 | 89.440 | 177.840 |
| Longer than eight seconds | 2.172 | 90.569 | 86.993 | 87.776 | 89.322 | 176.956 |
| Quiet DC signal | 2.215 | 88.377 | 87.062 | 87.048 | 89.466 | 176.079 |
| Constant signal | 2.186 | 88.158 | 86.971 | 86.949 | 89.400 | 175.073 |
| Spoken question | 2.182 | 88.399 | 87.032 | 87.272 | 89.132 | 177.880 |
| Second speech sample | 2.209 | 90.170 | 87.207 | 88.107 | 89.318 | 176.381 |
| Halfway-cut question | 2.194 | 88.870 | 86.946 | 87.180 | 89.370 | 175.659 |

Rapida features cost 39.7-41.7 times as much, and total prediction costs 1.95-2.00
times as much. Inference differs by approximately 0-1% in these round medians;
no general inference-speed advantage is claimed. Raw rounds retain outliers,
including one 316 ms Rapida short-input round. These are not p95/p99 latencies.

The Go timed feature path reports zero heap allocations. Native inference and
full prediction report 72 Go-heap bytes and seven allocations per call. Native
ONNX allocations are outside these counters. GC is not the identified bottleneck.

## Baseline Correctness

All nine native parity cases passed. Maximum feature absolute error was
`7.74860382e-06`, below the `2e-5` bound. Both inference on Python-produced
features and full Go prediction passed their probability tolerances (`1e-5`
and `1e-4`, respectively); all completion decisions agreed.

| Speech Sample | Pipecat Probability | Rapida Probability | Both Decisions |
|---|---:|---:|---|
| Complete question | 0.616009891 | 0.616009891 | Complete |
| Second speech sample | 0.978554368 | 0.978554368 | Complete |
| Halfway-cut question | 0.102818280 | 0.102818280 | Incomplete |

This verifies numerical behavior for this corpus, not semantic accuracy across
real conversations. Synthetic silence scoring as complete is a model-level
result; the EOS speech-activity gate must prevent silent input from making a turn.

Rapida and the bundled Pipecat model have identical graph protobufs, including
weights. File hashes differ because of IR version and opset-import metadata:

- Rapida: `c02d673e1d0b7c1acfb3323b5792bb7b37aa7746538145dbf1e9a01dcecd99b7`, IR 9.
- Pipecat bundle: `2bb026316b14a660486a75b1733cd3fbab8c2fd0314dc9af7be49f8cca967e4f`, IR 10.

The reference checkout declares ORT 1.24.3. A separate correctness probe
with that version and the same selected model produced exactly the same nine
probabilities as Python ORT 1.16.0. The timing table is the controlled 1.16.0
comparison, not a performance claim about Pipecat's default dependency set.

## Baseline Bottleneck

The feature-only CPU profile attributes 95.45% of total sampled CPU directly to
`whisperFeatures.extractInto`. Its direct DFT inner loop accounts for about 87%
of total samples, concentrated at `mel_spectrogram.go:66-69`. It iterates over
800 frames, 201 frequency bins, and 400 samples per prediction. Pipecat uses
`numpy.fft.rfft` at `_whisper_features.py:119`.

This identified the 400-point FFT optimization now integrated into production,
with the same window, padding, scaling, and feature layout. A 512-point
transform would change model inputs and was not used.

## Runtime Boundaries

Source comparison supports alignment for VAD restart, committed-text readiness,
incomplete predictions, received-silence completion, and stale-result rejection
under matched budgets. It also confirms these observable contract adaptations:

- Pipecat's explicit `finalized=True` STT acknowledgement can bypass transcript
  waiting. Rapida's ordinary committed STT packet has no equivalent flag.
- Pipecat can use `wait_for_transcript=False`; Rapida's EOS contract requires text.
- Pipecat derives transcript timing from STT latency metadata; Rapida uses its
  configured fallback budget, so equal configuration must not be assumed.
- Orphan VAD stops and generic inference errors have different recovery paths.
  Pipecat's special timeout exception also differs from Rapida's predictor errors.
- Audio buffer selection uses monotonic timestamps upstream and sample indices
  in Rapida. Unpaced replay is not proof of identical selected audio windows.

These benchmarks exclude that scheduling layer. Existing Rapida runtime tests
cover its mapped behavior; the full upstream event loop was not replayed here.
The user's original microphone conversation was not available as an audio fixture.

## Evidence

Reproduction commands are in `README.md`. Local artifacts are retained under
`/private/tmp/rapida-pipecat-results/ort116`: `reference.json`, `python.bench`,
`go.bench`, `summary.json`, `parity.log`, `features.cpu`, `go-linkage.txt`, and the test binary.
The cross-version probe is under `/private/tmp/rapida-pipecat-results/ort124`.
Go's runtime version is established by the binary linkage, not by the Python report.
