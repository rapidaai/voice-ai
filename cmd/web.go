package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	healthCheckApi "github.com/rapidaai/api/health-check-api"
	web_authenticators "github.com/rapidaai/api/web-api/auth"
	webApi "github.com/rapidaai/api/web-api/handler"
	config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authenticators"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middlewares "github.com/rapidaai/pkg/middlewares"
	web_api "github.com/rapidaai/protos"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// wrapper for gin engine
type AppRunner struct {
	E         *gin.Engine
	S         *grpc.Server
	Cfg       *config.WebAppConfig
	Logger    commons.Logger
	Postgres  connectors.PostgresConnector
	Redis     connectors.RedisConnector
	Closeable []func(context.Context) error
}

func main() {
	// creating a common context
	ctx := context.Background()

	appRunner := AppRunner{E: gin.New()}
	// resolving configuration
	err := appRunner.ResolveConfig()
	if err != nil {
		panic(err)
	}
	// logging
	appRunner.Logging()

	// adding all connectors
	appRunner.AllConnectors()
	// init
	appRunner.S = grpc.NewServer(
		grpc.ChainStreamInterceptor(
			middlewares.NewRequestLoggerStreamServerMiddleware(appRunner.Cfg.Name, appRunner.Logger),
			middlewares.NewAuthenticationStreamServerMiddleware(
				web_authenticators.GetUserAuthenticator(appRunner.Logger, appRunner.Postgres),
				appRunner.Logger),
			middlewares.NewProjectAuthenticatorStreamServerMiddleware(web_authenticators.GetProjectAuthenticator(appRunner.Logger, appRunner.Postgres),
				appRunner.Logger),
		),
		grpc.ChainUnaryInterceptor(
			middlewares.NewRequestLoggerUnaryServerMiddleware(appRunner.Cfg.Name, appRunner.Logger),
			middlewares.NewAuthenticationUnaryServerMiddleware(web_authenticators.GetUserAuthenticator(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			middlewares.NewProjectAuthenticatorUnaryServerMiddleware(
				web_authenticators.GetProjectAuthenticator(appRunner.Logger, appRunner.Postgres),
				appRunner.Logger,
			),
			middlewares.NewServiceAuthenticatorUnaryServerMiddleware(
				authenticators.NewServiceAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Postgres),
				appRunner.Logger,
			),
		),
		grpc.MaxRecvMsgSize(commons.MaxRecvMsgSize), // 10 MB
		grpc.MaxSendMsgSize(commons.MaxSendMsgSize), // 10 MB
	)

	err = appRunner.Init(ctx)

	if err != nil {
		panic(err)
	}

	// add all middleware depends on configurations
	appRunner.AllMiddlewares()

	// all routers add all handlers which required to resolve the service request
	appRunner.AllRouters()
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", appRunner.Cfg.Host, appRunner.Cfg.Port))
	if err != nil {
		log.Fatalf("Failed to create connection tcp %v", err)
		panic(err)
	}

	defer appRunner.Close(ctx)
	cmuxListener := cmux.New(listener)

	// if application json
	http2GRPCFilteredListener := cmuxListener.Match(cmux.HTTP2())
	grpcFilteredListener := cmuxListener.Match(
		cmux.HTTP1HeaderField("content-type", "application/grpc-web+proto"),
		cmux.HTTP1HeaderField("x-grpc-web", "1"))
	rpcFilteredListener := cmuxListener.Match(cmux.Any())
	// rpcFilteredListener := cmuxListener.Match(cmux.HTTP2())
	// grpcFilteredListener := cmuxListener.Match(cmux.Any())

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err = appRunner.E.RunListener(rpcFilteredListener)
		if err != nil {
			appRunner.Logger.Errorf("Failed to start gin server err: %v", err)
		}
		return err
	})
	group.Go(func() error {
		//
		wrappedServer := grpcweb.WrapServer(appRunner.S, grpcweb.WithOriginFunc(func(origin string) bool { return true }))
		handler := func(resp http.ResponseWriter, req *http.Request) {
			wrappedServer.ServeHTTP(resp, req)
		}

		httpServer := http.Server{
			Handler: http.HandlerFunc(handler),
		}
		//
		err = httpServer.Serve(grpcFilteredListener)
		if err != nil {
			appRunner.Logger.Errorf("Failed to start grpc server err: %v", err)
		}
		return err

	})

	group.Go(func() error {
		err = appRunner.S.Serve(http2GRPCFilteredListener)
		if err != nil {
			appRunner.Logger.Errorf("Failed to start grpc server err: %v", err)
		}
		return err
	})
	//serve now
	err = cmuxListener.Serve()
	if err != nil {
		appRunner.Logger.Errorf("Failed to start grpc server err: %v", err)
		panic(err)
	}

	err = group.Wait()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	// done with ctx
	ctx.Done()
	<-quit
}

