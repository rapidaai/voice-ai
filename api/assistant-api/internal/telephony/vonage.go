package internal_telephony

import (
	"fmt"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"github.com/vonage/vonage-go-sdk"
	"github.com/vonage/vonage-go-sdk/ncco"
)

type vonageTelephony struct {
	cfg           utils.Option
	logger        commons.Logger
	privateKey    string
	applicationId string
}

func NewVonageTelephony(logger commons.Logger, vaultCredential *lexatic_backend.VaultCredential, cfg utils.Option) (Telephony, error) {
	privateKey, ok := vaultCredential.GetValue().AsMap()["private_key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config privateKey is not found")
	}
	applicationId, ok := vaultCredential.GetValue().AsMap()["application_id"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config application_id is not found")
	}
	return &vonageTelephony{
		cfg:           cfg,
		logger:        logger,
		privateKey:    privateKey.(string),
		applicationId: applicationId.(string),
	}, nil
}

func (vt *vonageTelephony) CreateCall(
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistantId, assistantConversationId uint64) (map[string]interface{}, error) {

	redirectUrl := "assistant-01.rapida.ai"
	// redirectUrl = "integral-presently-cub.ngrok-free.app"

	clientAuth, _ := vonage.CreateAuthFromAppPrivateKey(vt.applicationId, []byte(vt.privateKey))
	client := vonage.NewVoiceClient(clientAuth)

	connectAction := ncco.Ncco{}
	nccoConnect := ncco.ConnectAction{
		EventType: "synchronous",
		Endpoint: []ncco.Endpoint{ncco.WebSocketEndpoint{
			Uri: fmt.Sprintf("wss://%s/%s",
				redirectUrl,
				GetAnswerPath("vonage", auth, assistantId, assistantConversationId, toPhone)),
			ContentType: "audio/l16;rate=16000",
		}},
	}
	connectAction.AddAction(nccoConnect)
	result, err, apiError := client.CreateCall(vonage.CreateCallOpts{
		From: vonage.CallFrom{Type: "phone", Number: fromPhone},
		To:   vonage.CallTo{Type: "phone", Number: toPhone},
		Ncco: connectAction,
	})

	if apiError != nil {
		vt.logger.Errorf("error while calling vonage %+v", apiError)
		return nil, apiError
	}

	if err.Error != nil {
		vt.logger.Errorf("error while calling vonage %+v", err.Error)
		return nil, fmt.Errorf("unable to make call with vonage")
	}
	return map[string]interface{}{
		"uuid":              result.Uuid,
		"status":            result.Status,
		"direction":         result.Direction,
		"conversation_uuid": result.ConversationUuid,
	}, nil
}
