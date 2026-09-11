// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package lifecycle

import "time"

const (
	MessageStateUserIdle      MessageState = "user_idle"
	MessageStateUserListening MessageState = "user_listening"
	MessageStateUserSpeaking  MessageState = "user_speaking"
	MessageStateUserThinking  MessageState = "user_thinking"
	MessageStateUserFinished  MessageState = "user_finished"
	MessageStateUserPrompted  MessageState = "user_prompted"

	MessageStateAssistantGenerating MessageState = "assistant_generating"
	MessageStateAssistantGenerated  MessageState = "assistant_generated"
	MessageStateAssistantSpeaking   MessageState = "assistant_speaking"
	MessageStateAssistantFinished   MessageState = "assistant_finished"
	MessageStateAssistantIdle       MessageState = "assistant_idle"
	MessageStateAssistantPrompted   MessageState = "assistant_prompted"
)

const (
	StateNew SessionState = iota
	StateInitializing
	StateReady
	StateSwitching
	StateDisconnecting
	StateDisconnected
	StateFailed
)

const (
	EventConnectRequested SessionEvent = iota + 1
	EventInitializationCompleted
	EventInitializationFailed
	EventSwitchRequested
	EventSwitchCompleted
	EventSwitchFailedRecoverable
	EventSwitchFailedFatal
	EventDisconnectRequested
	EventDisconnectCompleted
)

// InterruptionDecisionWindow bounds the pause while speech is being confirmed.
const InterruptionDecisionWindow = 500 * time.Millisecond

// InterruptionEnabledByDefault preserves the staged interruption rollout.
const InterruptionEnabledByDefault = false
