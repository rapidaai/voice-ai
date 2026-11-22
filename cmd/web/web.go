// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package main

import (
	"context"
	"errors"
	"flag"
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
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/improbable-eng/grpc-web/go/grpcweb"

	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	web_authenticators "github.com/rapidaai/api/web-api/authenticator"
	"github.com/rapidaai/api/web-api/config"
	web_router "github.com/rapidaai/api/web-api/router"
	command_lifecycle "github.com/rapidaai/cmd/internal/lifecycle"
	"github.com/rapidaai/pkg/authentication/policy"
	"github.com/rapidaai/pkg/authentication/resolver"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/middlewares"
	"github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
)

// wrapper for gin engine
type AppRunner struct {
	E         *gin.Engine
	S         *grpc.Server
	Cfg       *config.WebAppConfig
	Logger    commons.Logger
	Postgres  connectors.PostgresConnector
	Redis     connectors.RedisConnector
	Clients   *rapida_client.RapidaClient
	Closeable []func(context.Context) error
}

func main() {
	// creating a common context
	ctx := context.Background()

	appRunner := AppRunner{E: gin.New()}
	// resolving configuration
	if err := appRunner.ResolveConfig(); err != nil {
		panic(err)
	}
	// logging
	if err := appRunner.Logging(); err != nil {
		panic(err)
	}

	// adding all connectors
	appRunner.AllConnectors()

	// Migration if needed to run
	if err := appRunner.Migrate(); err != nil {
		appRunner.Logger.Errorf("Warning: Migration failed: %v", err)
		panic(err)
	}
	if err := appRunner.Init(ctx); err != nil {
		appRunner.Close(ctx)
		panic(err)
	}
	defer appRunner.Close(ctx)

	authorizationPolicy, err := policy.NewWebAuthorizationPolicy()
	if err != nil {
		panic(err)
	}
	appRunner.S = grpc.NewServer(
		grpc.ChainStreamInterceptor(
			middlewares.NewRequestLoggerStreamServerMiddleware(appRunner.Cfg.Name, appRunner.Logger),
			middlewares.NewRecoveryStreamServerMiddleware(appRunner.Logger),
			authentication.NewProjectStreamMiddleware(web_authenticators.NewProjectResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewOrganizationStreamMiddleware(web_authenticators.NewOrganizationResolver(appRunner.Cfg.OrganizationCredentialFingerprintKey(), appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewUserStreamMiddleware(web_authenticators.NewUserResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewServiceStreamMiddleware(resolver.NewServicePrincipalResolver(appRunner.Cfg.Name, appRunner.Cfg.ServiceAuthenticationKey()), appRunner.Logger),
			authentication.NewPolicyStreamMiddleware(authorizationPolicy, appRunner.Logger),
		),
		grpc.ChainUnaryInterceptor(
			middlewares.NewRequestLoggerUnaryServerMiddleware(appRunner.Cfg.Name, appRunner.Logger),
			middlewares.NewRecoveryUnaryServerMiddleware(appRunner.Logger),
			authentication.NewProjectUnaryMiddleware(web_authenticators.NewProjectResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewOrganizationUnaryMiddleware(web_authenticators.NewOrganizationResolver(appRunner.Cfg.OrganizationCredentialFingerprintKey(), appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewUserUnaryMiddleware(web_authenticators.NewUserResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
			authentication.NewServiceUnaryMiddleware(resolver.NewServicePrincipalResolver(appRunner.Cfg.Name, appRunner.Cfg.ServiceAuthenticationKey()), appRunner.Logger),
			authentication.NewPolicyUnaryMiddleware(authorizationPolicy, appRunner.Logger),
		),
		grpc.MaxRecvMsgSize(commons.MaxRecvMsgSize), // 10 MB
		grpc.MaxSendMsgSize(commons.MaxSendMsgSize), // 10 MB
	)
	// add all middleware depends on configurations
	appRunner.AllMiddlewares(
		authentication.NewProjectHTTPMiddleware(web_authenticators.NewProjectResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
		authentication.NewOrganizationHTTPMiddleware(web_authenticators.NewOrganizationResolver(appRunner.Cfg.OrganizationCredentialFingerprintKey(), appRunner.Logger, appRunner.Postgres), appRunner.Logger),
		authentication.NewUserHTTPMiddleware(web_authenticators.NewUserResolver(appRunner.Logger, appRunner.Postgres), appRunner.Logger),
		authentication.NewServiceHTTPMiddleware(resolver.NewServicePrincipalResolver(appRunner.Cfg.Name, appRunner.Cfg.ServiceAuthenticationKey()), appRunner.Logger),
		authentication.NewPolicyHTTPMiddleware(authorizationPolicy, appRunner.Logger),
	)

	// all routers add all handlers which required to resolve the service request
	appRunner.AllRouters(appRunner.Cfg.OrganizationCredentialFingerprintKey())
	appRunner.AllProxyRouter()
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", appRunner.Cfg.Host, appRunner.Cfg.Port))
	if err != nil {
		log.Fatalf("Failed to create connection tcp %v", err)
		panic(err)
	}

	cmuxListener := command_lifecycle.NewMux(listener, 10*time.Second)

	// if application json
	http2GRPCFilteredListener := cmuxListener.Match(cmux.HTTP2())
	grpcFilteredListener := cmuxListener.Match(
		cmux.HTTP1HeaderField("content-type", "application/grpc-web+proto"),
		cmux.HTTP1HeaderField("x-grpc-web", "1"))
	rpcFilteredListener := cmuxListener.Match(cmux.Any())
	httpServer := &http.Server{
		Handler:           appRunner.E,
		ReadHeaderTimeout: 10 * time.Second,
	}
	wrappedServer := grpcweb.WrapServer(appRunner.S, grpcweb.WithOriginFunc(func(origin string) bool { return true }))
	grpcWebServer := &http.Server{
		Handler:           wrappedServer,
		ReadHeaderTimeout: 10 * time.Second,
	}
	// rpcFilteredListener := cmuxListener.Match(cmux.HTTP2())
	// grpcFilteredListener := cmuxListener.Match(cmux.Any())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	group, groupContext := errgroup.WithContext(ctx)
	serveDone := make(chan struct{})
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		select {
		case <-quit:
		case <-groupContext.Done():
		case <-serveDone:
			return
		}
		cmuxListener.Close()
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			appRunner.Logger.Errorf("Failed to close server listener: %v", closeErr)
		}
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if shutdownErr := command_lifecycle.Shutdown(shutdownContext, appRunner.S, httpServer, grpcWebServer); shutdownErr != nil {
			appRunner.Logger.Errorf("Failed to drain servers: %v", shutdownErr)
		}
	}()

	group.Go(func() error {
		serveErr := httpServer.Serve(rpcFilteredListener)
		if serveErr != nil && !errors.Is(serveErr, cmux.ErrServerClosed) && !errors.Is(serveErr, cmux.ErrListenerClosed) {
			appRunner.Logger.Errorf("Failed to start gin server err: %v", serveErr)
		}
		return serveErr
	})
	group.Go(func() error {
		serveErr := grpcWebServer.Serve(grpcFilteredListener)
		if serveErr != nil && !errors.Is(serveErr, cmux.ErrServerClosed) && !errors.Is(serveErr, cmux.ErrListenerClosed) && !errors.Is(serveErr, http.ErrServerClosed) {
			appRunner.Logger.Errorf("Failed to start grpc-web server err: %v", serveErr)
		}
		return serveErr
	})

	group.Go(func() error {
		serveErr := appRunner.S.Serve(http2GRPCFilteredListener)
		if serveErr != nil && !errors.Is(serveErr, cmux.ErrServerClosed) && !errors.Is(serveErr, cmux.ErrListenerClosed) && !errors.Is(serveErr, grpc.ErrServerStopped) {
			appRunner.Logger.Errorf("Failed to start grpc server err: %v", serveErr)
		}
		return serveErr
	})
	// serve now
	serveErr := cmuxListener.Serve()
	close(serveDone)
	groupErr := group.Wait()
	<-shutdownDone
	if serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
		appRunner.Logger.Errorf("Failed to serve multiplexed listener: %v", serveErr)
		panic(serveErr)
	}
	if groupErr != nil && !errors.Is(groupErr, cmux.ErrServerClosed) && !errors.Is(groupErr, cmux.ErrListenerClosed) && !errors.Is(groupErr, grpc.ErrServerStopped) && !errors.Is(groupErr, http.ErrServerClosed) {
		panic(groupErr)
	}
}

func (app *AppRunner) Logging() error {
	aLogger, err := commons.NewApplicationLogger(
		commons.Level(app.Cfg.LogLevel),
		commons.Name(app.Cfg.Name),
	)
	if err != nil {
		return err
	}
	app.Logger = aLogger
	return nil
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
	// debug mode of gin when running log in debug mode.
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	}
	return nil
}

// init for app close
func (app *AppRunner) Init(ctx context.Context) error {
	if app.Postgres != nil {
		err := app.Postgres.Connect(ctx)
		if err != nil {
			app.Logger.Error("error while connecting to postgres.", err)
			return err
		}
		app.Closeable = append(app.Closeable, app.Postgres.Disconnect)
	}
	if app.Redis != nil {
		err := app.Redis.Connect(ctx)
		if err != nil {
			app.Logger.Error("error while connecting to redis.", err)
			return err
		}
		app.Closeable = append(app.Closeable, app.Redis.Disconnect)
	}

	clients, err := rapida_client.New(&app.Cfg.AppConfig, app.Logger, app.Redis)
	if err != nil {
		return fmt.Errorf("initialize internal clients: %w", err)
	}
	app.Clients = clients
	app.Closeable = append(app.Closeable, app.Clients.Close)
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
func (g *AppRunner) AllRouters(organizationCredentialKey []byte) {
	web_router.HealthCheckRoutes(g.Cfg, g.E, g.Logger, g.Postgres)
	web_router.WebApiRoute(g.Cfg, organizationCredentialKey, g.E, g.S, g.Logger, g.Postgres, g.Redis)
}

func (g *AppRunner) AllProxyRouter() {
	web_router.ProxyApiRoute(g.Cfg, g.S, g.Logger, g.Postgres, g.Redis, g.Clients)
}

// all middleware
func (g *AppRunner) AllMiddlewares(handlers ...gin.HandlerFunc) {
	g.RecoveryMiddleware()
	g.CorsMiddleware()
	g.RequestLoggerMiddleware()
	g.E.Use(handlers...)
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
		AllowMethods:    []string{"GET", "PUT", "POST", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders: []string{
			"Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Cache-Control",
			"Access-Control-Allow-Origin", "X-Grpc-Web", "X-Auth-Id", "X-Organization-Id", "X-Project-Id", "X-Api-Key",
			"X-Org-Key", "X-Internal-Service-Key",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}

// Logger Middleware
func (g *AppRunner) RequestLoggerMiddleware() {
	g.Logger.Info("Adding request middleware to the application.")
	g.E.Use(middlewares.NewRequestLoggerMiddleware(g.Cfg.Name, g.Logger))
}

func (app *AppRunner) Migrate() error {
	skipMigration := flag.Bool("skip-migration", false, "Skip migration when provided, eg: -skip-migration")
	flag.Parse()
	if *skipMigration {
		app.Logger.Infof("Skipping migration due to -skip-migration flag")
		return nil
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		app.Cfg.PostgresConfig.Auth.User,
		app.Cfg.PostgresConfig.Auth.Password,
		app.Cfg.PostgresConfig.Host,
		app.Cfg.PostgresConfig.Port,
		app.Cfg.PostgresConfig.DBName,
		app.Cfg.PostgresConfig.SslMode,
	)
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	migrationsPath := fmt.Sprintf("file://%s/api/web-api/migrations", currentDir)
	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("migration initialization failed: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			app.Logger.Errorf("Source closing error: %v", sourceErr)
		}
		if databaseErr != nil {
			app.Logger.Errorf("Database connection closing error: %v", databaseErr)
		}
	}()

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("error fetching migration version: %w", err)
	}
	if dirty {
		app.Logger.Warnf("Database is in a dirty state at version: %d. Forcing clean...", version)
		if version == 0 {
			return fmt.Errorf("cannot force migration below version zero")
		}
		previousVersion, conversionErr := utils.Uint64ToInt(uint64(version - 1))
		if conversionErr != nil {
			return fmt.Errorf("invalid migration version: %w", conversionErr)
		}
		if err := m.Force(previousVersion); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
		app.Logger.Infof("Migration state forced to clean at version: %d. Proceeding with migration.", version-1)
	}

	// Perform database migration
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			app.Logger.Infof("No migration changes detected.")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	app.Logger.Infof("Migrations completed successfully.")
	return nil
}
