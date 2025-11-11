package internal_adapter_request_generic

import (
	"context"
	"fmt"
	"time"

	internal_adapter_request_customizers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/customizers"
	internal_analyzers "github.com/rapidaai/api/assistant-api/internal/analyzers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
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
			&lexatic_backend.AssistantConversationUserMessage{
				Id: msgi.GetId(),
				Message: &lexatic_backend.AssistantConversationUserMessage_Text{
					Text: &lexatic_backend.AssistantConversationMessageTextContent{
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
	act internal_analyzers.SpeechEndActivity) error {
	start := time.Now()
	defer io.logger.Benchmark("io.OnSilenceBreakActivity", time.Since(start))

	msg, err := io.messaging.GetMessage(type_enums.UserActor)
	if err != nil {
		io.logger.Tracef(ctx, "illegal message state with error %v", err)
		return nil
	}
	io.messaging.Transition(internal_adapter_request_customizers.UserCompleted)
	if err := io.Notify(ctx,
		&lexatic_backend.AssistantConversationUserMessage{
			Id: msg.GetId(),
			Message: &lexatic_backend.AssistantConversationUserMessage_Text{
				Text: &lexatic_backend.AssistantConversationMessageTextContent{
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
			&lexatic_backend.AssistantConversationInterruption{
				Type: lexatic_backend.AssistantConversationInterruption_INTERRUPTION_TYPE_WORD,
				Time: timestamppb.Now(),
			})
	default:
		if err := io.messaging.Transition(internal_adapter_request_customizers.Interrupt); err != nil {
			return nil
		}
		if io.messaging.GetMode().Audio() {
			io.recorder.Interrupt()
		}
		io.Notify(ctx, &lexatic_backend.AssistantConversationInterruption{
			Type: lexatic_backend.AssistantConversationInterruption_INTERRUPTION_TYPE_VAD,
			Time: timestamppb.Now(),
		})
	}
	return nil

}

// Start Input
// =====================================
func (io *GenericRequestor) Input(message *lexatic_backend.AssistantConversationUserMessage) error {
	switch msg := message.GetMessage().(type) {
	case *lexatic_backend.AssistantConversationUserMessage_Audio:
		return io.InputAudio(io.Context(), msg.Audio.GetContent())
	case *lexatic_backend.AssistantConversationUserMessage_Text:
		return io.InputText(io.Context(), msg.Text.GetContent())
	default:
		return fmt.Errorf("illegal input from the user %+v", msg)
	}

}

func (io *GenericRequestor) InputAudio(ctx context.Context, in []byte) error {

	// record the user input
	utils.Go(context.Background(), func() {
		io.recorder.User(in)
	})

	// switch mode
	io.messaging.SwitchMode(type_enums.AudioMode)

	// listen to audio as this input audio
	io.ListenAudio(ctx, in)
	return nil
}

func (io *GenericRequestor) InputText(ctx context.Context, msg string) error {

	// changing to text mode
	io.messaging.SwitchMode(type_enums.TextMode)

	// mark it interrupted
	io.messaging.Transition(internal_adapter_request_customizers.Interrupted)

	//
	interim := io.messaging.Create(type_enums.UserActor, msg)

	// notify the user message
	io.
		Notify(
			ctx,
			&lexatic_backend.AssistantConversationUserMessage{
				Time:      timestamppb.Now(),
				Id:        interim.GetId(),
				Completed: false,
				Message: &lexatic_backend.AssistantConversationUserMessage_Text{
					Text: &lexatic_backend.AssistantConversationMessageTextContent{
						Content: interim.String(),
					},
				},
			},
		)

	return io.
		ListenText(
			ctx,
			&internal_analyzers.UserTextAnalyzerInput{
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
		io.logger.Warnf("testing: context id mismatched %+v current %v", contextId, inputMessage.GetId())
		return nil
	}

	if err := io.messaging.Transition(internal_adapter_request_customizers.AgentSpeaking); err != nil {
		io.logger.Warnf("testing: illegal transition to speaking")
		return nil
	}

	if err := io.
		Notify(
			io.Context(),
			&lexatic_backend.AssistantConversationAssistantMessage{
				Time:      timestamppb.Now(),
				Id:        contextId,
				Completed: completed,
				Message: &lexatic_backend.AssistantConversationAssistantMessage_Audio{
					Audio: &lexatic_backend.AssistantConversationMessageAudioContent{
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

func (io *GenericRequestor) Output(
	ctx context.Context,
	contextId string,
	msg *types.Message,
	completed bool,
) error {
	inputMessage, err := io.messaging.GetMessage(type_enums.UserActor)
	if err != nil {
		io.logger.Debug("illegal output, as there is no input specified")
		return err
	}
	// //
	if contextId != inputMessage.GetId() {
		io.logger.Warnf("testing: context id mismatched %+v current %v", contextId, inputMessage.GetId())
		return nil
	}

	aMsg := msg.String()
	if len(msg.ToolCalls) > 0 {
		aMsg = " "
	}

	if err := io.messaging.Transition(internal_adapter_request_customizers.AgentSpeaking); err != nil {
		io.logger.Warnf("Can't notify the assistant think as user is speaking")
		return nil
	}

	io.
		Notify(
			ctx,
			&lexatic_backend.AssistantConversationAssistantMessage{
				Time:      timestamppb.Now(),
				Id:        contextId,
				Completed: completed,
				Message: &lexatic_backend.AssistantConversationAssistantMessage_Text{
					Text: &lexatic_backend.AssistantConversationMessageTextContent{
						Content: msg.String(),
					},
				},
			},
		)
	if completed {
		if io.messaging.GetMode().Audio() {
			io.FinishSpeaking(contextId)
		}
		io.
			Notify(
				ctx,
				&lexatic_backend.AssistantConversationMessage{
					MessageId:               contextId,
					AssistantId:             io.assistant.Id,
					AssistantConversationId: io.assistantConversation.Id,
					Request:                 inputMessage.ToProto(),
					Response:                msg.ToProto(),
				},
			)

		//
		io.messaging.Transition(internal_adapter_request_customizers.AgentCompleted)
		return nil
	}

	if io.messaging.GetMode().Audio() {
		err := io.Speak(
			contextId,
			aMsg,
		)
		if err != nil {
			io.logger.Errorf("unable to speak for the user, please check the config error = %+v", err)
		}
	}
	return nil
}
