// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	internal_adapter "github.com/rapidaai/api/assistant-api/internal/adapters"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

// CallReciever handles incoming calls for the given assistant.
// Thin controller — business logic delegated to pipeline's handleCallReceived.
func (cApi *ConversationApi) CallReciever(c *gin.Context) {
	provider := c.Param("telephony")
	if c.Query("CustomField") != "" {
		callInfo, err := cApi.inboundDispatcher.ReceiveCall(c, provider)
		if err != nil {
			cApi.logger.Errorf("failed to handle provider answer hook: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to answer call"})
			return
		}
		if callInfo == nil {
			return
		}
	}

	auth, authErr := types.Authorize(c.Request.Context())
	if authErr != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": authErr.Error()})
		return
	}
	iAuth, scopeErr := auth.Scope(types.AuthTypeUser, types.AuthTypeProject)
	if scopeErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": scopeErr.Error()})
		return
	}

	assistantID := c.Param("assistantId")
	if assistantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return
	}
	assistantId, err := utils.StringToUint64(assistantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistant ID"})
		return
	}

	observer := cApi.Observability(c, iAuth, observability.WithGracePeriod())
	defer observer.Close(context.Background())

	// Pipeline handles: create conversation, answer provider, emit events
	result := cApi.channelPipeline.Run(c, channel_pipeline.CallReceivedPipeline{
		ID:          uuid.NewString(),
		Provider:    provider,
		Auth:        iAuth,
		AssistantID: assistantId,
		GinContext:  c,
		Observer:    observer,
	})
	if result.Error != nil {
		cApi.logger.Errorf("failed to handle inbound call: %v", result.Error)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to initiate talker"})
	}
}

// CallAnswerByContext returns the <Stream> answer XML for a resolved call context.
// A provider's answer_url fetches this to open the bidirectional media stream.
// TODO(plivo): move answer-XML generation onto the Telephony interface (inlined here for now).
func (cApi *ConversationApi) CallAnswerByContext(c *gin.Context) {
	contextID := c.Param("contextId")
	if !validator.NotBlank(contextID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing contextId"})
		return
	}

	cc, err := cApi.callContextStore.Get(c, contextID)
	if err != nil {
		cApi.logger.Errorf("failed to resolve call context %s for answer: %v", contextID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid context to answer"})
		return
	}

	provider := c.Param("telephony")
	if !validator.NotBlank(provider) {
		provider = cc.Provider
	}

	public := cApi.cfg.Assistant.Public
	wsURL := fmt.Sprintf("wss://%s/%s", public, internal_type.GetContextAnswerPath(provider, cc.ContextID))
	eventURL := fmt.Sprintf("https://%s/%s", public, internal_type.GetContextEventPath(provider, cc.ContextID))

	answerXML := fmt.Sprintf(`<Response>
	<Stream bidirectional="true" keepCallAlive="true" contentType="audio/x-mulaw;rate=8000" statusCallbackUrl="%s" statusCallbackMethod="POST">%s</Stream>
</Response>`, eventURL, wsURL)

	c.Data(http.StatusOK, "text/xml", []byte(answerXML))
}

// CallTalkerByContext handles WebSocket connections for media streaming.
// Thin controller — pipeline handles: resolve context, create streamer/talker, Talk(), cleanup.
// contextId is read from URL path (:contextId) or query param (?contextId=).
// Query param fallback supports Asterisk chan_websocket which appends params via v() dialstring option.
func (cApi *ConversationApi) CallTalkerByContext(c *gin.Context) {
	upgrader := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool { return true }}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to upgrade connection"})
		return
	}

	contextID := c.Query("contextId")
	if contextID == "" {
		contextID = c.Param("contextId")
	}
	if contextID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing contextId"})
		return
	}

	callContext, vaultCredential, err := cApi.inboundDispatcher.ResolveCallSessionByContext(c, contextID)
	if err != nil {
		cApi.logger.Errorf("failed to resolve call context %s: %v", contextID, err)
		return
	}
	auth, err := callContext.ToAuthentication()
	if err != nil {
		cApi.logger.Errorf("failed to reconstruct call authentication: %v", err)
		return
	}
	observer := cApi.Observability(c, auth, observability.WithGracePeriod())
	defer observer.Close(context.Background())

	streamer, err := channel_telephony.Telephony(callContext.Provider).NewStreamer(
		cApi.logger,
		callContext,
		vaultCredential,
		channel_telephony.WithWebSocketStreamer(ws),
		channel_telephony.WithObserver(observer),
	)
	if err != nil {
		cApi.logger.Errorf("failed to create streamer for context %s: %v", contextID, err)
		return
	}
	talker, err := internal_adapter.New(
		internal_adapter.WithSource(utils.PhoneCall),
		internal_adapter.WithContext(c),
		internal_adapter.WithConfig(cApi.cfg),
		internal_adapter.WithLogger(cApi.logger),
		internal_adapter.WithPostgres(cApi.postgres),
		internal_adapter.WithOpenSearch(cApi.opensearch),
		internal_adapter.WithRedis(cApi.redis),
		internal_adapter.WithStorage(cApi.storage),
		internal_adapter.WithStreamer(streamer),
		internal_adapter.WithObserver(observer),
	)
	if err != nil {
		cApi.logger.Errorf("failed to create talker for context %s: %v", contextID, err)
		return
	}

	result := cApi.channelPipeline.Run(c, channel_pipeline.SessionConnectedPipeline{
		ID:          contextID,
		ContextID:   contextID,
		CallContext: callContext,
		Talker:      talker,
		Observer:    observer,
	})
	if result != nil && result.Error != nil {
		cApi.logger.Errorf("talk failed for context %s: %v", contextID, result.Error)
	}
}
