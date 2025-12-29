package internal_exotel_telephony

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/config"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	internal_telephony "github.com/rapidaai/api/assistant-api/internal/telephony"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type exotelTelephony struct {
	logger commons.Logger
	appCfg *config.AssistantConfig
}

func (tpc *exotelTelephony) CatchAllCallback(ctx *gin.Context) (*string, []*types.Metric, []*types.Event, error) {
	return nil, nil, nil, nil
}

// EventCallback implements [Telephony].
func (tpc *exotelTelephony) Callback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) ([]*types.Metric, []*types.Event, error) {
	form, err := c.MultipartForm()
	if err != nil {
		tpc.logger.Errorf("failed to parse multipart form-data with error %+v", err)
		return nil, nil, fmt.Errorf("failed to parse multipart form-data")
	}

	eventDetails := make(map[string]interface{})
	for key, values := range form.Value {
		if len(values) > 0 {
			eventDetails[key] = values[0] // Take only the first value for simplicity
		} else {
			eventDetails[key] = nil
		}
	}
	callStatus := eventDetails["Status"]
	return []*types.Metric{types.NewMetric("STATUS", fmt.Sprintf("%v", callStatus), utils.Ptr("Status of call or update"))},
		[]*types.Event{types.NewEvent(fmt.Sprintf("%v", callStatus), eventDetails)},
		nil

}

func NewExotelTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_telephony.Telephony, error) {
	return &exotelTelephony{
		logger: logger,
		appCfg: config,
	}, nil
}

func (tpc *exotelTelephony) ClientUrl(
	vaultCredential *protos.VaultCredential,
	opts utils.Option) (*string, error) {
	accountSid, ok := vaultCredential.GetValue().AsMap()["account_sid"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config accountSid is not found")
	}
	clientId, ok := vaultCredential.GetValue().AsMap()["client_id"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config client_id not found")
	}
	authToken, ok := vaultCredential.GetValue().AsMap()["client_secret"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config")
	}
	return utils.Ptr(fmt.Sprintf("https://%s:%s@api.exotel.com/v1/Accounts/%s/Calls/connect.json",
		clientId.(string), authToken.(string), accountSid.(string))), nil

}

func (tpc *exotelTelephony) AppUrl(
	vaultCredential *protos.VaultCredential,
	opts utils.Option) (*string, error) {
	accountSid, ok := vaultCredential.GetValue().AsMap()["account_sid"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config accountSid is not found")
	}
	app_id, err := opts.GetString("app_id")
	if err != nil {
		return nil, fmt.Errorf("illegal app_id option is not found")
	}
	return utils.Ptr(fmt.Sprintf("http://my.exotel.com/%s/exoml/start_voice/%s", accountSid.(string), app_id)), nil

}

func (tpc *exotelTelephony) MakeCall(
	auth types.SimplePrinciple,
	// customer number
	toPhone string,
	// exo number
	fromPhone string,
	assistantId, sessionId uint64,
	vaultCredential *protos.VaultCredential,
	opts utils.Option) ([]*types.Metadata, []*types.Metric, []*types.Event, error) {
	mtds := []*types.Metadata{
		types.NewMetadata("telephony.toPhone", toPhone),
		types.NewMetadata("telephony.fromPhone", toPhone),
		types.NewMetadata("telephony.provider", "exotel"),
	}
	event := []*types.Event{
		types.NewEvent("api-call", map[string]interface{}{}),
	}
	clientUrl, err := tpc.ClientUrl(vaultCredential, opts)
	if err != nil {
		event = append(event, types.NewEvent("FAILED", "Failed to build url, check credentials"))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	appUrl, err := tpc.AppUrl(vaultCredential, opts)
	if err != nil {
		event = append(event, types.NewEvent("FAILED", "Failed to build app url"))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	formData := url.Values{}
	formData.Set("From", toPhone)
	formData.Set("CallerId", fromPhone)
	formData.Set("To", fromPhone)
	formData.Set("Url", *appUrl)
	formData.Set("StatusCallback", fmt.Sprintf("https://%s/%s", tpc.appCfg.PublicAssistantHost, internal_telephony.GetEventPath("exotel", auth, assistantId, sessionId)))
	tpc.logger.Debugf("Caller %+v and url %s", formData, *clientUrl)
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", *clientUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))}, event, err
	}
	if resp.StatusCode != http.StatusOK {
		tpc.logger.Errorf("Unexpected HTTP Status: %d, Response Body: %s\n", resp.StatusCode, string(bodyBytes))
		event = append(event, types.NewEvent("Failed", string(bodyBytes)))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony API - HTTP error"))}, event, fmt.Errorf("status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var jsonResponse struct {
		Call struct {
			Sid              string  `json:"Sid"`
			Status           string  `json:"Status"`
			RecordingUrl     string  `json:"RecordingUrl"`
			ConversationUuid *string `json:"ParentCallSid"` // Use pointers for nullable fields
		} `json:"Call"`
	}

	// Wrap the JSON decoding in a detailed error message
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		event = append(event, types.NewEvent(jsonResponse.Call.Status, "Failed to decode response"))
		return mtds, []*types.Metric{types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of transaction"))}, event, err
	}

	mtds = append(mtds, types.NewMetadata("telephony.conversation_reference", jsonResponse.Call.Sid))
	event = append(event, types.NewEvent(jsonResponse.Call.Status, jsonResponse))
	return mtds, []*types.Metric{types.NewMetric("STATUS", "SUCCESS", utils.Ptr("Status of telephony api"))}, event, nil
}

func (tpc *exotelTelephony) ReceiveCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	response := map[string]string{
		"url": fmt.Sprintf("wss://%s/%s",
			tpc.appCfg.PublicAssistantHost,
			internal_telephony.GetAnswerPath("exotel", auth, assistantId, assistantConversationId, clientNumber)),
	}

	c.JSON(http.StatusOK, response)
	return nil
}

func (tpc *exotelTelephony) Streamer(c *gin.Context, connection *websocket.Conn, assistantID uint64, assistantVersion string, assistantConversationID uint64) internal_streamers.Streamer {
	return NewExotelWebsocketStreamer(tpc.logger, connection, assistantID,
		assistantVersion,
		assistantConversationID)
}

func (tpc *exotelTelephony) GetCaller(c *gin.Context) (string, bool) {
	// Extract query parameters
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["CallFrom"]
	return clientNumber, ok
}
