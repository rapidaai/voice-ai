// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_factory "github.com/rapidaai/api/assistant-api/internal/factory"
	telephony "github.com/rapidaai/api/assistant-api/internal/factory/telephony"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (cApi *ConversationApi) UnviersalCallback(c *gin.Context) {
	body, err := c.GetRawData() // Extract raw request body
	if err != nil {
		cApi.logger.Errorf("failed to read event body with error %+v", err)
	}
	cApi.logger.Debugf("event from exotel | body: %s", string(body))
}

func (cApi *ConversationApi) Callback(c *gin.Context) {
	iAuth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		cApi.logger.Debugf("illegal unable to authenticate")
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthenticated request"})
		return
	}
	assistantIdStr := c.Param("assistantId")
	assistantId, err := strconv.ParseUint(assistantIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return
	}

	conversationIdStr := c.Param("conversationId")
	conversationId, err := strconv.ParseUint(conversationIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	tlp := c.Param("telephony")
	_telephony, err := telephony.GetTelephony(
		telephony.Telephony(tlp),
		cApi.cfg,
		cApi.logger,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telephony"})
		return
	}

	mtr, evts, err := _telephony.Callback(c, iAuth, assistantId, conversationId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	_, err = cApi.
		assistantConversationService.
		ApplyConversationTelephonyEvent(c, iAuth, tlp, assistantId, conversationId, evts)
	if err != nil {
		c.Status(http.StatusOK)
		return
	}

	_, err = cApi.
		assistantConversationService.
		ApplyConversationMetrics(c, iAuth, assistantId, conversationId, mtr)
	if err != nil {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusCreated)
	return
}

// CallReciever handles incoming calls for the given assistant.
// @Router /v1/call/:assistantId [post]
// @Summary Recieve call for given assistant
// @Produce json
// @Success 200 {object} app.Response
// @Failure 500 {object} app.Response
func (cApi *ConversationApi) CallReciever(c *gin.Context) {
	iAuth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		cApi.logger.Debugf("illegal unable to authenticate")
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthenticated request"})
		return
	}

	assistantIdStr := c.Param("assistantId")
	assistantId, err := strconv.ParseUint(assistantIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return
	}

	tlp := c.Param("telephony")
	_telephony, err := telephony.GetTelephony(
		telephony.Telephony(tlp),
		cApi.cfg,
		cApi.logger,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Telephony is not connected"})
		return
	}

	clientNumber, ok := _telephony.GetCaller(c)
	if !ok {
		cApi.logger.Debugf("Missing 'Caller' OR 'from' number in Twilio request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'Caller' number"})
		return
	}

	assistant, err := cApi.
		assistantService.
		Get(c,
			iAuth,
			assistantId,
			utils.
				GetVersionDefinition("latest"),
			&internal_services.
				GetAssistantOption{
				InjectPhoneDeployment: true,
			})
	if err != nil {
		cApi.logger.Debugf("illegal unable to find assistant %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to initiate talker"})
		return
	}

	conversation, err := cApi.
		assistantConversationService.
		CreateConversation(
			c,
			iAuth,
			internal_factory.Identifier(utils.PhoneCall, c, iAuth, clientNumber),
			assistant.Id,
			assistant.AssistantProviderId,
			type_enums.DIRECTION_INBOUND,
			utils.PhoneCall,
		)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to initiate talker"})
		return
	}

	cApi.
		assistantConversationService.
		ApplyConversationMetrics(
			c, iAuth,
			assistantId,
			conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_CONNECTED)},
		)
	err = _telephony.ReceiveCall(c, iAuth, assistant.Id, clientNumber, conversation.Id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to initiate talker"})
		return
	}
	return

}

func (cApi *ConversationApi) CallTalker(c *gin.Context) {
	start := time.Now()
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	websocketConnection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		cApi.logger.Errorf("illegal while upgrading vonage talker with error %v", err)
		return
	}
	cApi.logger.Benchmark("ConversationApi.VonageCallTalker.upgradeConnection", time.Since(start))

	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		cApi.logger.Errorf("illegal unable to authenticate")
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthenticated request"})
		return
	}

	cApi.logger.Benchmark("conversationapi.VonageCallTalker.GetAuthPrinciple", time.Since(start))
	// Extract the client source from the stream context
	assistantId, err := strconv.ParseUint(c.Param("assistantId"), 10, 64)
	if err != nil {
		cApi.logger.Errorf("Invalid assistantId: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistantId"})
		return
	}

	conversationId, err := strconv.ParseUint(c.Param("conversationId"), 10, 64)
	if err != nil {
		cApi.logger.Errorf("Invalid conversationId: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversationId"})
		return
	}

	identifier := c.Param("identifier")
	tlp := c.Param("telephony")
	_telephony, err := telephony.GetTelephony(
		telephony.Telephony(tlp),
		cApi.cfg,
		cApi.logger,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telephony"})
		return
	}

	talker, err := internal_factory.GetTalker(
		utils.PhoneCall,
		c,
		cApi.cfg,
		cApi.logger,
		cApi.postgres,
		cApi.opensearch,
		cApi.redis,
		cApi.storage,
		_telephony.Streamer(
			c,
			websocketConnection,
			assistantId, "latest", conversationId,
		),
	)
	if err != nil {
		cApi.logger.Errorf("illegal to get talker %v", err)
		return
	}
	cidentifier := internal_factory.
		Identifier(utils.PhoneCall, c, auth, identifier)

	err = talker.Connect(
		c, auth, cidentifier, &protos.AssistantConversationConfiguration{
			AssistantConversationId: conversationId,
			Assistant: &protos.AssistantDefinition{
				AssistantId: assistantId,
				Version:     "latest",
			},
		},
	)
	if err != nil {
		cApi.logger.Errorf("illegal to connect with talker %v", err)
		return
	}
	talker.Talk(
		c,
		auth,
		cidentifier,
	)
}
