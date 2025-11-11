package endpoint_api

import (
	"context"
	"errors"
	"fmt"
	"time"

	config "github.com/rapidaai/api/endpoint-api/config"
	internal_services "github.com/rapidaai/api/endpoint-api/internal/service"
	internal_endpoint_service "github.com/rapidaai/api/endpoint-api/internal/service/endpoint"
	internal_log_service "github.com/rapidaai/api/endpoint-api/internal/service/log"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	integration_client_builders "github.com/rapidaai/pkg/clients/integration/builders"
	web_client "github.com/rapidaai/pkg/clients/web"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	invoker_api "github.com/rapidaai/protos"
)

type invokerApi struct {
	cfg                *config.EndpointConfig
	logger             commons.Logger
	postgres           connectors.PostgresConnector
	endpointService    internal_services.EndpointService
	endpointLogService internal_services.EndpointLogService
	integrationClient  integration_client.IntegrationServiceClient
	inputBuilder       integration_client_builders.InputChatBuilder
	vaultClient        web_client.VaultClient
}

type invokerGRPCApi struct {
	invokerApi
}

func NewInvokerGRPCApi(config *config.EndpointConfig, logger commons.Logger,
	postgres connectors.PostgresConnector, redis connectors.RedisConnector,
) invoker_api.DeploymentServer {
	return &invokerGRPCApi{
		invokerApi{
			cfg:                config,
			logger:             logger,
			postgres:           postgres,
			endpointService:    internal_endpoint_service.NewEndpointService(config, logger, postgres),
			integrationClient:  integration_client.NewIntegrationServiceClientGRPC(&config.AppConfig, logger, redis),
			inputBuilder:       integration_client_builders.NewChatInputBuilder(logger),
			vaultClient:        web_client.NewVaultClientGRPC(&config.AppConfig, logger, redis),
			endpointLogService: internal_log_service.NewEndpointLogService(logger, postgres),
		},
	}
}

func (invokeApi *invokerGRPCApi) Invoke(ctx context.Context, iRequest *invoker_api.InvokeRequest) (*invoker_api.InvokeResponse, error) {
	start := time.Now()
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		return utils.AuthenticateError[invoker_api.InvokeResponse]()
	}

	requestID := gorm_generator.ID()
	clientSource, ok := utils.GetClientSource(ctx)
	if !ok {
		clientSource = utils.SDK
	}

	arguments, err := utils.AnyMapToInterfaceMap(iRequest.GetArgs())
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Please check and provide a valid arguments.")
	}
	mtds, err := utils.AnyMapToInterfaceMap(iRequest.GetMetadata())
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Please check and provide a valid metadata.")
	}
	opts, err := utils.AnyMapToInterfaceMap(iRequest.GetOptions())
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Please check and provide a valid options.")
	}

	endpoint, err := invokeApi.endpointService.Get(ctx,
		iAuth,
		iRequest.GetEndpoint().GetEndpointId(),
		utils.GetVersionDefinition(iRequest.GetEndpoint().GetVersion()),
		internal_services.NewGetEndpointOption())

	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Please check endpoint configuration and try again.")
	}

	utils.Go(ctx, func() {
		invokeApi.endpointLogService.CreateEndpointLog(
			ctx,
			iAuth,
			clientSource,
			endpoint.Id,
			endpoint.EndpointProviderModelId,
			requestID,
			arguments, mtds, opts,
		)
	})

	credentialID, err := endpoint.
		EndpointProviderModel.
		GetOptions().GetUint64("rapida.credential_id")
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, errors.New("rapida.credential_id not found in model options"), "Please check endpoint configuration and try again.")
	}
	vlt, err := invokeApi.vaultClient.GetCredential(ctx, iAuth, credentialID)
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Please check credential for provider and update it.")
	}

	output, err := invokeApi.
		integrationClient.
		Chat(ctx,
			iAuth,
			endpoint.
				EndpointProviderModel.
				ModelProviderName,
			invokeApi.
				inputBuilder.
				Chat(
					&invoker_api.Credential{
						Id:    vlt.GetId(),
						Value: vlt.GetValue(),
					},
					invokeApi.
						inputBuilder.
						Options(
							endpoint.
								EndpointProviderModel.
								GetOptions(),
							iRequest.
								GetOptions(),
						),
					nil,
					map[string]string{
						"endpoint_id":                fmt.Sprintf("%d", endpoint.Id),
						"vault_id":                   fmt.Sprintf("%d", vlt.Id),
						"endpoint_provider_model_id": fmt.Sprintf("%d", endpoint.EndpointProviderModel.Id),
					},
					invokeApi.
						inputBuilder.
						Message(
							endpoint.
								EndpointProviderModel.
								Request.
								GetTextChatCompleteTemplate().
								Prompt,
							invokeApi.
								inputBuilder.
								Arguments(endpoint.
									EndpointProviderModel.
									Request.
									GetTextChatCompleteTemplate().Variables, iRequest.GetArgs()),
						)...,
				))

	utils.Go(context.Background(), func() {
		invokeApi.endpointLogService.UpdateEndpointLog(
			context.Background(),
			iAuth,
			requestID,
			output.GetMetrics(),
			uint64(time.Since(start)),
		)
	})
	if err != nil {
		return utils.ErrorWithCode[invoker_api.InvokeResponse](400, err, "Unable to execute the endpoint, please check and try again.")
	}

	return &invoker_api.InvokeResponse{
		RequestId: requestID,
		Code:      200,
		Success:   true,
		TimeTaken: uint64(time.Since(start).Microseconds()),
		Data:      output.GetData().GetContents(),
		Metrics:   output.GetMetrics(),
	}, nil

}

func (endpoint *invokerGRPCApi) Probe(ctx context.Context, rpv *invoker_api.ProbeRequest) (*invoker_api.ProbeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		return utils.AuthenticateError[invoker_api.ProbeResponse]()
	}
	return nil, nil
}

func (endpoint *invokerGRPCApi) Update(ctx context.Context, ur *invoker_api.UpdateRequest) (*invoker_api.UpdateResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		return utils.AuthenticateError[invoker_api.UpdateResponse]()
	}
	return nil, nil
}
