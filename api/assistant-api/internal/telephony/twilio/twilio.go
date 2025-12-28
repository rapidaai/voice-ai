package internal_twilio_telephony

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/config"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	internal_telephony "github.com/rapidaai/api/assistant-api/internal/telephony"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type twilioTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewTwilioTelephony(
	config *config.AssistantConfig,
	logger commons.Logger) (internal_telephony.Telephony, error) {
	return &twilioTelephony{
		appCfg: config,
		logger: logger,
	}, nil
}

func (tpc *twilioTelephony) CatchAllCallback(ctx *gin.Context) (*string, []*types.Metric, []*types.Event, error) {
	return nil, nil, nil, nil
}
func (tpc *twilioTelephony) Callback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) ([]*types.Metric, []*types.Event, error) {
	body, err := c.GetRawData() // Extract raw request body
	if err != nil {
		tpc.logger.Errorf("failed to read event body with error %+v", err)
		return nil, nil, fmt.Errorf("not implimented")
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		tpc.logger.Errorf("failed to parse body with error %+v", err)
		return nil, nil, fmt.Errorf("failed to parse request body")
	}

	eventDetails := make(map[string]interface{})
	for key, value := range values {
		if len(value) > 0 {
			eventDetails[key] = value[0]
		} else {
			eventDetails[key] = nil
		}
	}

	callStatusOrStreamEvent := eventDetails["CallStatus"]
	if streamEvent, ok := eventDetails["StreamEvent"]; ok {
		callStatusOrStreamEvent = streamEvent
	}

	tpc.logger.Infof("parsed twilio event details: %+v", eventDetails)
	return []*types.Metric{types.NewMetric("STATUS", fmt.Sprintf("%v", callStatusOrStreamEvent), utils.Ptr("Status of conversation"))}, []*types.Event{types.NewEvent(fmt.Sprintf("%v", callStatusOrStreamEvent), eventDetails)}, nil

}
func (tpc *twilioTelephony) TwilioClientParam(vaultCredential *protos.VaultCredential,
	opts utils.Option) (*twilio.ClientParams, error) {
	accountSid, ok := vaultCredential.GetValue().AsMap()["account_sid"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config accountSid is not found")
	}
	authToken, ok := vaultCredential.GetValue().AsMap()["account_token"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config account_token not found")
	}
	return &twilio.ClientParams{
		Username: accountSid.(string),
		Password: authToken.(string),
	}, nil
}

func (tpc *twilioTelephony) TwilioClient(vaultCredential *protos.VaultCredential,
	opts utils.Option) (*twilio.RestClient, error) {
	clientParams, err := tpc.TwilioClientParam(vaultCredential, opts)
	if err != nil {
		return nil, err
	}
	return twilio.NewRestClientWithParams(*clientParams), nil
}

func (tpc *twilioTelephony) MakeCall(
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
		types.NewMetadata("telephony.provider", "twilio"),
	}
	event := []*types.Event{
		types.NewEvent("api-call", map[string]interface{}{}),
	}

	client, err := tpc.TwilioClient(vaultCredential, opts)
	if err != nil {
		mtds = append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("authentication error: %s", err.Error())))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}
	callParams := &openapi.CreateCallParams{}
	callParams.SetTo(toPhone)
	callParams.SetFrom(fromPhone)
	callParams.SetStatusCallback(
		fmt.Sprintf("https://%s/%s", tpc.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("twilio", auth, assistantId, assistantConversationId)),
	)
	callParams.SetStatusCallbackEvent([]string{
		"initiated", "ringing", "answered", "completed",
	})
	callParams.SetStatusCallbackMethod("POST")
	callParams.SetTwiml(
		tpc.CreateTwinML(
			tpc.appCfg.PublicAssistantHost,
			fmt.Sprintf("%d__%d", assistantId, assistantConversationId),
			internal_telephony.GetAnswerPath("twilio", auth, assistantId,
				assistantConversationId,
				toPhone,
			),
			fmt.Sprintf("https://%s/%s", tpc.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("twilio", auth, assistantId, assistantConversationId)),
			assistantId,
			toPhone),
	)
	resp, err := client.Api.CreateCall(callParams)
	if err != nil || resp.Status == nil || resp.Sid == nil {
		mtds = append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("API error: %s", err.Error())))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	event = append(event, types.NewEvent(*resp.Status, resp))
	mtds = append(mtds, types.NewMetadata("telephony.conversation_reference", *resp.Sid))
	return mtds, []*types.Metric{types.NewMetric("STATUS", "SUCCESS", utils.Ptr("Status of telephony api"))}, event, nil
}

func (tpc *twilioTelephony) CreateTwinML(mediaServer string, name, path string, callback string, assistantId uint64, clientNumber string) string {
	return fmt.Sprintf(`
	    <Response>
		 	<Connect>
	        	<Stream url="wss://%s/%s" name="%s" statusCallback="%s" statusCallbackEvent="initiated ringing answered completed">
					<Parameter name="assistant_id" value="%d"/>
					<Parameter name="client_number" value="%s"/>
				</Stream>
			</Connect>
	    </Response>
	`,
		mediaServer,
		path,
		name,
		callback,
		assistantId,
		clientNumber,
	)
}

func (tpc *twilioTelephony) ReceiveCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	c.Data(http.StatusOK, "text/xml", []byte(
		tpc.CreateTwinML(
			tpc.appCfg.PublicAssistantHost,
			fmt.Sprintf("%d__%d", assistantId, assistantConversationId),
			fmt.Sprintf("v1/talk/twilio/prj/%d/%s/%d/%s",
				assistantId,
				clientNumber, assistantConversationId, auth.GetCurrentToken()),
			fmt.Sprintf("https://%s/%s", tpc.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("twilio", auth, assistantId, assistantConversationId)),
			assistantId, clientNumber),
	))
	return nil
}

func (tpc *twilioTelephony) Streamer(c *gin.Context, connection *websocket.Conn, assistantID uint64, assistantVersion string, assistantConversationID uint64) internal_streamers.Streamer {
	return NewTwilioWebsocketStreamer(tpc.logger, connection, assistantID,
		assistantVersion,
		assistantConversationID)
}

func (tpc *twilioTelephony) GetCaller(c *gin.Context) (string, bool) {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["from"]
	return clientNumber, ok

}

func (tpc *twilioTelephony) InputStreamConfig() *protos.StreamConfig {
	return &protos.StreamConfig{
		Audio: internal_audio.NewMulaw8khzMonoAudioConfig(),
	}
}

func (tpc *twilioTelephony) OutputStreamConfig() *protos.StreamConfig {
	return &protos.StreamConfig{
		Audio: internal_audio.NewMulaw8khzMonoAudioConfig(),
	}
}