func (app *AppRunner) Logging() {
	aLogger := commons.NewApplicationLoggerWithOptions(
		commons.Level(app.Cfg.LogLevel),
		commons.Name(app.Cfg.Name),
	)
	aLogger.InitLogger()
	aLogger.Info("Added Logger middleware to the application.")
	app.Logger = aLogger
}

func (g *AppRunner) AllConnectors() {
	postgres := connectors.NewPostgresConnector(&g.Cfg.PostgresConfig, g.Logger)
	redis := connectors.NewRedisConnector(&g.Cfg.RedisConfig, g.Logger)
	g.Postgres = postgres
	g.Redis = redis
}

// initialize the config of application using viper and return loaded appconfig to be used in
// if any error return or nil.
func (app *AppRunner) ResolveConfig() error {
	vConfig, err := config.InitConfig()
	if err != nil {
		log.Fatalf("Unable to parse viper config to application configuration : %v", err)
		return err
	}

	cfg, err := config.GetApplicationConfig(vConfig)
	if err != nil {
		log.Fatalf("Unable to parse viper config to application configuration : %v", err)
		return err
	}

	app.Cfg = cfg
	gin.SetMode(gin.ReleaseMode)
	// debug mode of gin when runing log in debug mode.
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	}
	return nil

}

// init for app close
func (app *AppRunner) Init(ctx context.Context) error {
	err := app.Postgres.Connect(ctx)
	if err != nil {
		app.Logger.Error("error while connecting to postgres.", err)
		return err
	}
	err = app.Redis.Connect(ctx)
	if err != nil {
		app.Logger.Error("error while connecting to redis.", err)
		return err
	}
	app.Closeable = append(app.Closeable, app.Postgres.Disconnect)
	app.Closeable = append(app.Closeable, app.Redis.Disconnect)
	return nil
}

// closer for app runner
func (app *AppRunner) Close(ctx context.Context) {
	if len(app.Closeable) > 0 {
		app.Logger.Debug("there are closeable references to closed")
		for _, closeable := range app.Closeable {
			err := closeable(ctx)
			if err != nil {
				app.Logger.Errorf("error while closing %v", err)
			}
		}
	}
}

// all router initialize
func (g *AppRunner) AllRouters() {
	g.HealthCheckRoutes()
	g.AuthApiRoutes()
	g.OauthApiRoute()
	g.VaultApiRoute()
	g.OrganizationApiRoute()
	g.ProjectApiRoute()
	g.ActivityApiRoute()
	g.EndpointApiRoute()
	g.InvokeApiRoute()
	g.KnowledgeApiRoute()
	g.AssistantApiRoute()
	g.DocumentApiRoute()
	g.ProviderApiRoute()
	g.KnowledgeConnectApiRoute()
	g.LeadApiRoute()

}

// all middleware
func (g *AppRunner) AllMiddlewares() {
	g.LoggerMiddleware()
	g.RecoveryMiddleware()
	g.CorsMiddleware()
	g.RequestLoggerMiddleware()
	g.E.Use(middlewares.NewAuthenticationMiddleware(web_authenticators.GetUserAuthenticator(g.Logger, g.Postgres), g.Logger))
}

func (g *AppRunner) AuthApiRoutes() {
	apiv1 := g.E.Group("/v1")
	{
		apiv1.POST("/auth/authenticate/", webApi.NewAuthRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres).Authenticate)
		apiv1.POST("/auth/register-user/", webApi.NewAuthRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres).RegisterUser)

	}
	web_api.RegisterAuthenticationServiceServer(g.S, webApi.NewAuthGRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres))

}

func (g *AppRunner) OauthApiRoute() {
	apiOauth := g.E.Group("/oauth")
	auth := webApi.NewAuthRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres)
	{
		apiOauth.GET("/google/", auth.Google)
		apiOauth.GET("/linkedin/", auth.Linkedin)
		apiOauth.GET("/github/", auth.Github)
	}
}

