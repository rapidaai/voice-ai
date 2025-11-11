package endpoint_health_api

import (
	"github.com/gin-gonic/gin"
	config "github.com/rapidaai/api/endpoint-api/config"
	commons "github.com/rapidaai/pkg/commons"
	connectors "github.com/rapidaai/pkg/connectors"
)

type healthCheckApi struct {
	cfg      *config.EndpointConfig
	postgres connectors.Connector
	logger   commons.Logger
}

func New(config *config.EndpointConfig, logger commons.Logger,
	postgres connectors.Connector) *healthCheckApi {
	return &healthCheckApi{
		cfg:      config,
		logger:   logger,
		postgres: postgres,
	}
}

// @Router /v1/healthz [get]
// @Summary Readiness of application state
// @Produce json
// @Success 200 {object} app.Response
// @Failure 500 {object} app.Response
func (hcApi *healthCheckApi) Healthz(c *gin.Context) {
	c.JSON(200, commons.Response{
		Code:    200,
		Success: true,
		Data: commons.HealthCheck{
			Healthy: true,
		},
	})
}
