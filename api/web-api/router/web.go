package web_router

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	webApi "github.com/rapidaai/api/web-api/api"
	webProxyApi "github.com/rapidaai/api/web-api/api/proxy"
	"github.com/rapidaai/api/web-api/config"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
)

func WebApiRoute(
	Cfg *config.WebAppConfig,
	organizationCredentialKey []byte,
	E *gin.Engine,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
) {
	apiv1 := E.Group("/v1")
	apiv1.POST("/auth/authenticate/", webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres).Authenticate)
	apiv1.POST("/auth/register-user/", webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres).RegisterUser)

	//
	apiOauth := E.Group("/oauth")
	auth := webApi.NewAuthRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres)
	apiOauth.GET("/google/", auth.Google)
	apiOauth.GET("/linkedin/", auth.Linkedin)
	apiOauth.GET("/github/", auth.Github)

	//

	connectApi := webApi.NewConnectRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres)
	apiv1.GET("/connect-knowledge/notion/", connectApi.NotionConnect)
	apiv1.GET("/connect-knowledge/confluence/", connectApi.ConfluenceConnect)
	apiv1.GET("/connect-knowledge/google-drive/", connectApi.GoogleDriveConnect)
	apiv1.GET("/connect-knowledge/github/", connectApi.GithubCodeConnect)
	apiv1.GET("/connect-knowledge/gitlab/", connectApi.GitlabCodeConnect)
	apiv1.GET("/connect-knowledge/microsoft-onedrive/", connectApi.MicrosoftOnedriveConnect)
	apiv1.GET("/connect-knowledge/sharepoint/", connectApi.MicrosoftSharepointConnect)
	apiv1.GET("/connect-action/gmail/", connectApi.GmailActionConnect)
	apiv1.GET("/connect-action/jira/", connectApi.JiraActionConnect)
	apiv1.GET("/connect-action/slack/", connectApi.SlackActionConnect)
	apiv1.GET("/connect-crm/hubspot/", connectApi.HubspotCRMConnect)

	protos.RegisterAuthenticationServiceServer(S, webApi.NewAuthGRPC(Cfg, &Cfg.OAuthConfig, organizationCredentialKey, Logger, Postgres))
	protos.RegisterVaultServiceServer(S, webApi.NewVaultGRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres, Redis))
	protos.RegisterOrganizationServiceServer(S, webApi.NewOrganizationGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterProjectServiceServer(S, webApi.NewProjectGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterConnectServiceServer(S, webApi.NewConnectGRPC(Cfg, &Cfg.OAuthConfig, Logger, Postgres))
	protos.RegisterNotificationServiceServer(S, webApi.NewNotificationGRPC(Cfg, Logger, Postgres, Redis))
	protos.RegisterProductUsageServiceServer(S, webApi.NewProductUsageGRPC(Logger, Postgres))
	protos.RegisterAccessManagementServiceServer(S, webApi.NewAccessManagementGRPC(Logger, Postgres))
	protos.RegisterDashboardServiceServer(S, webApi.NewDashboardGRPC(Cfg, Logger, Postgres, Redis))
}

func ProxyApiRoute(Cfg *config.WebAppConfig,
	S *grpc.Server,
	Logger commons.Logger,
	Postgres connectors.PostgresConnector,
	Redis connectors.RedisConnector,
	Clients *rapida_client.RapidaClient) {
	protos.RegisterDeploymentServer(S, webProxyApi.NewInvokeGRPC(Cfg, Logger, Postgres, Redis, Clients.Deployment))
	protos.RegisterAuditLoggingServiceServer(S, webProxyApi.NewActivityGRPC(Cfg, Logger, Postgres, Redis, Clients.Audit))
	protos.RegisterEndpointServiceServer(S, webProxyApi.NewEndpointGRPC(Cfg, Logger, Postgres, Redis, Clients.Endpoint))
	protos.RegisterKnowledgeServiceServer(S, webProxyApi.NewKnowledgeGRPC(Cfg, Logger, Postgres, Redis, Clients.Knowledge))
	protos.RegisterAssistantServiceServer(S, webProxyApi.NewAssistantGRPC(Cfg, Logger, Postgres, Redis, Clients.Assistant))
	protos.RegisterAssistantDeploymentServiceServer(S, webProxyApi.NewAssistantDeploymentGRPCApi(Cfg, Logger, Postgres, Redis, Clients.Assistant))
	protos.RegisterObservabilityServiceServer(S, webProxyApi.NewObservabilityGRPC(Cfg, Logger, Postgres, Redis, Clients.Observability))
	protos.RegisterDocumentServiceServer(S, webProxyApi.NewDocumentGRPCApi(Cfg, Logger, Postgres, Redis, Clients.Indexer))
}
