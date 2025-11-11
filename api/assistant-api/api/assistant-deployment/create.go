package assistant_deployment_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantApiDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantApiDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantApiDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant api deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	if deployment.GetApi() == nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for api.",
		)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreateApiDeployment(ctx,
		iAuth, deployment.GetApi().GetAssistantId(),
		deployment.GetApi().Greeting,
		deployment.GetApi().Mistake,
		deployment.GetApi().GetInputAudio(),
		deployment.GetApi().GetOutputAudio(),
	)
	if err != nil {
		return utils.Error[assistant_api.GetAssistantApiDeploymentResponse](
			errors.New("unauthenticated request for create assistant api deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantApiDeploymentResponse](wpDeployment)
}

// CreateAssistantDebuggerDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantDebuggerDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantDebuggerDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	if deployment.GetDebugger() == nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for debugger.",
		)
	}

	iconPath := "https://cdn-01.rapida.ai/partners/rapida.png"
	switch icon := deployment.GetDebugger().GetIcon().(type) {
	case *assistant_api.AssistantDebuggerDeployment_Raw:
		so := deploymentApi.storage.Store(ctx,
			icon.Raw.GetName(),
			icon.Raw.GetContent(),
		)
		if so.Error != nil {
			deploymentApi.logger.Errorf("error while uploading the image to cdn %+v", so.Error)
			break
		}
		iconPath = so.CompletePath
	case *assistant_api.AssistantDebuggerDeployment_Url:
		iconPath = icon.Url
		break
	}

	wpDeployment, err := deploymentApi.deploymentService.CreateDebuggerDeployment(ctx,
		iAuth, deployment.GetDebugger().GetAssistantId(),
		deployment.GetDebugger().GetName(),
		iconPath,
		deployment.GetDebugger().Greeting,
		deployment.GetDebugger().Mistake,
		deployment.GetDebugger().GetInputAudio(),
		deployment.GetDebugger().GetOutputAudio(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantDebuggerDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantDebuggerDeploymentResponse](wpDeployment)

}

// CreateAssistantPhoneDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantPhoneDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("unauthenticated request for create assistant phone deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	// name, role, tone, expertise, greeting, mistake, ending string,
	//
	if deployment.GetPhone() == nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for phone.",
		)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreatePhoneDeployment(ctx,
		iAuth, deployment.GetPhone().GetAssistantId(),
		deployment.GetPhone().Greeting,
		deployment.GetPhone().Mistake,
		deployment.GetPhone().GetPhoneProviderId(),
		deployment.GetPhone().GetPhoneProviderName(),
		deployment.GetPhone().GetInputAudio(),
		deployment.GetPhone().GetOutputAudio(),
		deployment.GetPhone().GetPhoneOptions(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal request for create assistant phone deployment"),
			"Please provider valid a valid request to create assistant phone deployment.",
		)
	}
	return utils.Success[assistant_api.GetAssistantPhoneDeploymentResponse](wpDeployment)
}

// CreateAssistantWebpluginDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantWebpluginDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantWebpluginDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("unauthenticated request for create assistant web plugin deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	if deployment.GetPlugin() == nil {
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for webplugin.",
		)
	}
	iconPath := "https://cdn-01.rapida.ai/partners/rapida.png"
	switch icon := deployment.GetPlugin().GetIcon().(type) {
	case *assistant_api.AssistantWebpluginDeployment_Raw:
		so := deploymentApi.storage.Store(ctx,
			icon.Raw.GetName(),
			icon.Raw.GetContent(),
		)
		if so.Error != nil {
			deploymentApi.logger.Errorf("error while uploading the image to cdn %+v", so.Error)
			break
		}
		iconPath = so.CompletePath
	case *assistant_api.AssistantWebpluginDeployment_Url:
		iconPath = icon.Url
		break
	}

	wpDeployment, err := deploymentApi.deploymentService.CreateWebPluginDeployment(ctx,
		iAuth, deployment.GetPlugin().GetAssistantId(),
		deployment.GetPlugin().GetName(),
		iconPath,
		deployment.GetPlugin().Greeting,
		deployment.GetPlugin().Mistake,
		deployment.GetPlugin().GetSuggestion(),
		deployment.GetPlugin().GetHelpCenterEnabled(),
		deployment.GetPlugin().GetProductCatalogEnabled(),
		deployment.GetPlugin().GetArticleCatalogEnabled(),
		deployment.GetPlugin().GetInputAudio(),
		deployment.GetPlugin().GetOutputAudio(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantWebpluginDeploymentResponse](
			errors.New("unauthenticated request for create assistant webplugin deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantWebpluginDeploymentResponse](wpDeployment)
}

// CreateAssistantWhatsappDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentApi) CreateAssistantWhatsappDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantWhatsappDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("unauthenticated request for create assistant whatsapp deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	if deployment.GetWhatsapp() == nil {
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for whatsapp.",
		)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreateWhatsappDeployment(ctx,
		iAuth, deployment.GetWhatsapp().GetAssistantId(),
		deployment.GetWhatsapp().Greeting,
		deployment.GetWhatsapp().Mistake,
		deployment.GetWhatsapp().GetWhatsappProviderId(),
		deployment.GetWhatsapp().GetWhatsappProviderName(),
		deployment.GetWhatsapp().GetWhatsappOptions(),
	)

	if err != nil {
		return utils.Error[assistant_api.GetAssistantWhatsappDeploymentResponse](
			errors.New("unauthenticated request for create assistant debugger deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	return utils.Success[assistant_api.GetAssistantWhatsappDeploymentResponse](wpDeployment)
}
