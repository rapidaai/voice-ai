package main

import (
	"context"
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
	"github.com/rapidaai/api/assistant-api/config"
	router "github.com/rapidaai/api/assistant-api/router"
	"github.com/rapidaai/pkg/authenticators"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/middlewares"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// wrapper for gin engine
type AppRunner struct {
	E          *gin.Engine
	S          *grpc.Server
	Cfg        *config.AssistantConfig
	Logger     commons.Logger
	Postgres   connectors.PostgresConnector
	Redis      connectors.RedisConnector
	Opensearch connectors.OpenSearchConnector
	Closeable  []func(context.Context) error
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

	// Migration if needed to run
	if err := appRunner.Migrate(); err != nil {
		appRunner.Logger.Errorf("Warning: Migration failed: %v", err)
		panic(err)
	}

	// init
	appRunner.S = grpc.NewServer(
		grpc.ChainStreamInterceptor(
			middlewares.NewRequestLoggerStreamServerMiddleware(appRunner.Cfg.Name, appRunner.Logger),
			middlewares.NewServiceAuthenticatorStreamServerMiddleware(
				authenticators.NewServiceAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Postgres),
				appRunner.Logger,
			),
			middlewares.NewAuthenticationStreamServerMiddleware(
				authenticators.NewUserAuthenticator(&appRunner.Cfg.AppConfig,
					appRunner.Logger,
					web_client.NewAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Redis)),
				appRunner.Logger,
			),
			middlewares.NewProjectAuthenticatorStreamServerMiddleware(
				authenticators.NewProjectAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger,
					web_client.NewAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Redis)),
				appRunner.Logger,
			),
			middlewares.NewClientInformationStreamServerMiddleware(
				appRunner.Logger,
			),
		),
		grpc.ChainUnaryInterceptor(
			middlewares.NewRequestLoggerUnaryServerMiddleware(appRunner.Cfg.AppConfig.Name, appRunner.Logger),
			middlewares.NewProjectAuthenticatorUnaryServerMiddleware(
				authenticators.NewProjectAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger,
					web_client.NewAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Redis)),
				appRunner.Logger,
			),
			middlewares.NewAuthenticationUnaryServerMiddleware(
				authenticators.NewUserAuthenticator(&appRunner.Cfg.AppConfig,
					appRunner.Logger,
					web_client.NewAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Redis)),
				appRunner.Logger,
			),
			middlewares.NewServiceAuthenticatorUnaryServerMiddleware(
				authenticators.NewServiceAuthenticator(&appRunner.Cfg.AppConfig, appRunner.Logger, appRunner.Postgres),
				appRunner.Logger,
			),
			middlewares.NewClientInformationUnaryServerMiddleware(
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

	// all router add all handlers which required to resolve the service request
	appRunner.AllRouters()
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", appRunner.Cfg.Host, appRunner.Cfg.Port))
	if err != nil {
		log.Fatalf("Failed to create connection tcp %v", err)
		panic(err)
	}

	defer appRunner.Close(ctx)

	cmuxListener := cmux.New(listener)
	http2GRPCFilteredListener := cmuxListener.Match(cmux.HTTP2())
	grpcFilteredListener := cmuxListener.Match(
		cmux.HTTP1HeaderField("content-type", "application/grpc-web+proto"),
		cmux.HTTP1HeaderField("sec-websocket-protocol", "grpc-websockets"),
		cmux.HTTP1HeaderField("x-grpc-web", "1"))
	rpcFilteredListener := cmuxListener.Match(cmux.Any())
	group, ctx := errgroup.WithContext(ctx)

	// here is grpc
	group.Go(func() error {
		err = appRunner.S.Serve(http2GRPCFilteredListener)
		if err != nil {
			appRunner.Logger.Errorf("Failed to start grpc server err: %v", err)
		}
		return err
	})
	group.Go(func() error {
		err = appRunner.E.RunListener(rpcFilteredListener)
		if err != nil {
			appRunner.Logger.Errorf("Failed to start gin server err: %v", err)
		}
		return err
	})
	group.Go(func() error {
		//
		wrappedServer := grpcweb.WrapServer(appRunner.S,
			grpcweb.WithWebsockets(true),
			grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
				return true
			}),
			grpcweb.WithWebsocketPingInterval(45*time.Second),
			grpcweb.WithWebsocketsMessageReadLimit(100*1024*1024),
		)
		handler := func(resp http.ResponseWriter, req *http.Request) {
			wrappedServer.ServeHTTP(resp, req)
		}

		httpServer := http.Server{
			Handler: http.HandlerFunc(handler),
		}
		err = httpServer.Serve(grpcFilteredListener)
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
	g.Postgres = connectors.NewPostgresConnector(&g.Cfg.PostgresConfig, g.Logger)
	g.Redis = connectors.NewRedisConnector(&g.Cfg.RedisConfig, g.Logger)
	g.Opensearch = connectors.NewOpenSearchConnector(&g.Cfg.OpenSearchConfig, g.Logger)
	// g.Weaviate = vdb_connectors.NewWeaviateConnector(&g.Cfg.WeaviateConfig, g.Logger)
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

	err = app.Opensearch.Connect(ctx)
	if err != nil {
		app.Logger.Error("error while connecting to opensearch.", err)
		return err
	}

	app.Closeable = append(app.Closeable, app.Opensearch.Disconnect)
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
	router.AssistantApiRoute(g.Cfg, g.S, g.Logger, g.Postgres, g.Redis, g.Opensearch)
	router.HealthCheckRoutes(g.Cfg, g.E, g.Logger, g.Postgres)
	router.KnowledgeApiRoute(g.Cfg, g.S, g.Logger, g.Postgres, g.Redis, g.Opensearch)
	router.DocumentApiRoute(g.Cfg, g.S, g.Logger, g.Postgres, g.Redis, g.Opensearch)
	router.AssistantConversationApiRoute(g.Cfg, g.S, g.Logger, g.Postgres, g.Redis, g.Opensearch)
	router.AssistantDeploymentApiRoute(g.Cfg, g.S, g.Logger, g.Postgres)

	// rpc call handle by gin handler
	router.TalkCallbackApiRoute(g.Cfg, g.E, g.Logger, g.Postgres, g.Redis, g.Opensearch)
	router.ConversationApiRoute(g.Cfg, g.E, g.Logger, g.Postgres, g.Redis, g.Opensearch)

}

