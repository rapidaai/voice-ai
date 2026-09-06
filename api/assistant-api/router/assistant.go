// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_router

import (
	"github.com/gin-gonic/gin"
	assistantApi "github.com/rapidaai/api/assistant-api/api/assistant"
	assistantDeploymentApi "github.com/rapidaai/api/assistant-api/api/assistant-deployment"
	observabilityApi "github.com/rapidaai/api/assistant-api/api/observability"
	assistantTalkApi "github.com/rapidaai/api/assistant-api/api/talk"
	"github.com/rapidaai/api/assistant-api/config"
	sip_runtime "github.com/rapidaai/api/assistant-api/sip/runtime"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	workflow_api "github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

func AssistantApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	engine *gin.Engine,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
) {
	assistantServiceServer := assistantApi.NewAssistantGRPCApi(Cfg,
		Logger,
		Postgres,
		Redis,
		Opensearch,
		Opensearch,
	)
	workflow_api.RegisterAssistantServiceServer(S, assistantServiceServer)
	workflow_api.RegisterObservabilityServiceServer(S,
		observabilityApi.NewObservabilityGRPCApi(Cfg,
			Logger,
			Opensearch,
		))

	apiv1 := engine.Group("v1/assistant")
	createAssistantRestHandler := assistantServiceServer.(interface {
		CreateAssistantRest(*gin.Context)
		CreateAssistantConfigurationRest(*gin.Context)
		UpdateAssistantConfigurationRest(*gin.Context)
		GetAssistantConfigurationRest(*gin.Context)
		GetAllAssistantConfigurationRest(*gin.Context)
		DeleteAssistantConfigurationRest(*gin.Context)
	})
	apiv1.POST("/create-assistant", createAssistantRestHandler.CreateAssistantRest)
	apiv1.POST("/configurations", createAssistantRestHandler.CreateAssistantConfigurationRest)
	apiv1.GET("/configurations/:assistantId", createAssistantRestHandler.GetAllAssistantConfigurationRest)
	apiv1.GET("/configurations/:assistantId/:id", createAssistantRestHandler.GetAssistantConfigurationRest)
	apiv1.PATCH("/configurations/:assistantId/:id", createAssistantRestHandler.UpdateAssistantConfigurationRest)
	apiv1.DELETE("/configurations/:assistantId/:id", createAssistantRestHandler.DeleteAssistantConfigurationRest)
}

func AssistantDeploymentApiRoute(Cfg *config.AssistantConfig,
	S *grpc.Server,
	engine *gin.Engine,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector) {
	workflow_api.RegisterAssistantDeploymentServiceServer(S,
		assistantDeploymentApi.NewAssistantDeploymentGRPCApi(Cfg,
			Logger,
			Postgres,
		))

	apiv1 := engine.Group("v1/assistant-deployment")
	deploymentApi := assistantDeploymentApi.NewAssistantDeploymentApi(Cfg, Logger, Postgres)
	apiv1.POST("/create-api-deployment", deploymentApi.CreateAssistantApiDeploymentRest)
	apiv1.POST("/create-debugger-deployment", deploymentApi.CreateAssistantDebuggerDeploymentRest)
	apiv1.POST("/create-phone-deployment", deploymentApi.CreateAssistantPhoneDeploymentRest)
	apiv1.POST("/create-webplugin-deployment", deploymentApi.CreateAssistantWebpluginDeploymentRest)
	apiv1.POST("/create-whatsapp-deployment", deploymentApi.CreateAssistantWhatsappDeploymentRest)
	apiv1.GET("/get-api-deployment/:assistantId", deploymentApi.GetAssistantApiDeploymentRest)
	apiv1.GET("/get-debugger-deployment/:assistantId", deploymentApi.GetAssistantDebuggerDeploymentRest)
	apiv1.GET("/get-phone-deployment/:assistantId", deploymentApi.GetAssistantPhoneDeploymentRest)
	apiv1.GET("/get-webplugin-deployment/:assistantId", deploymentApi.GetAssistantWebpluginDeploymentRest)
	apiv1.GET("/get-whatsapp-deployment/:assistantId", deploymentApi.GetAssistantWhatsappDeploymentRest)
	apiv1.GET("/get-all-api-deployment/:assistantId", deploymentApi.GetAllAssistantApiDeploymentRest)
	apiv1.GET("/get-all-debugger-deployment/:assistantId", deploymentApi.GetAllAssistantDebuggerDeploymentRest)
	apiv1.GET("/get-all-phone-deployment/:assistantId", deploymentApi.GetAllAssistantPhoneDeploymentRest)
	apiv1.GET("/get-all-webplugin-deployment/:assistantId", deploymentApi.GetAllAssistantWebpluginDeploymentRest)
	apiv1.GET("/get-all-whatsapp-deployment/:assistantId", deploymentApi.GetAllAssistantWhatsappDeploymentRest)
}

func AssistantConversationApiRoute(
	Cfg *config.AssistantConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Opensearch connectors.OpenSearchConnector,
	sipServer *sip_runtime.Server,
	rapidaClient *rapida_client.RapidaClient,
) {
	workflow_api.RegisterTalkServiceServer(S,
		assistantTalkApi.NewConversationGRPCApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			Opensearch,
			sipServer,
			rapidaClient,
		))
	workflow_api.RegisterWebRTCServer(S,
		assistantTalkApi.NewWebRtcApi(Cfg,
			Logger,
			Postgres,
			Redis,
			Opensearch,
			Opensearch,
			sipServer,
			rapidaClient,
		))
}

func TalkApiRoute(
	cfg *config.AssistantConfig, engine *gin.Engine, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	sipServer *sip_runtime.Server,
	rapidaClient *rapida_client.RapidaClient,
) {
	apiv1 := engine.Group("v1/talk")
	talkRpcApi := assistantTalkApi.NewConversationApi(cfg, logger, postgres, redis, opensearch, opensearch, sipServer, rapidaClient)
	{
		apiv1.POST("/create-phone-call", talkRpcApi.CreatePhoneCallRest)
		apiv1.POST("/create-bulk-phone-call", talkRpcApi.CreateBulkPhoneCallRest)

		// global catch-all event logging
		apiv1.GET("/:telephony/event/:assistantId", talkRpcApi.UnviersalCallback)
		apiv1.POST("/:telephony/event/:assistantId", talkRpcApi.UnviersalCallback)

		// inbound call receiver — webhook from telephony provider, saves call context to Postgres
		// Both GET and POST: Twilio defaults to POST, Vonage/Exotel use GET, Asterisk uses GET.
		apiv1.GET("/:telephony/call/:assistantId", talkRpcApi.CallReciever)
		apiv1.POST("/:telephony/call/:assistantId", talkRpcApi.CallReciever)

		// contextId-based routes — all auth, assistant, conversation resolved from Postgres call context
		// Used by all telephony providers (Twilio, Exotel, Vonage, Asterisk, SIP)
		apiv1.GET("/:telephony/ctx/:contextId", talkRpcApi.CallTalkerByContext)
		apiv1.GET("/:telephony/ctx/:contextId/event", talkRpcApi.CallbackByContext)
		apiv1.POST("/:telephony/ctx/:contextId/event", talkRpcApi.CallbackByContext)
	}
}
