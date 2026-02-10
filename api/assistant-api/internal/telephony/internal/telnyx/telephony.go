// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_telnyx_telephony

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_streamers "github.com/rapidaai/api/assistant-api/internal/streamers"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type telnyxTelephony struct {
	appCfg *config.AssistantConfig
	logger commons.Logger
}

func NewTelnyxTelephony(config *config.AssistantConfig, logger commons.Logger) (internal_type.Telephony, error) {
	return &telnyxTelephony{
		appCfg: config,
		logger: logger,
	}, nil
}

func (tpc *telnyxTelephony) CatchAllStatusCallback(ctx *gin.Context) ([]types.Telemetry, error) {
	return nil, nil
}

func (tpc *telnyxTelephony) StatusCallback(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, assistantConversationId uint64) ([]types.Telemetry, error) {
	body, err := c.GetRawData()
	if err != nil {
		tpc.logger.Errorf("failed to read event body with error %+v", err)
		return nil, fmt.Errorf("not implemented")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		tpc.logger.Errorf("failed to parse body with error %+v", err)
		return nil, fmt.Errorf("failed to parse request body")
	}

	eventDetails := make(map[string]interface{})
	for key, value := range payload {
		eventDetails[key] = value
	}

	callStatusOrStreamEvent := eventDetails["status"]
	if streamEvent, ok := eventDetails["event"]; ok {
		callStatusOrStreamEvent = streamEvent
	}
	return []types.Telemetry{types.NewMetric("STATUS", fmt.Sprintf("%v", callStatusOrStreamEvent), utils.Ptr("Status of conversation")), types.NewEvent(fmt.Sprintf("%v", callStatusOrStreamEvent), eventDetails)}, nil
}

func (tpc *telnyxTelephony) OutboundCall(auth types.SimplePrinciple, toPhone string, fromPhone string, assistantId, assistantConversationId uint64, vaultCredential *protos.VaultCredential, opts utils.Option) ([]types.Telemetry, error) {
	mtds := []types.Telemetry{
		types.NewMetadata("telephony.toPhone", toPhone),
		types.NewMetadata("telephony.fromPhone", fromPhone),
		types.NewMetadata("telephony.provider", "telnyx"),
	}

	client, err := tpc.client(vaultCredential)
	if err != nil {
		return append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("authentication error: %s", err.Error())), types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))), err
	}

	answerUrl := fmt.Sprintf("wss://%s/%s",
		tpc.appCfg.PublicAssistantHost,
		internal_type.GetAnswerPath("telnyx", auth, assistantId, assistantConversationId, toPhone),
	)
	eventUrl := fmt.Sprintf("https://%s/%s",
		tpc.appCfg.PublicAssistantHost,
		internal_type.GetEventPath("telnyx", auth, assistantId, assistantConversationId),
	)

	callParams := map[string]interface{}{
		"to":        toPhone,
		"from":      fromPhone,
		"answer_url": answerUrl,
		"event_url":  eventUrl,
	}

	resp, err := tpc.createCall(client, callParams)
	if err != nil {
		return append(mtds, types.NewMetadata("telephony.error", fmt.Sprintf("API error: %s", err.Error())), types.NewMetric("STATUS", "FAILED", utils.Ptr("Status of telephony api"))), err
	}

	return append(mtds,
		types.NewMetadata("telephony.uuid", resp.CallID),
		types.NewEvent(resp.Status, resp),
		types.NewMetric("STATUS", "SUCCESS", utils.Ptr("Status of telephony api"))), nil
}

func (tpc *telnyxTelephony) CreateTeXML(mediaServer string, name string, path string, callback string, assistantId uint64, clientNumber string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
	<Connect>
		<Stream url="wss://%s/%s" name="%s" statusCallback="%s" statusCallbackEvent="initiated ringing answered completed">
			<Parameter name="assistant_id" value="%d"/>
			<Parameter name="client_number" value="%s"/>
		</Stream>
	</Connect>
</Response>`,
		mediaServer,
		path,
		name,
		callback,
		assistantId,
		clientNumber,
	)
}

func (tpc *telnyxTelephony) InboundCall(c *gin.Context, auth types.SimplePrinciple, assistantId uint64, clientNumber string, assistantConversationId uint64) error {
	c.Data(http.StatusOK, "text/xml", []byte(
		tpc.CreateTeXML(
			tpc.appCfg.PublicAssistantHost,
			fmt.Sprintf("%d__%d", assistantId, assistantConversationId),
			fmt.Sprintf("%d/%s/%d/%s",
				assistantId,
				clientNumber, assistantConversationId, auth.GetCurrentToken()),
			fmt.Sprintf("https://%s/%s", tpc.appCfg.PublicAssistantHost, internal_type.GetEventPath("telnyx", auth, assistantId, assistantConversationId)),
			assistantId, clientNumber),
	))
	return nil
}

func (tpc *telnyxTelephony) Streamer(c *gin.Context, connection *websocket.Conn, assistant *internal_assistant_entity.Assistant, conversation *internal_conversation_entity.AssistantConversation, vlt *protos.VaultCredential) internal_streamers.Streamer {
	return NewTelnyxWebsocketStreamer(tpc.logger, connection, assistant, conversation, vlt)
}

func (tpc *telnyxTelephony) ReceiveCall(c *gin.Context) (*string, []types.Telemetry, error) {
	queryParams := make(map[string]string)
	telemetry := []types.Telemetry{}
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	clientNumber, ok := queryParams["From"]
	if !ok || clientNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return nil, telemetry, fmt.Errorf("missing or empty 'from' query parameter")
	}

	if v, ok := queryParams["CallSid"]; ok && v != "" {
		telemetry = append(telemetry,
			types.NewMetadata("telephony.uuid", v),
		)
	}
	return utils.Ptr(clientNumber), append(telemetry, types.NewEvent("webhook", queryParams), types.NewMetric("STATUS", "SUCCESS", utils.Ptr("Status of telephony api"))), nil
}

func (tpc *telnyxTelephony) client(vaultCredential *protos.VaultCredential) (*http.Client, error) {
	apiKey, ok := vaultCredential.GetValue().AsMap()["api_key"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config api_key is not found")
	}
	apiSecret, ok := vaultCredential.GetValue().AsMap()["api_secret"]
	if !ok {
		return nil, fmt.Errorf("illegal vault config api_secret not found")
	}

	client := &http.Client{}
	return client, nil
}

func (tpc *telnyxTelephony) createCall(client *http.Client, params map[string]interface{}) (*TelnyxCallResponse, error) {
	apiKey, ok := params["api_key"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key not found")
	}

	urlStr := fmt.Sprintf("https://api.telnyx.com/v2/voice/calls")
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var callResp TelnyxCallResponse
	if err := json.Unmarshal(body, &callResp); err != nil {
		return nil, err
	}

	return &callResp, nil
}

func (tpc *telnyxTelephony) clientParam(vaultCredential *protos.VaultCredential) (string, string, error) {
	apiKey, ok := vaultCredential.GetValue().AsMap()["api_key"]
	if !ok {
		return "", "", fmt.Errorf("illegal vault config api_key is not found")
	}
	apiSecret, ok := vaultCredential.GetValue().AsMap()["api_secret"]
	if !ok {
		return "", "", fmt.Errorf("illegal vault config api_secret not found")
	}
	return apiKey.(string), apiSecret.(string), nil
}