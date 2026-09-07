# Policy Channel

`pkg/channel` provides a generic concurrent FIFO whose capacity and overflow
behavior are explicit at construction time. It is intended for runtime queues
where a plain buffered Go channel does not express the required overload
policy.

```go
audio, err := channel.New[AudioFrame](channel.Config{
	CapacityPolicy: channel.FixedCapacity(1000),
	OverflowPolicy: channel.ReplaceOldestWhenFull,
})
```

## Capacity policies

- `FixedCapacity(n)` allocates a bounded queue that never grows.
- `GrowingCapacity(initial, maximum)` grows geometrically up to an explicit
  maximum. The package intentionally does not provide an unlimited queue
  because sustained producer pressure must remain memory-bounded.

## Overflow policies

- `BlockWhenFull` waits for capacity and honors context cancellation.
- `RejectNewestWhenFull` preserves queued values and rejects the incoming one.
- `ReplaceOldestWhenFull` atomically removes the oldest queued value and
  accepts the incoming one. This is appropriate for realtime media where fresh
  audio is more useful than stale audio.

`Send` returns a `SendResult` so the owner can record replacement or rejection
without duplicating queue logic.

## Receiving and lifecycle

- `Receive` blocks until a value, cancellation, or closure.
- `TryReceive` performs a non-blocking receive.
- `Ready` supports selection alongside other event sources. A notification is
  level-triggered, so callers must invoke `TryReceive` after waking.
- `Drain` removes every queued value.
- `RemoveIf` removes selected values while preserving the order of retained
  values.
- `Close` rejects new sends, wakes blocked operations, and allows accepted
  values to drain before `ErrClosed` is returned.

The zero value is not usable. Construct channels with `New`. A channel owns its
ring buffer and does not start background goroutines.

## Attribution

The policy vocabulary and bounded-buffer concepts were informed by
`github.com/eapache/channels`. This implementation uses a different generic,
context-aware, mutex-protected ring buffer designed for this repository. The
upstream MIT license is preserved in `LICENSE.eapache-channels`.
