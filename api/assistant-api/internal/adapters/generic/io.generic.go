// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"
	"fmt"
	"time"

	internal_adapter_request_customizers "github.com/rapidaai/api/assistant-api/internal/adapters/customizers"
	internal_end_of_speech "github.com/rapidaai/api/assistant-api/internal/end_of_speech"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// START ListenCallback Interface
// =====================================

func (lio *GenericRequestor) OnRecieveTranscript(ctx context.Context,
	transcript string,
	confidence float64,
	language string,
	isCompleted bool) (*types.Message, error) {
	lio.OnInterrupt(ctx, "word")
	if isCompleted {
		msgi := lio.messaging.Create(
			type_enums.UserActor,
			transcript)
		lio.Notify(ctx,
			&protos.AssistantConversationUserMessage{
				Id: msgi.GetId(),
				Message: &protos.AssistantConversationUserMessage_Text{
					Text: &protos.AssistantConversationMessageTextContent{
						Content: msgi.String(),
					},
				},
				Completed: false,
				Time:      timestamppb.New(time.Now()),
			})
		return msgi, nil
	}
	return types.NewMessage("user", &types.Content{
		ContentType:   commons.TEXT_CONTENT.String(),
		ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
		Content:       []byte(transcript),
	}), nil
}

func (io *GenericRequestor) OnSilenceBreak(
	ctx context.Context,
) error {
	start := time.Now()
	defer io.logger.Benchmark("io.OnSilenceBreakActivity", time.Since(start))

	msg, err := io.messaging.GetMessage(type_enums.UserActor)
	if err != nil {
		io.logger.Tracef(ctx, "illegal message state with error %v", err)
		return nil
	}
	io.messaging.Transition(internal_adapter_request_customizers.UserCompleted)
	if err := io.Notify(ctx,
		&protos.AssistantConversationUserMessage{
			Id: msg.GetId(),
			Message: &protos.AssistantConversationUserMessage_Text{
				Text: &protos.AssistantConversationMessageTextContent{
					Content: msg.String(),
				},
			},
			Completed: true,
			Time:      timestamppb.New(time.Now()),
		}); err != nil {
		io.logger.Tracef(ctx, "might be returing processing the duplicate message so cut it out.")
		return nil
	}
	io.messaging.Transition(internal_adapter_request_customizers.LLMGenerating)
	return io.Execute(
		ctx,
		msg.GetId(),
		msg,
	)
}

func (io *GenericRequestor) OnInterrupt(ctx context.Context, source string) error {
	switch source {
	case "word":
		if err := io.messaging.Transition(internal_adapter_request_customizers.Interrupted); err != nil {
			return nil
		}
		if io.messaging.GetMode().Audio() {
			io.recorder.Interrupt()
		}
		io.Notify(ctx,
			&protos.AssistantConversationInterruption{
				Type: protos.AssistantConversationInterruption_INTERRUPTION_TYPE_WORD,
				Time: timestamppb.Now(),
			})
	default:
		if err := io.messaging.Transition(internal_adapter_request_customizers.Interrupt); err != nil {
			return nil
		}
		if io.messaging.GetMode().Audio() {
			io.recorder.Interrupt()
		}
		io.Notify(ctx, &protos.AssistantConversationInterruption{
			Type: protos.AssistantConversationInterruption_INTERRUPTION_TYPE_VAD,
			Time: timestamppb.Now(),
		})
	}
	return nil

}

// Start Input
// =====================================
func (io *GenericRequestor) Input(message *protos.AssistantConversationUserMessage) error {
	switch msg := message.GetMessage().(type) {
	case *protos.AssistantConversationUserMessage_Audio:
		return io.InputAudio(io.Context(), msg.Audio.GetContent())
	case *protos.AssistantConversationUserMessage_Text:
		return io.InputText(io.Context(), msg.Text.GetContent())
	default:
		return fmt.Errorf("illegal input from the user %+v", msg)
	}

}

func (io *GenericRequestor) InputAudio(ctx context.Context, in []byte) error {
	io.messaging.SwitchMode(type_enums.AudioMode)
	v, _ := io.ListenAudio(ctx, in)
	utils.Go(context.Background(), func() {
		io.recorder.User(v)
	})
	return nil
}

