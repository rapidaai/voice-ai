package internal_adapter_request_generic

import (
	"context"
	"errors"
	"strings"

	internal_adapter_request_customizers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/customizers"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (gr *GenericRequestor) GetBehavior() (*internal_assistant_gorm.AssistantDeploymentBehavior, error) {
	switch gr.source {
	case utils.PhoneCall:
		if a := gr.assistant; a != nil && a.AssistantPhoneDeployment != nil {
			return &a.AssistantPhoneDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Whatsapp:
		if a := gr.assistant; a != nil && a.AssistantWhatsappDeployment != nil {
			return &a.AssistantWhatsappDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.SDK:
		if a := gr.assistant; a != nil && a.AssistantApiDeployment != nil {
			return &a.AssistantApiDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.WebPlugin:
		if a := gr.assistant; a != nil && a.AssistantWebPluginDeployment != nil {
			return &a.AssistantWebPluginDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Debugger:
		if a := gr.assistant; a != nil && a.AssistantDebuggerDeployment != nil {
			return &a.AssistantDebuggerDeployment.AssistantDeploymentBehavior, nil
		}
	}
	return nil, errors.New("deployment is not enabled for source")

}

// start Mocked message
// =====================================
func (io *GenericRequestor) Greeting(ctx context.Context) error {
	behavior, err := io.GetBehavior()
	if err != nil {
		return nil
	}
	if behavior.Greeting == nil {
		return nil
	}
	greetingCnt := io.templateParser.Parse(*behavior.Greeting, io.GetArgs())
	if strings.TrimSpace(greetingCnt) == "" {
		io.logger.Warnf("empty greeting message, could be space in the table or argument contains space")
		return nil
	}

	inGreet := io.messaging.Create(type_enums.UserActor, "")
	greet := io.messaging.Create(type_enums.AssistantActor, greetingCnt)
	io.messaging.Transition(internal_adapter_request_customizers.UserCompleted)

	io.Speak(inGreet.GetId(), greetingCnt)
	io.FinishSpeaking(inGreet.GetId())
	io.
		Notify(
			ctx,
			&lexatic_backend.AssistantConversationAssistantMessage{
				Time:      timestamppb.Now(),
				Id:        inGreet.GetId(),
				Completed: true,
				Message: &lexatic_backend.AssistantConversationAssistantMessage_Text{
					Text: &lexatic_backend.AssistantConversationMessageTextContent{
						Content: greetingCnt,
					},
				},
			},
		)
	io.
		Notify(
			ctx,
			&lexatic_backend.AssistantConversationMessage{
				MessageId:               inGreet.GetId(),
				AssistantId:             io.assistant.Id,
				AssistantConversationId: io.assistantConversation.Id,
				Request:                 inGreet.ToProto(),
				Response:                greet.ToProto(),
			},
		)
	return nil
}
