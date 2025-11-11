package internal_telephony

import (
	"encoding/json"
	"fmt"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type twilioTelephony struct {
	logger     commons.Logger
	accountSid string
	authToken  string
	cfg        utils.Option
}

func NewTwilioTelephony(
	logger commons.Logger,
	vaultCredential *lexatic_backend.VaultCredential,
	cfg utils.Option) (Telephony, error) {
	accountSid, ok := vaultCredential.GetValue().AsMap()["account_sid"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config accountSid is not found")
	}
	authToken, ok := vaultCredential.GetValue().AsMap()["account_token"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config account_token not found")
	}
	return &twilioTelephony{
		cfg:        cfg,
		logger:     logger,
		accountSid: accountSid.(string),
		authToken:  authToken.(string),
	}, nil
}

func (tpc *twilioTelephony) TwilioClientParam() twilio.ClientParams {
	return twilio.ClientParams{
		Username: tpc.accountSid,
		Password: tpc.authToken,
	}
}

func (tpc *twilioTelephony) TwilioClient() *twilio.RestClient {
	return twilio.NewRestClientWithParams(tpc.TwilioClientParam())
}

func (tpc *twilioTelephony) CreateCall(
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistantId, sessionId uint64,
) (map[string]interface{}, error) {

	callParams := &openapi.CreateCallParams{}
	callParams.SetTo(toPhone)
	callParams.SetFrom(fromPhone)

	switch auth.Type() {
	case "project":
		callParams.SetTwiml(
			CreateTwinML(
				GetAnswerPath("twilio", auth, assistantId,
					sessionId,
					toPhone,
				),
				assistantId,
				toPhone),
		)

	case "user":
		callParams.SetTwiml(
			CreateTwinML(
				GetAnswerPath("twilio", auth, assistantId,
					sessionId,
					toPhone,
				),
				assistantId,
				toPhone),
		)

	}
	tpc.logger.Debugf("calling params %+v", callParams)
	resp, err := tpc.TwilioClient().Api.CreateCall(callParams)
	if err != nil {
		return nil, err
	}
	// Convert entire response to JSON, then to map
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("error marshaling response to JSON: %v", err)
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(jsonData, &responseMap)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON to map: %v", err)
	}
	return responseMap, nil
}

func CreateTwinML(path string, assistantId uint64, clientNumber string) string {
	redirectUrl := "assistant-01.rapida.ai"
	// redirectUrl = "integral-presently-cub.ngrok-free.app"

	return fmt.Sprintf(`
	    <Response>
		 	<Connect>
	        	<Stream url="wss://%s/%s">
					<Parameter name="assistant_id" value="%d"/>
					<Parameter name="client_number" value="%s"/>
				</Stream>
			</Connect>
	    </Response>
	`,
		redirectUrl,
		path,
		assistantId,
		clientNumber,
	)
}

func GetAnswerPath(provider string, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64, toPhone string) string {
	switch auth.Type() {
	case "project":
		return fmt.Sprintf("v1/talk/%s/prj/%d/%s/%d/%s",
			provider,
			assistantId,
			toPhone,
			assistantConversationId,
			// authentication
			auth.GetCurrentToken())
	default:
		return fmt.Sprintf("v1/talk/%s/usr/%d/%s/%d/%s/%d/%d",
			provider,
			assistantId,
			toPhone,
			assistantConversationId,
			// authentication
			auth.GetCurrentToken(),
			*auth.GetUserId(),
			*auth.GetCurrentProjectId())
	}
}
