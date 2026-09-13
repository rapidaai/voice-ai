package lifecycle

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"google.golang.org/protobuf/proto"
)

// MessageOption configures a message lifecycle before it handles events.
type MessageOption func(*messageLifecycle)

// WithContextID sets the initial message identity. An empty ID generates a new one.
func WithContextID(contextID string) MessageOption {
	return func(message *messageLifecycle) { message.contextID = contextID }
}

// WithMode sets the initial message mode. Runtime mode switches use SetMode.
func WithMode(mode type_enums.MessageMode) MessageOption {
	return func(message *messageLifecycle) { message.mode = mode }
}

// WithInterruption enables or disables speech-confirmed interruption.
func WithInterruption(enabled bool) MessageOption {
	return func(message *messageLifecycle) { message.interruptionEnabled = enabled }
}

// WithOnPacket receives lifecycle metrics and timeout notifications, including timer callbacks.
func WithOnPacket(onPacket func(...internal_type.Packet) error) MessageOption {
	return func(message *messageLifecycle) { message.onPacket = onPacket }
}

// WithSend supplies the transport for assistant output and playback controls.
func WithSend(send func(proto.Message) error) MessageOption {
	return func(message *messageLifecycle) { message.sendOutput = send }
}

// WithDispatch delivers turn updates and admitted input synchronously.
// Input callbacks must continue downstream without repeating admission.
func WithDispatch(dispatch func(context.Context, internal_type.Packet)) MessageOption {
	return func(message *messageLifecycle) { message.dispatchPacket = dispatch }
}

// WithInterruptionExpiry routes expired speech-confirmation decisions.
func WithInterruptionExpiry(onExpired func(internal_type.InterruptionDecisionExpiredPacket)) MessageOption {
	return func(message *messageLifecycle) { message.onInterruptionExpired = onExpired }
}

// WithBehavior supplies deployment behavior when Initialize is called.
func WithBehavior(load func() (*internal_assistant_entity.AssistantDeploymentBehavior, error)) MessageOption {
	return func(message *messageLifecycle) { message.loadBehavior = load }
}
