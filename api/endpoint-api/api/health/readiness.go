package endpoint_health_api

import (
	"github.com/gin-gonic/gin"
	commons "github.com/rapidaai/pkg/commons"
)

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
