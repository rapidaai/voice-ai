package web_router

import (
	"github.com/gin-gonic/gin"
	webApi "github.com/rapidaai/api/web-api/api"
	webProxyApi "github.com/rapidaai/api/web-api/api/proxy"
	"github.com/rapidaai/api/web-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

func WebApiRoute(
	Cfg *config.WebAppConfig,
	E *gin.Engine,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
) {
	apiv1 := E.Group("/v1")
	{
		apiv1.POST("/auth/authenticate/", webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres).Authenticate)
		apiv1.POST("/auth/register-user/", webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres).RegisterUser)

	}
	protos.RegisterAuthenticationServiceServer(S, webApi.NewAuthGRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres))
	apiOauth := E.Group("/oauth")
	auth := webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres)
	{
		apiOauth.GET("/google/", auth.Google)
		apiOauth.GET("/linkedin/", auth.Linkedin)
		apiOauth.GET("/github/", auth.Github)
	}
	protos.RegisterVaultServiceServer(S, webApi.NewVaultGRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres, Redis))
	protos.RegisterOrganizationServiceServer(S, webApi.NewOrganizationGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterProjectServiceServer(S, webApi.NewProjectGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterLeadGeneratorServiceServer(S, webApi.NewLeadGRPC(Cfg, Logger, Postgres, Redis))

	protos.RegisterConnectServiceServer(S, webApi.NewConnectGRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres))
	Logger.Info("Internal HealthCheckRoutes and Connectors added to engine.")

	//
	connectKnowledgeApi := E.Group("/connect-knowledge")
	connectApi := webApi.NewConnectRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres)
	{
		// working
		connectKnowledgeApi.GET("/notion/", connectApi.NotionConnect)

		connectKnowledgeApi.GET("/confluence/", connectApi.ConfluenceConnect)
		connectKnowledgeApi.GET("/google-drive/", connectApi.GoogleDriveConnect)
		//
		connectKnowledgeApi.GET("/github/", connectApi.GithubCodeConnect)
		connectKnowledgeApi.GET("/gitlab/", connectApi.GitlabCodeConnect)

		connectKnowledgeApi.GET("/microsoft-onedrive/", connectApi.MicrosoftOnedriveConnect)
		connectKnowledgeApi.GET("/sharepoint/", connectApi.MicrosoftSharepointConnect)
	}

	actionApiv1 := E.Group("/connect-action")
	{
		actionApiv1.GET("/gmail/", connectApi.GmailActionConnect)
		actionApiv1.GET("/jira/", connectApi.JiraActionConnect)
		actionApiv1.GET("/slack/", connectApi.SlackActionConnect)
	}

	crmConnectApiv1 := E.Group("/connect-crm")
	{
		crmConnectApiv1.GET("/hubspot/", connectApi.HubspotCRMConnect)
	}

}

func ProxyApiRoute(Cfg *config.WebAppConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector) {
	protos.RegisterDeploymentServer(S, webProxyApi.NewInvokeGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterAuditLoggingServiceServer(S, webProxyApi.NewActivityGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterEndpointServiceServer(S, webProxyApi.NewEndpointGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterKnowledgeServiceServer(S, webProxyApi.NewKnowledgeGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterAssistantServiceServer(S, webProxyApi.NewAssistantGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterAssistantDeploymentServiceServer(S, webProxyApi.NewAssistantDeploymentGRPCApi(Cfg, Logger, Postgres, Redis))
	protos.RegisterDocumentServiceServer(S, webProxyApi.NewDocumentGRPCApi(Cfg, Logger, Postgres, Redis))

}