// all middleware
func (g *AppRunner) AllMiddlewares() {
	g.RecoveryMiddleware()
	g.CorsMiddleware()
	g.RequestLoggerMiddleware()
	g.AuthenticationMiddleware()
}

// Recovery middleware
func (g *AppRunner) RecoveryMiddleware() {
	g.Logger.Info("Added Default Recovery middleware to the application.")
	g.E.Use(gin.Recovery())
}

func (g *AppRunner) AuthenticationMiddleware() {
	g.Logger.Info("Adding request middleware to the application.")
	g.E.Use(middlewares.NewAuthenticationMiddleware(
		authenticators.NewUserAuthenticator(&g.Cfg.AppConfig,
			g.Logger,
			web_client.NewAuthenticator(&g.Cfg.AppConfig, g.Logger, g.Redis)),
		g.Logger,
	))
	g.E.Use(middlewares.NewProjectAuthenticatorMiddleware(
		authenticators.NewProjectAuthenticator(
			&g.Cfg.AppConfig,
			g.Logger,
			web_client.NewAuthenticator(&g.Cfg.AppConfig, g.Logger, g.Redis),
		),
		g.Logger,
	))
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
	g.Logger.Info("Adding request middleware to the application.")
	g.E.Use(middlewares.NewRequestLoggerMiddleware(g.Cfg.Name, g.Logger))

}
func (app *AppRunner) Migrate() error {
	withMigration := flag.Bool("with-migration", false, "Run migration when provided, eg: -with-migration")
	flag.Parse()
	if withMigration == nil || *withMigration == false {
		app.Logger.Infof("Skipping the migration, if not you need to check the argument -with-migration")
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
	migrationsPath := fmt.Sprintf("file://%s/api/assistant-api/migrations", currentDir)

	app.Logger.Infof("Looking for migration files at path: %s", migrationsPath)
	app.Logger.Infof("Using DSN for migration: %s", dsn)

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
		app.Logger.Warnf("Database is in a dirty state at version: %d. Trying to force clean...", version)
		if err := m.Force(int(version - 1)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
		app.Logger.Infof("Migration state forced to clean. You can restart migration.")
		return nil
	}

	// Perform database migration
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			app.Logger.Infof("No migration changes detected.")
		} else {
			return fmt.Errorf("migration failed: %w", err)
		}
	} else {
		app.Logger.Infof("Migrations completed successfully.")
	}

	return nil
}
