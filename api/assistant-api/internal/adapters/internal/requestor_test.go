package adapter_internal

import (
	"context"
	"testing"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/config"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	web_client "github.com/rapidaai/pkg/clients/web"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func TestNewGenericRequestorLifecycleUsesCurrentStreamer(t *testing.T) {
	cfg := &assistant_config.AssistantConfig{}
	cfg.Integration.Host = "localhost:1"
	original := &streamTestStreamer{}
	requestor := NewGenericRequestor(context.Background(), cfg, nil, nil, utils.Debugger,
		nil, nil, nil, nil, original, nil)
	defer requestor.cancelSession()
	require.False(t, requestor.messageLifecycle.InterruptionEnabled())
	require.Equal(t, type_enums.TextMode, requestor.GetMode())
	require.NotEmpty(t, requestor.GetID())
	message := &protos.ConversationAssistantMessage{Id: requestor.GetID(), Message: &protos.ConversationAssistantMessage_Text{Text: "hello"}}
	require.NoError(t, requestor.Notify(context.Background(), message))
	require.Len(t, original.sent, 1)

	current := &streamTestStreamer{}
	requestor.streamer = current
	require.NoError(t, requestor.Notify(context.Background(), message))
	require.NoError(t, requestor.sendOutputControl(&protos.ConversationPlaybackPause{Id: requestor.GetID()}))
	require.Len(t, original.sent, 1)
	require.Len(t, current.sent, 2)
	require.IsType(t, &protos.ConversationPlaybackPause{}, current.sent[1])

	requestor.streamer = nil
	require.ErrorContains(t, requestor.Notify(context.Background(), message), "streamer is unavailable")
	require.ErrorContains(t, requestor.messageLifecycle.SendPlaybackControl(&protos.ConversationPlaybackFlush{}), "streamer is unavailable")
}

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
