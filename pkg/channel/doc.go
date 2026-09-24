// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package channel provides concurrent generic FIFO channels with explicit
// capacity and overflow policies.
//
// A Channel does not expose its internal storage as a native Go channel.
// Producers must call Send and consumers must call Receive. This keeps buffer
// mutations atomic and prevents callers from bypassing the configured policy.
//
// FixedCapacity creates a constant-size buffer. GrowingCapacity provides the
// production-safe equivalent of an unbounded channel by growing only to an
// explicit maximum. Once the maximum is reached, the configured overflow
// policy blocks, rejects the newest value, or replaces the oldest value.
//
// Channel does not create a background goroutine. Blocking operations wait on
// condition variables and honor context cancellation. Close wakes blocked
// operations, rejects future sends, and permits already accepted values to be
// drained. Ready and TryReceive support callers that must select across several
// independently owned channels.
package channel
