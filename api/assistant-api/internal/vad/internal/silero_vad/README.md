# Silero VAD

Native voice activity detection using the Silero ONNX model with direct
ONNX Runtime C API integration (no third-party Go wrapper).

## Audio Configuration

| Parameter | Value |
|---|---|
| Input sample rate | 16 kHz |
| Input format | LINEAR16 (signed 16-bit PCM) |
| Input channels | mono |
| Chunk duration | 80 ms |
| Chunk size | 1280 samples (2560 bytes) |
| Silero window size | 512 samples (32 ms) |
| Windows per chunk | 2 |
| Context prepend | 64 samples from previous window |

The VAD accepts audio from any sample rate/format and resamples internally
to 16 kHz mono. When the input already matches (16 kHz LINEAR16 mono),
resampling is a no-op.

## Initialization Logic

1. Resolve the Silero ONNX model path:

   - If the environment variable `SILERO_MODEL_PATH` is set, use its value.
   - Otherwise, use the bundled model path:
     ```
     models/silero_vad_20251001.onnx
     ```

2. Load the ONNX model via the native C bridge into an ONNX Runtime session.

   - Single-threaded execution (intra/inter op threads = 1)
   - All graph optimizations enabled
   - If loading fails, return an error and abort initialization.

3. Resolve the Pipecat-style VAD parameters:

   - Read `microphone.vad.confidence` from configuration.
   - Read `microphone.vad.start_secs` from configuration.
   - Read `microphone.vad.stop_secs` from configuration.
   - If not provided, default to confidence `0.7`, start `0.2s`, stop `0.2s`.

4. Initialize the detector with:

   - The loaded ONNX session
   - The resolved confidence threshold
   - Speech-start debounce duration
   - Speech-stop debounce duration

5. Create an internal audio processing configuration:

   - Sample rate: 16 kHz
   - Channels: mono
   - Audio format: linear PCM converted to `float32`

6. Store the activity callback function used to report detected speech.

7. Return the initialized `SileroVAD` instance.

---

## Audio Processing Logic (`Process`)

1. Receive an incoming 80 ms chunk of audio data (1280 samples at 16 kHz).

2. Validate and normalize the audio format:

   - If the input audio is not 16 kHz mono, resample it accordingly.
   - If already 16 kHz LINEAR16 mono, pass through directly (no resampling).

3. Convert the normalized audio samples to `float32`.

4. Execute the Silero VAD detector on the converted audio buffer:

   - Slide a 512-sample window across the buffer (2 windows per 80 ms chunk).
   - For each window, prepend 64 samples of context from the previous window.
   - Run ONNX inference to get speech probability.
   - Track speech onset/offset using confidence plus start/stop debounce windows.

5. If the buffer is smaller than one window (512 samples / 32 ms):

   - Return immediately with no error and no segments.
   - This handles small network chunks at stream boundaries.

6. If **no speech segments** are detected:

   - Do not invoke the activity callback.
   - Return `nil`.

7. If **one or more speech segments** are detected:

   - Identify the earliest segment start time.
   - Identify the latest segment end time.
   - Merge all detected segments into a single speech activity window.

8. Invoke the activity callback with an `InterruptionPacket` and an
   `ObservabilityEventRecordPacket` containing the merged start/end times.

9. Return `nil` unless an error occurred during processing.

---

## Identifier Logic (`Name`)

1. Return the constant VAD identifier:
   ```
   silero_vad
   ```

---

## Cleanup Logic (`Close`)

1. Acquire the write lock.
2. Mark as terminated (idempotent — subsequent calls are no-ops).
3. Destroy the ONNX detector: release session, environment, memory info,
   and all pre-allocated C strings.
4. Return `nil`.

Context cancellation triggers `Close` automatically via the lifecycle manager.

---

## Reset Logic (`Reset`)

1. Zero the ONNX hidden state (`[2][1][128]` float32).
2. Zero the context buffer (64 float32).
3. Reset sample counter, triggered flag, and silence timer.
4. The ONNX session remains loaded — no re-initialization needed.

Use `Reset` to reuse the detector for a new audio stream without the
~120 ms initialization cost.

---

## Runtime Constraints and Behavior

- Audio must be processed as a continuous stream of 80 ms chunks.
- The detector is stateful and maintains ONNX hidden state across `Process` calls.
- A single `SileroVAD` instance must be reused for the lifetime of a voice stream.
- The `Detector` is NOT safe for concurrent use — `SileroVAD` wraps it with a mutex.
- Speech that starts in one chunk and ends in a later chunk is handled correctly:
  the onset is reported immediately, and the offset resets state without error.
- Detection accuracy depends on:
  - Microphone quality
  - Environmental noise
  - Correct audio resampling and conversion

## Performance (Apple M1 Pro)

| Metric | Value |
|---|---|
| Processing cost (80 ms chunk) | ~400 µs |
| Processing cost (500 ms chunk) | ~1.99 ms |
| Throughput | 243× real-time |
| Initialization | ~121 ms (one-time) |
| Memory per 80 ms chunk | ~28 KB, 44 allocs |
