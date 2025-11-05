package assistant_talk_api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	internal_factories "github.com/rapidaai/api/assistant-api/internal/factories"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

// CallReciever handles incoming calls for the given assistant.
// @Router /v1/call/:assistantId [post]
// @Summary Recieve call for given assistant
// @Produce json
// @Success 200 {object} app.Response
// @Failure 500 {object} app.Response
func (cApi *ConversationApi) ExotelCallReciever(c *gin.Context) {
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

	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	toNumber, ok := queryParams["CallFrom"]
	if !ok {
		cApi.logger.Debugf("Missing 'Caller' number in Twilio request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'Caller' number"})
		return
	}

	assistant, err := cApi.assistantService.Get(c,
		iAuth,
		assistantId,
		utils.GetVersionDefinition("latest"),
		&internal_services.GetAssistantOption{
			InjectPhoneDeployment: true,
		})
	if err != nil {
		cApi.logger.Debugf("illegal unable to find assistant %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to initiate talker"})
		return
	}

	conversation, err := cApi.assistantConversationService.CreateConversation(
		c,
		iAuth,
		internal_factories.Identifier(utils.PhoneCall, c, iAuth, toNumber),
		assistant.Id,
		assistant.AssistantProviderId,
		type_enums.DIRECTION_INBOUND,
		utils.PhoneCall,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to initiate talker"})
		return
	}

	cApi.
		assistantConversationService.
		ApplyConversationMetrics(
			c, iAuth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_CONNECTED)},
		)

	// this is temp
	redirectUrl := "assistant-01.rapida.ai"
	// redirectUrl = "integral-presently-cub.ngrok-free.app"
	response := map[string]string{
		"url": fmt.Sprintf("wss://%s/v1/talk/prj/stream/%s/%s/%d/%s",
			redirectUrl, assistantIdStr, toNumber, conversation.Id, iAuth.GetCurrentToken()),
	}

	c.JSON(http.StatusOK, response)
}

func (cApi *ConversationApi) ExotelCallTalker(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		cApi.logger.Errorf("illegal unable to authenticate")
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthenticated request"})
		return
	}

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
	websocketConnection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		cApi.logger.Errorf("illegal while upgrading twillio talker with error %v", err)
		return
	}

	cApi.logger.Debugf("starting a call talker for twillio with %s and params %+v", identifier, c.Params)
	talker, err := internal_factories.GetTalker(
		utils.PhoneCall,
		c,
		cApi.cfg,
		cApi.logger,
		cApi.postgres,
		cApi.opensearch,
		cApi.redis,
		cApi.storage,
		internal_adapter_request_streamers.NewExotelWebsocketStreamer(
			cApi.logger,
			websocketConnection,
			assistantId,
			"latest",
			conversationId,
		),
	)
	if err != nil {
		cApi.logger.Errorf("illegal to get talker %v", err)
		return
	}
	cIdentifier := internal_factories.Identifier(utils.PhoneCall, c, auth, identifier)
	err = talker.Connect(
		c, auth, cIdentifier, &lexatic_backend.AssistantConversationConfiguration{
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
		cIdentifier,
	)
}
