package health_check_api

import (
	"github.com/gin-gonic/gin"
	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	connectors "github.com/lexatic/web-backend/pkg/connectors"
)

type healthCheckApi struct {
	cfg      *config.WebAppConfig
	postgres connectors.Connector
	logger   commons.Logger
}

func New(config *config.WebAppConfig, logger commons.Logger,
	postgres connectors.Connector) *healthCheckApi {
	return &healthCheckApi{
		cfg:      config,
		logger:   logger,
		postgres: postgres,
	}
}

// @Router /v1/readiness [get]
// @Summary Readiness of service state of connections and other dependencies
// @Produce json
// @Success 200 {object} app.Response
// @Failure 500 {object} app.Response
func (hcApi *healthCheckApi) Readiness(c *gin.Context) {

	c.JSON(200, commons.Response{
		Code:    200,
		Success: true,
		Data: map[string]bool{
			hcApi.postgres.Name(): hcApi.postgres.IsConnected(c),
		},
	})

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
