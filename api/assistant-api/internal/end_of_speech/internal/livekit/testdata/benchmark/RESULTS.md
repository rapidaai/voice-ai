# LiveKit EOS Alignment Results

## Scope

Comparison and verification: September 9-10, 2026. Reference: the supplied LiveKit checkout at
`ab6e8a1caf0b87ae019173aae28f26babb510776`, not an assertion about upstream main.

No public API, packet schema, provider-option, model-selection, or dispatch
changes. Existing threshold, endpoint delays, history option, and model paths
remain authoritative. This aligns fixed endpointing, not LiveKit's optional
dynamic endpointing or language-based calibration policy.

Runtime corrections:

- Prediction runs in the EOS worker, with LiveKit's three-second inference
  deadline. New final text, resumed VAD speech, and close cancel obsolete work.
- Each native call owns its ONNX run options. Cancellation terminates the run;
  callback cleanup finishes before releasing options or destroying the session.
- The worker delivers runtime notifications in FIFO order. State publication
  and command admission share one lock; callbacks execute outside that lock.
- Callback re-entry can enqueue a finite burst without waiting for its own
  worker. Admitted interim notifications precede timer-based final delivery.
- Successful external close joins the worker before destroying the detector and
  emitting closed telemetry. A closing caller's deadline may expire while
  cleanup continues. Callback sinks must make progress; callbacks must return
  before an external owner waits for shutdown.
- English uses the pinned English tokenizer's special role tokens. Multilingual
  retains its distinct role/newline template and text preparation.

Notification scheduling is now worker-owned: `Execute` need not wait for an
interim callback to return. The FIFO preserves packets and finite reentrant
bursts, but cannot bound memory if a sink stalls indefinitely or generates work
faster than it consumes it. No packet-dropping policy was introduced.

## Model Pairing

The supplied English and multilingual model digests differ from the revisions
declared by this reference checkout. The supplied tokenizer matches the pinned
English tokenizer, but not the pinned multilingual tokenizer. Exact hashes and
publisher commits are recorded in `metadata.json` and each reference artifact.

Measurements use identical selected local assets in both implementations. This
tests code parity with those assets. It does not certify a stock LiveKit model
deployment. Model provisioning or migration was not changed.

## Reproduced English Input Bug

Before correction, all 29 diagnostic prompts differed from LiveKit's native
template. The largest probability discrepancy was 0.4943187386. Tokenization
and ONNX inference agreed when given the same prompt. At the unchanged 0.0289
threshold, this small baseline corpus happened to have no decision flips.

After correction, the same 29 cases have identical prompts, token IDs, model
probabilities, and decisions. Baseline artifacts are under
`/private/tmp/rapida-livekit-before-template-en/`. Its repetition case predates
the final corpus update to the user's exact transcript, so corpus hashes must
not be treated as interchangeable.

## Latency Baseline

Seven alternating Python/Go rounds, ten measured calls per stage and three
warmups. Apple M1 Pro, CPU ONNX Runtime 1.16.0, one intra-op/inter-op thread.
These are medians of round means, not request percentiles or full-call latency.

| English case | Python model path | Go deadline-bearing model path |
| --- | ---: | ---: |
| Short complete response | 8.508 ms | 8.750 ms |
| User's repeated incomplete sentence | 13.954 ms | 13.850 ms |
| Long history, left-truncated to 128 tokens | 107.032 ms | 441.488 ms |

The long case's tokenizer costs 337.152 ms in Go versus 0.438 ms in Python;
raw inference costs 106.589 ms versus 105.636 ms. Full merge-table scanning,
not ONNX execution, is the identified implementation bottleneck.

Baseline artifacts: `/private/tmp/rapida-livekit-en-comparison/`.

## Ranked-Merge Results

The rank map now selects the leftmost lowest-rank adjacent pair and recomputes
neighbors after each merge. It replaces a complete merge-table scan for every
text piece. The preceding CPU profile attributes 52.65% cumulative sampled CPU
to `applyMerge`; the profiled long-case call allocated about 259 MB and 5.36
million Go objects before the fix.

| Model / case | Matched Python model path | Optimized Go deadline-bearing path |
| --- | ---: | ---: |
| English / short complete | 6.759 ms | 6.353 ms |
| English / repeated incomplete sentence | 10.913 ms | 10.571 ms |
| English / long history | 83.868 ms | 83.749 ms |
| Multilingual / short complete | 35.290 ms | 34.740 ms |
| Multilingual / repeated incomplete sentence | 75.189 ms | 74.862 ms |
| Multilingual / long history | 594.023 ms | 599.107 ms |

The optimized long-case tokenizer takes 0.235 ms in both models, versus Python
0.345 ms (English) and 0.414 ms (multilingual). All 58 diagnostic model/case
combinations retain exact prompts, IDs, probabilities, and decisions.

English uses seven rounds of ten measured calls; the optimized multilingual run
uses seven rounds of three calls. Both use three warmups. Baseline multilingual
used ten measured calls per round. Raw round samples are retained; these are
descriptive comparisons, not significance claims or request percentiles.

