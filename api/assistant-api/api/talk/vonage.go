package assistant_talk_api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	internal_factories "github.com/rapidaai/api/assistant-api/internal/factories"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

func (cApi *ConversationApi) VonageCallTalker(c *gin.Context) {
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
	cApi.logger.Debugf("starting a call talker for vonage with %s and params %+v", identifier, c.Params)
	vonageStreamer := internal_adapter_request_streamers.
		NewVonageWebsocketStreamer(
			cApi.logger,
			websocketConnection,
			assistantId, "latest", conversationId,
		)

	talker, err := internal_factories.GetTalker(
		utils.PhoneCall,
		c,
		cApi.cfg,
		cApi.logger,
		cApi.postgres,
		cApi.opensearch,
		cApi.redis,
		cApi.storage,
		vonageStreamer,
	)
	if err != nil {
		cApi.logger.Errorf("illegal to get talker %v", err)
		return
	}
	cApi.logger.Benchmark("conversationapi.VonageCallTalker.GetTalker", time.Since(start))
	cidentifier := internal_factories.
		Identifier(utils.PhoneCall, c, auth, identifier)

	err = talker.Connect(
		c, auth, cidentifier, &lexatic_backend.AssistantConversationConfiguration{
			AssistantConversationId: conversationId,
			Assistant: &lexatic_backend.AssistantDefinition{
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
