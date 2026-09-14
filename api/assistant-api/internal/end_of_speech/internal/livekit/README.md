# Text Turn End Of Speech

This provider predicts turn completion from committed STT text and conversation
history. VAD supplies speech boundaries; audio supplies timing, not model input.
It does not own playback or interruption decisions. The message lifecycle owns
the turn after receiving `EndOfSpeechPacket`.

## Parameters

Pass parameters through `WithOptions`. Missing or nil values use defaults.
Parsing uses the existing `utils.Option` getters, without another conversion layer.
Malformed, negative, non-finite, or out-of-range numeric values fail construction
with `errLivekitInvalidOption` rather than silently selecting defaults.

| Key | Default | Unit and contract |
| --- | --- | --- |
| `microphone.eos.threshold` | `0.0289` | Finite probability in `[0, 1]`, parsed with `GetFloat64`. Probability `>=` threshold selects the quick timeout; lower values select the extended timeout. |
| `microphone.eos.quick_timeout` | `250` | Non-negative integer milliseconds, parsed with `GetUint64`. Also used after prediction failure. |
| `microphone.eos.extended_timeout` | `3000` | Non-negative integer milliseconds, parsed with `GetUint64`. |
| `microphone.eos.max_history_turns` | `6` | Non-negative integer, parsed with `GetUint64` and bounded by the platform's maximum `int`. Zero disables history slicing, not history collection. |
| `microphone.eos.model` | `en` | Parsed with `GetString`. Only exact `multilingual` selects multilingual behavior; all other labels, including custom labels, retain English compatibility. Empty uses the default. |
| `microphone.eos.livekit.model_path` | empty | ONNX model path, parsed with `GetString`; empty uses the model-specific environment variable, then the source-relative default below. |
| `microphone.eos.livekit.tokenizer_path` | empty | Tokenizer path, parsed with `GetString`; empty uses `LIVEKIT_TURN_TOKENIZER_PATH`, then source-relative `models/tokenizer.json`. |

Timeouts are bounded by `math.MaxInt64 / int64(time.Millisecond)`, or
`9223372036854` whole milliseconds. Zero skips the selected wait, not the committed
text requirement. Explicit nonempty paths take precedence over environment values.
English uses `LIVEKIT_TURN_MODEL_PATH`, then `models/model_q8.onnx`; multilingual
uses `LIVEKIT_TURN_MULTI_MODEL_PATH`, then `models/model_q8_multilingual.onnx`.
Relative defaults are resolved against the provider source directory.

## Packet Flow

```text
EOS audio       -> count mono PCM16 samples at 16 kHz for VAD stop timing
VAD start       -> cancel prediction; invalidate pending completion; retain STT text
VAD stop        -> record stop time; queue prediction when committed text exists
STT interim     -> publish preview; retain previously committed text
STT final       -> append committed text; queue prediction unless VAD is speaking
prediction      -> select quick or extended deadline for the current revision
timer           -> retire revision; record user history; emit committed text and telemetry
user text       -> queue preview and immediate completion without prediction
LLM done        -> append nonempty assistant text to history
```

`Interim=false` commits a transcript segment, not an entire utterance. Interim-only
input cannot finalize. Deadlines use the VAD stop time, corrected for received
audio when a valid stop offset exists; without VAD, final STT establishes the base
time. Prediction time counts toward the selected wait. An eligible, context-matched
`EndOfSpeechInterruptionPacket` arms the extended wait for committed text while idle.

## History And Templates

`chat_template.go` retains nonempty user/assistant messages and appends current user
text. A positive history limit selects the last entries before adjacent same-role
messages are merged, so the limit includes current text and is not a turn-pair count.
Zero leaves this list unsliced; native inference still keeps only the last 128 tokens.

English preserves content and uses `<|im_start|><|user|>` or the assistant role token.
Multilingual lowercases text, applies NFKC, removes punctuation except apostrophes
and hyphens, and collapses whitespace. It uses role names followed by newlines.
Both templates leave the final message open without `<|im_end|>`.

The tokenizer supports English whitespace preparation with `Digits -> ByteLevel`
and multilingual NFC composition with `Split -> ByteLevel`. Split accepts only the pinned regex,
`Isolated` behavior, and `invert=false`; unsupported configurations fail during
loading. Byte-level prefix and regex settings apply after that first stage.
Model and tokenizer paths remain explicit: supporting the multilingual tokenizer
does not automatically select or replace deployed assets.

## Ownership

`Execute` updates state synchronously and queues ordered commands. One worker owns
prediction, timer handling, and delivery. Each prediction derives a three-second
timeout from its command context. New committed text, VAD boundaries, user text,
and shutdown cancel obsolete inference; revision checks reject stale results.
Completed user history is recorded under the state lock; callbacks run after unlock.

`WithOnPacket` is required. Callbacks should enqueue promptly; failures are not
retried or rolled back. Callers must not mutate the supplied options map after
construction; `Options` returns that configuration without inserting defaults.

`Close` is idempotent: it rejects new work, clears queued commands, cancels inference,
and stops the worker. Cleanup waits for worker exit before destroying native
resources and emitting final telemetry. A canceled closing context returns its
error, but cleanup continues; later calls can wait for the same cleanup completion.

## Errors

| Condition | Reporting and behavior |
| --- | --- |
| Missing packet callback | `New` returns `errLivekitOnPacketRequired`. |
| Invalid supplied parameter | `New` returns `errLivekitInvalidOption` with the key and reason; parse errors retain their cause. |
| Model or tokenizer initialization failure | `New` returns `errLivekitInitTurnDetector` wrapping the underlying error. Constructor failures are returned only, without duplicate error logs. |
| Prediction failure | `predictEOU` returns `(float64, error)`, never a `-1` probability marker. The worker logs genuine failures once as `turn prediction failed`, with an `error` attribute, and selects the quick timeout. |
| Prediction cancellation or deadline | No failure log. A still-current failure uses the quick timeout; canceled command contexts and stale revisions cannot finalize. |

Use `errors.Is` for sentinels in `error.go` and `errors.As` for wrapped parse errors.
Asynchronous prediction errors cannot be returned by the completed `Execute` call.

## Reference Differences

The configured threshold is fixed; language metadata does not select thresholds.
Model/tokenizer assets are selected locally; this cleanup does not change them
or claim numerical parity against other model revisions.
