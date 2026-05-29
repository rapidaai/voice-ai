// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

type AssistantWebhookEvent string

const (
	//
	ConversationBegin     AssistantWebhookEvent = "conversation.begin"
	ConversationResume    AssistantWebhookEvent = "conversation.resume"
	ConversationCompleted AssistantWebhookEvent = "conversation.completed"
	// Triggered when a conversation ends successfully.

	ConversationFailed AssistantWebhookEvent = "conversation.failed"
	// Triggered when a conversation encounters an error.

	// SIP/call lifecycle events (pre/during conversation).
	CallCreated    AssistantWebhookEvent = "call.created"
	CallRinging    AssistantWebhookEvent = "call.ringing"
	CallAnswered   AssistantWebhookEvent = "call.answered"
	CallMediaStarted AssistantWebhookEvent = "call.media.started"
	CallNoAnswer   AssistantWebhookEvent = "call.no_answer"
	CallBusy       AssistantWebhookEvent = "call.busy"
	CallRejected   AssistantWebhookEvent = "call.rejected"
	CallCancelled  AssistantWebhookEvent = "call.cancelled"
	CallEnded      AssistantWebhookEvent = "call.ended"
	CallFailed     AssistantWebhookEvent = "call.failed"
	TransferRequested      AssistantWebhookEvent = "transfer.requested"
	TransferAttemptStarted AssistantWebhookEvent = "transfer.attempt.started"
	TransferTargetRinging  AssistantWebhookEvent = "transfer.target.ringing"
	TransferConnected      AssistantWebhookEvent = "transfer.connected"
	TransferAttemptEnded   AssistantWebhookEvent = "transfer.attempt.ended"
	TransferCancelled      AssistantWebhookEvent = "transfer.cancelled"
	TransferFailed         AssistantWebhookEvent = "transfer.failed"
)

func (r AssistantWebhookEvent) Get() string {
	return string(r)
}
