package web_router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	web_config "github.com/rapidaai/api/web-api/config"
)

func TestWebAPIRouteRegistersProductUsageService(t *testing.T) {
	server := grpc.NewServer()
	WebApiRoute(&web_config.WebAppConfig{}, gin.New(), server, nil, nil, nil)

	if _, ok := server.GetServiceInfo()["billing_api.ProductUsageService"]; !ok {
		t.Fatal("ProductUsageService is not registered")
	}
}
