package adapter_internal

import (
	"testing"

	"github.com/rapidaai/config"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/stretchr/testify/require"
)

func TestInternalCallersUseRapidaClient(t *testing.T) {
	config := &config.AppConfig{}
	vault := web_client.NewVaultClientWithClient(config, nil, nil, nil)
	integration := integration_client.NewIntegrationServiceClientWithClient(config, nil, nil, nil)
	deployment := endpoint_client.NewDeploymentServiceClientWithClient(config, nil, nil, nil)
	client := &rapida_client.RapidaClient{
		Vault:       vault,
		Integration: integration,
		Deployment:  deployment,
	}
	requestor := &genericRequestor{rapidaClient: client}

	require.Same(t, vault, requestor.VaultCaller())
	require.Same(t, integration, requestor.IntegrationCaller())
	require.Same(t, deployment, requestor.DeploymentCaller())
}
