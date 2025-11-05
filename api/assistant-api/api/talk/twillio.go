package assistant_talk_api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	internal_adapter_request_streamers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/streamers"
	internal_factories "github.com/rapidaai/api/assistant-api/internal/factories"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_telephony "github.com/rapidaai/api/assistant-api/internal/telephony"
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
func (cApi *ConversationApi) PhoneCallReciever(c *gin.Context) {
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

	cApi.logger.Debugf("%+v", queryParams)
	clientNumber, ok := queryParams["Caller"]
	if !ok {
		cApi.logger.Debugf("Missing 'Caller' number in Twilio request")
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
			internal_factories.Identifier(utils.PhoneCall, c, iAuth, clientNumber),
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
			c, iAuth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_CONNECTED)},
		)

	c.Data(http.StatusOK, "text/xml", []byte(
		internal_telephony.CreateTwinML(
			fmt.Sprintf("v1/talk/twilio/prj/%d/%s/%d/%s",
				assistantId,
				clientNumber, conversation.Id, iAuth.GetCurrentToken()), assistantId, clientNumber),
	))
}

func (cApi *ConversationApi) TwilioCallTalker(c *gin.Context) {
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
		cApi.logger.Errorf("illegal while upgrading twillio talker with error %v", err)
		return
	}
	cApi.logger.Benchmark("ConversationApi.TwilioCallTalker.upgradeConnection", time.Since(start))

	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		cApi.logger.Errorf("illegal unable to authenticate")
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthenticated request"})
		return
	}

	cApi.logger.Benchmark("conversationapi.TwilioCallTalker.GetAuthPrinciple", time.Since(start))
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
	cApi.logger.Debugf("starting a call talker for twillio with %s and params %+v", identifier, c.Params)

	twilioStreamer := internal_adapter_request_streamers.NewTwilioWebsocketStreamer(
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
		twilioStreamer,
	)
	if err != nil {
		cApi.logger.Errorf("illegal to get talker %v", err)
		return
	}
	cApi.logger.Benchmark("conversationapi.TwilioCallTalker.GetTalker", time.Since(start))
	cidentifier := internal_factories.Identifier(utils.PhoneCall, c, auth, identifier)
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
