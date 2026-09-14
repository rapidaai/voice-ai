# Smart Turn End Of Speech

This provider combines audio-model predictions with committed STT text. It does
not detect speech activity itself and does not own playback or interruption
decisions. VAD supplies speech boundaries; the message lifecycle owns the turn
after receiving `EndOfSpeechPacket`.

## Parameters

Pass provider parameters through `WithOptions`. Missing or nil values use the
defaults below. Numeric parsing uses `utils.Option` getters. Supplied
malformed, negative, non-finite, or out-of-range values fail construction with
`errPipecatInvalidOption`; they do not silently select defaults.

| Key | Default | Unit and contract |
| --- | --- | --- |
| `microphone.eos.threshold` | `0.85` | Probability in `[0, 1]`. A prediction must be strictly greater than the threshold to mark speech complete. |
| `microphone.eos.fallback_timeout` | `1000` | Milliseconds of transcript safety wait, measured from speech stop when its audio timestamp is available. It does not itself override an incomplete model verdict. |
| `microphone.eos.extended_timeout` | `4000` | Milliseconds of received audio after VAD stop before the silence limit marks speech complete. This is not a wall-clock timer. |
| `microphone.eos.pipecat.model_path` | empty | String path to an ONNX model. An empty value uses `PIPECAT_TURN_MODEL_PATH`, then the bundled `models/smart-turn-v3.2-cpu.onnx` path relative to the provider source. |

Timeouts use `GetUint64`: non-negative whole milliseconds, from zero through
`9223372036854` milliseconds, the largest whole-millisecond value representable
by `time.Duration`. Zero skips that particular wait; it does not bypass the
other completion conditions. UI slider ranges are narrower tuning suggestions,
not additional provider parameters.

Fixed runtime settings:

- Input audio: mono PCM16 little-endian at 16 kHz.
- Audio window: at most eight seconds, including up to 500 ms before speech start
  when that audio remains available.
- Inactivity recovery: five seconds after VAD stop, renewed by text-bearing STT
  updates while speech remains stopped. Recovery requires committed text.
- Prediction budget: the later of the initial transcript deadline and the
  five-second recovery deadline. Later STT activity does not extend that prediction.

## Packet Flow

```text
audio           -> bounded speech buffer
VAD start       -> cancel old prediction and invalidate pending completion
VAD stop        -> snapshot audio -> queue prediction -> return
STT interim     -> publish interim text; do not commit it
STT final       -> append committed text; update transcript waiting
prediction      -> update current speech verdict, if its speech revision still matches
timer           -> release committed text when completion conditions are satisfied
```

`Interim=false` commits a transcript segment, not an entire utterance. The provider
therefore retains a transcript safety wait even after a positive model result.
Only committed text is released. An incomplete model result can be superseded by
the received-silence limit or the inactivity-recovery deadline. Resumed speech
invalidates the old decision. Text-only user input bypasses audio prediction.

With no VAD boundary, committed STT text uses an inactivity-based transcript wait.
Interim-only input does not produce a final EOS packet; unclear-input handling
remains with the message lifecycle.

## Ownership

`Execute` applies packet state synchronously. One prediction worker runs native
inference with at most one replaceable pending request. A separate worker owns
the completion timer. Speech revisions reject stale predictions; transcript
revisions reject stale timer commands. No prediction cache is needed.

`WithOnPacket` is required. The callback must be safe for concurrent calls and
should enqueue packets promptly. Callback failures are not retried and do not
roll back completion. Callers must not mutate the supplied options map after
construction; `Options` exposes the supplied configuration rather than a copy
with defaults inserted. The threshold uses `GetFloat64` and the model path uses
`GetString`; this provider does not define another conversion layer.

`Close` signals the timer worker, cancels inference, waits for the prediction
worker to exit, and then destroys native resources. It is idempotent. Its context
is used for final telemetry, not as a deadline for waiting on native cleanup.

## Errors

| Condition | Reporting and behavior |
| --- | --- |
| Missing packet callback | `New` returns `errPipecatOnPacketRequired`. |
| Invalid supplied parameter | `New` returns `errPipecatInvalidOption` with the parameter key and validation reason; parse errors retain their cause. |
| Model initialization failure | `New` returns `errPipecatInitDetector` wrapping the native operation error. Initialization errors are not also emitted as log packets. |
| Empty audio snapshot | Prediction is skipped and speech remains incomplete. |
| Native inference failure | The worker emits one error log with message `turn prediction failed`, operation `predict_end_of_turn`, and the error detail in attributes. Existing silence/recovery paths remain active. |
| Prediction cancellation or deadline | No error-level log. A stale speech revision cannot apply its result; a still-current failed prediction remains incomplete. |

Use `errors.Is` for the provider and native-operation sentinels declared in
`error.go`. Wrapped parse errors can also be inspected with `errors.As`.
Asynchronous inference errors cannot be returned by the already-completed
`Execute` call.

## Verification

```sh
go test -race -count=1 -timeout=180s ./api/assistant-api/internal/end_of_speech/...
```

The provider tests cover option validation, error propagation, duplicate stops,
resumed speech, delayed transcripts, stale predictions, inactivity recovery, and
shutdown. Native tests require the repository's ONNX Runtime and model setup.
