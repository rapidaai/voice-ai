package assistant_deployment_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
)

// GetAssistantApiDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantApiDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	apiDeployment, err := deploymentApi.deploymentService.GetAssistantApiDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &assistant_api.AssistantApiDeployment{}
	err = utils.Cast(apiDeployment, out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the api deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantApiDeploymentResponse, *assistant_api.AssistantApiDeployment](out)
}

// GetAssistantDebuggerDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantDebuggerDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	debuggerDeployment, err := deploymentApi.deploymentService.GetAssistantDebuggerDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &assistant_api.AssistantDebuggerDeployment{}
	err = utils.Cast(debuggerDeployment, out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the api deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantDebuggerDeploymentResponse, *assistant_api.AssistantDebuggerDeployment](out)
}

// GetAssistantPhoneDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantPhoneDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	phoneDeployment, err := deploymentApi.deploymentService.GetAssistantPhoneDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &assistant_api.AssistantPhoneDeployment{}
	err = utils.Cast(phoneDeployment, out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the api deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantPhoneDeploymentResponse, *assistant_api.AssistantPhoneDeployment](out)
}

// GetAssistantWebpluginDeployment implements lexatic_backend.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantWebpluginDeployment(ctx context.Context, getter *lexatic_backend.GetAssistantDeploymentRequest) (*lexatic_backend.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	webpluginDeployment, err := deploymentApi.deploymentService.GetAssistantWebpluginDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[lexatic_backend.GetAssistantWebpluginDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &lexatic_backend.AssistantWebpluginDeployment{}
	err = utils.Cast(webpluginDeployment, out)
	if err != nil {
		deploymentApi.logger.Warnf("unable to cast the web plugin deployment model to the response object")
	}

	out.Icon = &lexatic_backend.AssistantWebpluginDeployment_Url{
		Url: webpluginDeployment.Icon,
	}
	deploymentApi.logger.Debugf("responding %+v", out)
	return &lexatic_backend.GetAssistantWebpluginDeploymentResponse{
		Data:    out,
		Success: true,
		Code:    200,
	}, nil
}

// GetAssistantWhatsappDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) GetAssistantWhatsappDeployment(ctx context.Context, getter *assistant_api.GetAssistantDeploymentRequest) (*assistant_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	whatsappDeployment, err := deploymentApi.deploymentService.GetAssistantWhatsappDeployment(ctx, iAuth, getter.GetAssistantId())
	if err != nil {
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](err, "Unable to get deployment, please try again later.")
	}
	out := &assistant_api.AssistantWhatsappDeployment{}
	err = utils.Cast(whatsappDeployment, out)
	if err != nil {
		deploymentApi.logger.Errorf("unable to cast the whatsapp deployment model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantWhatsappDeploymentResponse, *assistant_api.AssistantWhatsappDeployment](out)
}
