// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_runtime

import (
	"fmt"

	"github.com/rapidaai/pkg/commons"
)

func newCallLifecycle(callID string, initial CallState, logger commons.Logger) *CallLifecycle {
	return &CallLifecycle{
		callID: callID,
		state:  initial,
		logger: logger,
	}
}

func (c *CallLifecycle) State() CallState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *CallLifecycle) Transition(next CallState, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == next {
		return nil
	}
	allowed := false
	switch c.state {
	case CallStateInitializing:
		allowed = next == CallStateRinging || next == CallStateConnected || next == CallStateEnding || next == CallStateFailed || next == CallStateCancelled
	case CallStateRinging:
		allowed = next == CallStateConnected || next == CallStateEnding || next == CallStateFailed || next == CallStateCancelled
	case CallStateConnected:
		allowed = next == CallStateOnHold || next == CallStateTransferring || next == CallStateBridgeConnected || next == CallStateEnding || next == CallStateFailed
	case CallStateOnHold:
		allowed = next == CallStateConnected || next == CallStateEnding || next == CallStateFailed
	case CallStateTransferring:
		allowed = next == CallStateConnected || next == CallStateBridgeConnected || next == CallStateEnding || next == CallStateFailed
	case CallStateBridgeConnected:
		allowed = next == CallStateConnected || next == CallStateEnding || next == CallStateFailed
	case CallStateEnding:
		allowed = next == CallStateEnded || next == CallStateFailed
	case CallStateFailed:
		allowed = next == CallStateEnding || next == CallStateEnded
	case CallStateCancelled:
		allowed = next == CallStateEnded
	}
	if !allowed {
		return fmt.Errorf("invalid lifecycle transition: %s -> %s", c.state, next)
	}

	prev := c.state
	c.state = next
	if c.logger != nil {
		c.logger.Infow("Call lifecycle transition",
			"call_id", c.callID,
			"from", prev,
			"to", next,
			"from_phase", lifecyclePhase(prev),
			"to_phase", lifecyclePhase(next),
			"reason", reason)
	}
	return nil
}

func lifecyclePhase(state CallState) string {
	switch state {
	case CallStateInitializing:
		return "inviting"
	case CallStateRinging:
		return "ringing"
	case CallStateTransferring:
		return "transferring"
	case CallStateBridgeConnected:
		return "bridged"
	case CallStateEnding:
		return "ending"
	case CallStateEnded, CallStateCancelled:
		return "terminal"
	default:
		return string(state)
	}
}