func (g *AppRunner) VaultApiRoute() {
	web_api.RegisterVaultServiceServer(g.S, webApi.NewVaultGRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) InvokeApiRoute() {
	web_api.RegisterDeploymentServer(g.S, webApi.NewInvokeGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) OrganizationApiRoute() {
	web_api.RegisterOrganizationServiceServer(g.S, webApi.NewOrganizationGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) ProjectApiRoute() {
	web_api.RegisterProjectServiceServer(g.S, webApi.NewProjectGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) ActivityApiRoute() {
	web_api.RegisterAuditLoggingServiceServer(g.S, webApi.NewActivityGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) EndpointApiRoute() {
	web_api.RegisterEndpointServiceServer(g.S, webApi.NewEndpointGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) KnowledgeApiRoute() {
	web_api.RegisterKnowledgeServiceServer(g.S, webApi.NewKnowledgeGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}
func (g *AppRunner) AssistantApiRoute() {
	web_api.RegisterAssistantServiceServer(g.S, webApi.NewAssistantGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
	web_api.RegisterAssistantDeploymentServiceServer(g.S, webApi.NewAssistantDeploymentGRPCApi(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) DocumentApiRoute() {
	web_api.RegisterDocumentServiceServer(g.S, webApi.NewDocumentGRPCApi(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) ProviderApiRoute() {
	web_api.RegisterProviderServiceServer(g.S, webApi.NewProviderGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) LeadApiRoute() {
	web_api.RegisterLeadGeneratorServiceServer(g.S, webApi.NewLeadGRPC(&g.Cfg.AppConfig, g.Logger, g.Postgres, g.Redis))
}

func (g *AppRunner) KnowledgeConnectApiRoute() {
	web_api.RegisterConnectServiceServer(g.S, webApi.NewConnectGRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres))
	g.Logger.Info("Internal HealthCheckRoutes and Connectors added to engine.")
	apiv1 := g.E.Group("/connect-knowledge")
	connectApi := webApi.NewConnectRPC(&g.Cfg.AppConfig, &g.Cfg.OAuthConfig, g.Logger, g.Postgres)
	{
		// working
		apiv1.GET("/notion/", connectApi.NotionConnect)

		apiv1.GET("/confluence/", connectApi.ConfluenceConnect)
		apiv1.GET("/google-drive/", connectApi.GoogleDriveConnect)
		//
		apiv1.GET("/github/", connectApi.GithubCodeConnect)
		apiv1.GET("/gitlab/", connectApi.GitlabCodeConnect)

		apiv1.GET("/microsoft-onedrive/", connectApi.MicrosoftOnedriveConnect)
		apiv1.GET("/sharepoint/", connectApi.MicrosoftSharepointConnect)
	}

	actionApiv1 := g.E.Group("/connect-action")
	{
		actionApiv1.GET("/gmail/", connectApi.GmailActionConnect)
		actionApiv1.GET("/jira/", connectApi.JiraActionConnect)
		actionApiv1.GET("/slack/", connectApi.SlackActionConnect)
	}

	crmConnectApiv1 := g.E.Group("/connect-crm")
	{
		crmConnectApiv1.GET("/hubspot/", connectApi.HubspotCRMConnect)
	}
}

func (g *AppRunner) HealthCheckRoutes() {
	g.Logger.Info("Internal HealthCheckRoutes and Connectors added to engine.")
	apiv1 := g.E.Group("")
	hcApi := healthCheckApi.New(g.Cfg, g.Logger, g.Postgres)
	{
		apiv1.GET("/readiness/", hcApi.Readiness)
		apiv1.GET("/healthz/", hcApi.Healthz)
	}
}

// Logger middleware
func (g *AppRunner) LoggerMiddleware() {
	aLogger := commons.NewApplicationLoggerWithOptions(
		commons.Level(g.Cfg.LogLevel),
		commons.Name(g.Cfg.Name),
	)
	aLogger.InitLogger()
	aLogger.Info("Added Logger middleware to the application.")
	g.Logger = aLogger
}

// Recovery middleware
func (g *AppRunner) RecoveryMiddleware() {
	g.Logger.Info("Added Default Recovery middleware to the application.")
	g.E.Use(gin.Recovery())
}

func (g *AppRunner) CorsMiddleware() {
	g.Logger.Info("Added Default Cors middleware to the application.")
	g.E.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "PUT", "POST", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Cache-Control", "Access-Control-Allow-Origin", "X-Grpc-Web"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}

// Logger Middleware
func (g *AppRunner) RequestLoggerMiddleware() {
	g.Logger.Info("Adding request middleware to the applicaiton.")
	g.E.Use(middlewares.NewRequestLoggerMiddleware(g.Cfg.Name, g.Logger))
}
