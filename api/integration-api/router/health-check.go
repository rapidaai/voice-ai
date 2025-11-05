package integration_routers

import (
	"github.com/gin-gonic/gin"
	healthCheckApi "github.com/rapidaai/api/integration-api/api/health"
	config "github.com/rapidaai/api/integration-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
)

func HealthCheckRoutes(cfg *config.IntegrationConfig, engine *gin.Engine, logger commons.Logger, postgres connectors.PostgresConnector) {
	logger.Info("Internal HealthCheckRoutes and Connectors added to engine.")
	apiv1 := engine.Group("")
	hcApi := healthCheckApi.New(cfg, logger, postgres)
	{
		apiv1.GET("/readiness/", hcApi.Readiness)
		apiv1.GET("/healthz/", hcApi.Healthz)
	}
}