Native reference execution also got faster between baseline and final runs on
this shared host. Do not attribute the entire wall-time reduction to the patch.
The matched English long-case implementation gap fell from about 334 ms to
approximately zero. Native thread counts and optimization settings did not
change. Expanding each session's native thread budget requires a separate
concurrent-session capacity experiment.

Final artifacts: `/private/tmp/rapida-livekit-en-optimized/` and
`/private/tmp/rapida-livekit-multilingual-optimized/`. The saved binaries and
per-run hashes distinguish the pre-fix and optimized implementations.

## Quality Limits

The 29-case corpus is hand-labeled diagnostic coverage, not an independent
accuracy dataset. English agrees with all 15 English labels; out-of-language
stress cases must not be conflated with English-model accuracy. Parity alone
does not establish real-world turn quality or improvement in label accuracy.

Multilingual agrees with only 15/26 diagnostic labels, including one false
complete and ten missed complete predictions. The supplied asset pairing and
unchanged threshold need validation before this can be considered acceptable
quality. Matching Python does not resolve that concern.

Both models classify the exact repeated sentence supplied by the user as
incomplete at threshold 0.0289: English 0.0002955681, multilingual 0.0105781164.
This is text-model behavior, not a replay of the earlier Pipecat audio decision.

The cached 256-row Pipecat evaluation sample cannot supply a text evaluation:
every cached `spoken_text` value is null. No transcript was invented from those
audio labels. The user's original audio, STT timing, and live SIP path have not
been replayed in this model benchmark.

## Published English Sample

Text-only validation rows 0 through 99 from `livekit/eot-bench-data`, revision
`ca9d98a9686b920a2d8c9eb984224ba9be74e4dd` (July 22, 2026), yield 213 nonempty
causal prefixes: 124 hold and 89 end-of-turn. Fifteen no-text cases are excluded
explicitly: four hold and eleven end-of-turn. No audio was downloaded.

The transformation follows `eot_harness/io.py` at benchmark revision
`6594d8b3b8af385b15f116dde310ce45af92d646` (August 28, 2026). It preserves original
span labels and uses 500 ms transcript lag. One score is taken 200 ms into each
qualifying silence span. This is a selected single-score subset, not the
official harness's full 100 ms decision grid or an official leaderboard result.

At the unchanged threshold, Rapida and Python agree on all 213 prompts, token
sequences, probabilities, and decisions. Each scores:

| Outcome | Count |
| --- | ---: |
| Correct hold | 98 / 124 |
| False complete | 26 / 124 (20.97%) |
| Correct complete | 18 / 89 |
| Missed complete at this snapshot | 71 / 89 |
| Overall snapshot label agreement | 116 / 213 (54.46%) |

These are snapshot classification results. Missed completion at 200 ms can mean
waiting for text that arrives later; false-complete predictions are not measured
live-call cutoffs. Training overlap, conversation independence, and production
generalization are unverified. Multiple pauses share turns, and positional
sampling is not a random population sample. This is not a passing quality gate.

The old English formatter scored 118/213 (55.40%) on the same prefixes, with
21 false-complete predictions and 74 missed-complete predictions. Aligning the
template changes 16 decisions: the new/reference result has five additional
false-complete predictions and three fewer missed-complete predictions. This
does not demonstrate improved classification quality. It demonstrates exact
reference parity, with a measured adverse quality tradeoff on this snapshot
subset. Publication for review does not satisfy the quality gate for merge.

Artifacts: `/private/tmp/rapida-livekit-eot-rows.json`,
`/private/tmp/rapida-livekit-eot-corpus.json`, and
`/private/tmp/rapida-livekit-eot-en/`. `dataset.py` reproduces the transformation
and rejects partial, truncated, or wrong-revision viewer rows. The HF dataset
card declares CC-BY-4.0; the benchmark README instead describes Apache-2.0.
That licensing discrepancy is recorded rather than silently resolved. Neither
raw rows nor derived corpus/reference outputs containing public text are
redistributed in this repository; they remain in the listed temporary paths.

## Verification

- Full EOS/options native race and integration suite passed after runtime and
  English-template corrections.
- Eleven Python harness/export tests passed, including actual local reference
  source, native tokenizer execution, and causal-prefix label checks.
- Native cancellation tests exercise cancellation before/during a run,
  deadline expiry, and detector reuse after cancellation.
- FIFO tests cover a 64-message reentrant burst, interim delivery blocked past
  the endpoint deadline, stale predictions, and close during a blocked callback.
- `just agent-finalize` passed for all changed paths: 24 provider UI suites,
  468 UI tests, LiveKit/Pipecat/options Go packages, and required test mapping.
  The first attempt was blocked by sandboxed Watchman access; the authorized
  rerun passed. `git diff --check` passed.

Independent production-code review approved the final tokenizer, runtime,
native cancellation, and template changes with no unresolved critical or major
code findings. This is code review approval, not acceptance of the measured
quality tradeoff. Commit and push were authorized for review; merge and model
migration remain outside that authorization.
