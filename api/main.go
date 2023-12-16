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
	healthCheckApi "github.com/lexatic/web-backend/api/health-check-api"
	webApi "github.com/lexatic/web-backend/api/web-api"
	config "github.com/lexatic/web-backend/config"
	internal_user_service "github.com/lexatic/web-backend/internal/services/user"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	middlewares "github.com/lexatic/web-backend/pkg/middlewares"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// wrapper for gin engine
type AppRunner struct {
	E         *gin.Engine
	S         *grpc.Server
	Cfg       *config.AppConfig
	Logger    commons.Logger
	Postgres  connectors.PostgresConnector
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
		grpc.StreamInterceptor(middlewares.StreamServerInterceptor(internal_user_service.NewAuthenticator(appRunner.Logger, appRunner.Postgres), appRunner.Logger)),
		grpc.UnaryInterceptor(middlewares.UnaryServerInterceptor(internal_user_service.NewAuthenticator(appRunner.Logger, appRunner.Postgres), appRunner.Logger)),
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
	// grpcFilteredListener := cmuxListener.Match(
	// 	cmux.HTTP1HeaderField("Content-type", "application/grpc-web+proto"),
	// 	cmux.HTTP1HeaderField("x-grpc-web", "1"))
	// rpcFilteredListener := cmuxListener.Match(cmux.Any())
	rpcFilteredListener := cmuxListener.Match(cmux.HTTP2())
	grpcFilteredListener := cmuxListener.Match(cmux.Any())

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
	g.Postgres = postgres
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
	app.Closeable = append(app.Closeable, app.Postgres.Disconnect)

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
	g.LeadApiRoute()

}

// all middleware
func (g *AppRunner) AllMiddlewares() {
	g.LoggerMiddleware()
	g.RecoveryMiddleware()
	g.CorsMiddleware()
	g.RequestLoggerMiddleware()
	g.E.Use(middlewares.AuthenticationMiddleware(internal_user_service.NewAuthenticator(g.Logger, g.Postgres), g.Logger))
}

func (g *AppRunner) AuthApiRoutes() {
	apiv1 := g.E.Group("/v1")
	{
		apiv1.POST("/auth/authenticate/", webApi.NewAuthRPC(g.Cfg, g.Logger, g.Postgres).Authenticate)
		apiv1.POST("/auth/register-user/", webApi.NewAuthRPC(g.Cfg, g.Logger, g.Postgres).RegisterUser)

	}
	web_api.RegisterAuthenticationServiceServer(g.S, webApi.NewAuthGRPC(g.Cfg, g.Logger, g.Postgres))

}

func (g *AppRunner) OauthApiRoute() {
	apiOauth := g.E.Group("/oauth")
	auth := webApi.NewAuthRPC(g.Cfg, g.Logger, g.Postgres)
	{
		apiOauth.GET("/google/", auth.Google)
		apiOauth.GET("/linkedin/", auth.Linkedin)
		apiOauth.GET("/github/", auth.Github)
	}
}

func (g *AppRunner) VaultApiRoute() {
	web_api.RegisterVaultServiceServer(g.S, webApi.NewVaultGRPC(g.Cfg, g.Logger, g.Postgres))
}

func (g *AppRunner) OrganizationApiRoute() {
	web_api.RegisterOrganizationServiceServer(g.S, webApi.NewOrganizationGRPC(g.Cfg, g.Logger, g.Postgres))
}

func (g *AppRunner) ProjectApiRoute() {
	web_api.RegisterProjectServiceServer(g.S, webApi.NewProjectGRPC(g.Cfg, g.Logger, g.Postgres))
}

func (g *AppRunner) LeadApiRoute() {
	web_api.RegisterLeadServiceServer(g.S, webApi.NewLeadGRPC(g.Cfg, g.Logger, g.Postgres))
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
		AllowAllOrigins: true,
		// AllowOrigins:     []string{".*"},
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
	g.E.Use(middlewares.RequestLoggerMiddleware(g.Cfg.Name, g.Logger))
}
