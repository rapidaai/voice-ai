package internal_vonage_telephony

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/config"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	internal_telephony "github.com/rapidaai/api/assistant-api/internal/telephony"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/vonage/vonage-go-sdk"
	"github.com/vonage/vonage-go-sdk/ncco"
)

type vonageTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewVonageTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_telephony.Telephony, error) {
	return &vonageTelephony{
		logger: logger,
		appCfg: config,
	}, nil
}

func (tpc *vonageTelephony) CatchAllCallback(ctx *gin.Context) (*string, []*types.Metric, []*types.Event, error) {
	return nil, nil, nil, nil
}
func (tpc *vonageTelephony) Callback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) ([]*types.Metric, []*types.Event, error) {
	body, err := c.GetRawData() // Extract raw request body
	if err != nil {
		tpc.logger.Errorf("failed to read request body with error %+v", err)
		return nil, nil, fmt.Errorf("failed to read request body")
	}

	// Parse the JSON body
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		tpc.logger.Errorf("failed to parse request body: %+v", err)
		return nil, nil, fmt.Errorf("failed to parse request body")
	}

	// Extract status from payload
	status, ok := payload["status"].(string)
	if !ok {
		tpc.logger.Errorf("status not found or invalid in payload")
		return nil, nil, fmt.Errorf("status not found in payload")
	}
	tpc.logger.Debugf("event processed | status: %s, payload: %+v", status, payload)
	return []*types.Metric{types.NewMetric("STATUS", status, utils.Ptr("Status of conversation"))}, []*types.Event{types.NewEvent(status, payload)}, nil
}

func (vt *vonageTelephony) Auth(
	vaultCredential *protos.VaultCredential,
	opts utils.Option) (vonage.Auth, error) {
	privateKey, ok := vaultCredential.GetValue().AsMap()["private_key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config privateKey is not found")
	}
	applicationId, ok := vaultCredential.GetValue().AsMap()["application_id"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config application_id is not found")
	}
	clientAuth, err := vonage.CreateAuthFromAppPrivateKey(applicationId.(string), []byte(privateKey.(string)))
	if err != nil {
		return nil, err
	}
	return clientAuth, nil
}

func (vt *vonageTelephony) MakeCall(
	auth types.SimplePrinciple,
	toPhone string,
	fromPhone string,
	assistantId, assistantConversationId uint64,
	vaultCredential *protos.VaultCredential,
	opts utils.Option,
) ([]*types.Metadata, []*types.Metric, []*types.Event, error) {
	mtds := []*types.Metadata{
		types.NewMetadata("telephony.toPhone", toPhone),
		types.NewMetadata("telephony.fromPhone", toPhone),
		types.NewMetadata("telephony.provider", "vonage"),
	}
	event := []*types.Event{
		types.NewEvent("api-call", map[string]interface{}{}),
	}

	cAuth, err := vt.Auth(vaultCredential, opts)
	if err != nil {
		mtds = append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("authentication error: %s", err.Error())))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}
	ct := vonage.NewVoiceClient(cAuth)

	connectAction := ncco.Ncco{}
	nccoConnect := ncco.ConnectAction{
		EventType: "synchronous",
		EventUrl:  []string{fmt.Sprintf("https://%s/%s", vt.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("vonage", auth, assistantId, assistantConversationId))},
		Endpoint: []ncco.Endpoint{ncco.WebSocketEndpoint{
			Uri: fmt.Sprintf("wss://%s/%s",
				vt.appCfg.PublicAssistantHost,
				internal_telephony.GetAnswerPath("vonage", auth, assistantId, assistantConversationId, toPhone)),
			ContentType: "audio/l16;rate=16000",
		}},
	}
	connectAction.AddAction(nccoConnect)
	result, vErr, apiError := ct.CreateCall(
		vonage.CreateCallOpts{
			From: vonage.CallFrom{Type: "phone", Number: fromPhone},
			To:   vonage.CallTo{Type: "phone", Number: toPhone},
			Ncco: connectAction,
		})

	if apiError != nil {
		mtds = append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("API error: %s", apiError.Error())))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	if vErr.Error != nil {
		mtds = append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("Calling error: %v", vErr.Error)))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	mtds = append(mtds, types.NewMetadata("telephony.conversation_reference", result.ConversationUuid))
	event = append(event, types.NewEvent(result.Status, result))
	return mtds, []*types.Metric{types.NewMetric("STATUS", "SUCCESS", utils.Ptr("Status of telephony api"))}, event, nil
}

func (vt *vonageTelephony) ReceiveCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	c.JSON(http.StatusOK, []gin.H{
		{
			"action":    "connect",
			"eventType": "synchronous",
			"eventUrl":  []string{fmt.Sprintf("https://%s/%s", vt.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("vonage", auth, assistantId, assistantConversationId))},
			"endpoint": []gin.H{
				{
					"type": "websocket",
					"uri": fmt.Sprintf("wss://%s/%s",
						vt.appCfg.PublicAssistantHost,
						internal_telephony.GetAnswerPath("vonage", auth, assistantId, assistantConversationId, clientNumber)),
					"content-type": "audio/l16;rate=16000",
				},
			},
		},
	})
	return nil
}

func (tpc *vonageTelephony) Streamer(c *gin.Context, connection *websocket.Conn, assistantID uint64, assistantVersion string, assistantConversationID uint64) internal_streamers.Streamer {
	return NewVonageWebsocketStreamer(tpc.logger, connection, assistantID,
		assistantVersion,
		assistantConversationID)
}

func (tpc *vonageTelephony) GetCaller(c *gin.Context) (string, bool) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["from"]
	return clientNumber, ok

}
