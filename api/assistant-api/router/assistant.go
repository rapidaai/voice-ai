package assistant_router

import (
	"github.com/gin-gonic/gin"
	assistantApi "github.com/rapidaai/api/assistant-api/api/assistant"
	assistantDeploymentApi "github.com/rapidaai/api/assistant-api/api/assistant-deployment"
	assistantTalkApi "github.com/rapidaai/api/assistant-api/api/talk"
	"github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	workflow_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

func AssistantApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
) {
	workflow_api.RegisterAssistantServiceServer(S,
		assistantApi.NewAssistantGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			Opensearch,
		))
}

func AssistantDeploymentApiRoute(Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector) {
	workflow_api.RegisterAssistantDeploymentServiceServer(S,
		assistantDeploymentApi.NewAssistantDeploymentGRPCApi(Cfg,
			Logger,
			Postgres,
		))
}

func AssistantConversationApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
) {
	workflow_api.RegisterTalkServiceServer(S,
		assistantTalkApi.NewConversationGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			Opensearch,
		))
}

func TalkCallbackApiRoute(
	cfg *config.AssistantConfig, engine *gin.Engine, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector) {
	apiv1 := engine.Group("v1/talk")
	talkRpcApi := assistantTalkApi.NewConversationApi(cfg,
		logger,
		postgres,
		redis,
		opensearch,
		opensearch,
	)
	{
		// exotel call
		apiv1.GET("/exotel/call/:assistantId", talkRpcApi.ExotelCallReciever)
		apiv1.GET("/exotel/usr/:assistantId/:identifier/:conversationId/:authorization/:x-auth-id/:x-project-id", talkRpcApi.ExotelCallTalker)
		apiv1.GET("/exotel/prj/:assistantId/:identifier/:conversationId/:x-api-key", talkRpcApi.ExotelCallTalker)

		// twillio call
		apiv1.GET("/twilio/call/:assistantId", talkRpcApi.PhoneCallReciever)
		apiv1.GET("/twilio/usr/:assistantId/:identifier/:conversationId/:authorization/:x-auth-id/:x-project-id", talkRpcApi.TwilioCallTalker)
		apiv1.GET("/twilio/prj/:assistantId/:identifier/:conversationId/:x-api-key", talkRpcApi.TwilioCallTalker)

		// twilio whatsapp
		apiv1.POST("/twilio/whatsapp/:assistantToken", talkRpcApi.WhatsappReciever)

		// vonage call
		apiv1.GET("/vonage/call/:assistantId", talkRpcApi.PhoneCallReciever)
		apiv1.GET("/vonage/usr/:assistantId/:identifier/:conversationId/:authorization/:x-auth-id/:x-project-id", talkRpcApi.VonageCallTalker)
		apiv1.GET("/vonage/prj/:assistantId/:identifier/:conversationId/:x-api-key", talkRpcApi.VonageCallTalker)

	}
}

func ConversationApiRoute(
	cfg *config.AssistantConfig, engine *gin.Engine, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector) {
	apiv1 := engine.Group("v1/conversation")
	talkRpcApi := assistantTalkApi.NewConversationApi(cfg,
		logger,
		postgres,
		redis,
		opensearch,
		opensearch,
	)
	apiv1.POST("/create-phone-call", talkRpcApi.ExotelCallReciever)
	apiv1.POST("/create-bulk-phone-call", talkRpcApi.ExotelCallReciever)

}