func (io *GenericRequestor) InputText(ctx context.Context, msg string) error {

	// changing to text mode
	io.messaging.SwitchMode(type_enums.TextMode)

	// mark it interrupted
	io.messaging.Transition(internal_adapter_request_customizers.Interrupted)
	//
	interim := io.messaging.Create(type_enums.UserActor, msg)
	if err := io.Notify(ctx, &protos.AssistantConversationUserMessage{
		Id:        interim.GetId(),
		Completed: false,
		Message: &protos.AssistantConversationUserMessage_Text{
			Text: &protos.AssistantConversationMessageTextContent{
				Content: interim.String(),
			},
		},
		Time: timestamppb.Now(),
	}); err != nil {
		io.logger.Tracef(ctx, "error while notifying the text input from user: %w", err)
	}
	return io.ListenText(
		ctx,
		&internal_end_of_speech.UserEndOfSpeechInput{
			Message: interim.String(),
			Time:    time.Now(),
		},
	)
}

// END Input
// =====================================

func (io *GenericRequestor) OutputAudio(
	contextId string,
	v []byte, completed bool) error {
	inputMessage, err := io.messaging.GetMessage(type_enums.UserActor)
	if err != nil {
		return err
	}
	// //
	if contextId != inputMessage.GetId() {
		// io.logger.Warnf("testing: context id mismatched %+v current %v", contextId, inputMessage.GetId())
		return nil
	}

	if err := io.messaging.Transition(internal_adapter_request_customizers.AgentSpeaking); err != nil {
		// io.logger.Warnf("testing: illegal transition to speaking")
		return nil
	}

	if err := io.
		Notify(
			io.Context(),
			&protos.AssistantConversationAssistantMessage{
				Time:      timestamppb.Now(),
				Id:        contextId,
				Completed: completed,
				Message: &protos.AssistantConversationAssistantMessage_Audio{
					Audio: &protos.AssistantConversationMessageAudioContent{
						Content: v,
					},
				},
			},
		); err != nil {
		io.logger.Tracef(io.ctx, "error while outputing chunk to the user: %w", err)
	}
	utils.Go(context.Background(), func() {
		io.recorder.System(v)
	})
	return nil
}

func (io *GenericRequestor) Output(ctx context.Context, contextId string, msg *types.Message, completed bool) error {
	// Determine the message content
	aMsg := msg.String()
	if len(msg.ToolCalls) > 0 {
		aMsg = " "
	}

	// Transition the messaging state to agent speaking
	if err := io.messaging.Transition(internal_adapter_request_customizers.AgentSpeaking); err != nil {
		return nil
	}

	// Notify the system of the assistant message
	if err := io.Notify(ctx, &protos.AssistantConversationAssistantMessage{
		Time:      timestamppb.Now(),
		Id:        contextId,
		Completed: completed,
		Message: &protos.AssistantConversationAssistantMessage_Text{
			Text: &protos.AssistantConversationMessageTextContent{
				Content: aMsg,
			},
		},
	}); err != nil {
		io.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
	}

	// Handle completion logic
	if completed {
		if io.messaging.GetMode().Audio() {
			io.FinishSpeaking(contextId)
		}

		// Attempt to get the user's message and send a completion notification
		if in, err := io.messaging.GetMessage(type_enums.UserActor); err == nil {
			if err := io.Notify(ctx, &protos.AssistantConversationMessage{
				MessageId:               contextId,
				AssistantId:             io.assistant.Id,
				AssistantConversationId: io.assistantConversation.Id,
				Request:                 in.ToProto(),
				Response:                msg.ToProto(),
			}); err != nil {
				io.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
			}
			io.messaging.Transition(internal_adapter_request_customizers.AgentCompleted)
			return nil
		}

		// Notify the response if there is no user message
		if err := io.Notify(ctx, &protos.AssistantConversationMessage{
			MessageId:               contextId,
			AssistantId:             io.assistant.Id,
			AssistantConversationId: io.assistantConversation.Id,
			Response:                msg.ToProto(),
		}); err != nil {
			io.logger.Tracef(ctx, "error while outputting chunk to the user: %w", err)
		}
		io.messaging.Transition(internal_adapter_request_customizers.AgentCompleted)
		return nil
	}

	// Handle audio mode logic if not completed
	if io.messaging.GetMode().Audio() {
		if err := io.Speak(contextId, aMsg); err != nil {
			io.logger.Errorf("unable to speak for the user, please check the config error = %+v", err)
		}
	}

	return nil
}
