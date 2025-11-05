package integration_api

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"time"

// 	config "github.com/rapidaai/api/integration-api/config"
// 	callers "github.com/rapidaai/api/integration-api/internal/caller"
// 	commons "github.com/rapidaai/pkg/commons"
// 	"github.com/rapidaai/pkg/connectors"
// 	provider_models "github.com/rapidaai/pkg/providers"
// 	"github.com/rapidaai/pkg/types"
// 	"github.com/rapidaai/pkg/utils"
// 	integration_api "github.com/rapidaai/protos"
// )

// type deepInfraIntegrationApi struct {
// 	integrationApi
// 	caller callers.Caller
// }

// type deepInfraIntegrationRPCApi struct {
// 	deepInfraIntegrationApi
// }

// type deepInfraIntegrationGRPCApi struct {
// 	deepInfraIntegrationApi
// }

// func NewDeepInfraRPCApi(config *config.IntegrationConfig, logger commons.Logger, caller callers.Caller, postgres connectors.PostgresConnector) *deepInfraIntegrationRPCApi {
// 	return &deepInfraIntegrationRPCApi{
// 		deepInfraIntegrationApi{
// 			integrationApi: NewInegrationApi(config, logger, postgres),
// 			caller:         caller,
// 		},
// 	}
// }

// func NewDeepInfraGRPC(config *config.IntegrationConfig, logger commons.Logger, caller callers.Caller, postgres connectors.PostgresConnector) integration_api.DeepInfraServiceServer {
// 	return &deepInfraIntegrationGRPCApi{
// 		deepInfraIntegrationApi{
// 			integrationApi: NewInegrationApi(config, logger, postgres),
// 			caller:         caller,
// 		},
// 	}
// }

// func (oiGRPC *deepInfraIntegrationGRPCApi) VerifyCredential(context.Context, *integration_api.VerifyCredentialRequest) (*integration_api.VerifyCredentialResponse, error) {
// 	return &integration_api.VerifyCredentialResponse{
// 		Code:    200,
// 		Success: true,
// 	}, nil
// }

// func (oiGRPC *deepInfraIntegrationGRPCApi) constructGenerateImageBody(irRequest *integration_api.GenerateTextToImageRequest) map[string]interface{} {
// 	input := map[string]interface{}{
// 		"prompt": irRequest.GetPrompt(),
// 		"model":
// 	}

// 	// construct parameter

// 	input = oiGRPC.ConstructParameter(irRequest.GetModelParameters(), input)

// 	if provider_models.IsDeepInfraV2ImageModel(irRequest.GetModel()) {
// 		requestBody := map[string]interface{}{
// 			"input": input,
// 		}
// 		return requestBody
// 	}
// 	return input

// }
// func (oiGRPC *deepInfraIntegrationGRPCApi) GenerateTextToImage(c context.Context, irRequest *integration_api.GenerateTextToImageRequest) (*integration_api.GenerateTextToImageResponse, error) {

// 	iAuth, isAuthenticated := types.GetClaimPrincipleGRPC[*types.ServiceScope](c)
// 	if !isAuthenticated || !iAuth.HasProject() {
// 		oiGRPC.logger.Errorf("unauthenticated request for invoke")
// 		return utils.Error[integration_api.GenerateTextToImageResponse](
// 			errors.New("unauthenticated request for text to image"),
// 			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
// 		)
// 	}

// 	oiGRPC.logger.Debugf("request for image generate deepInfra with request %+v", irRequest)
// 	headers := map[string]string{
// 		"Authorization": fmt.Sprintf("Bearer %s", irRequest.Credential.Value),
// 	}

// 	for _, param := range irRequest.ModelParameters {
// 		if param.Type == "header" {
// 			headers[param.Key] = param.Value
// 		}
// 	}

// 	requestBody := oiGRPC.constructGenerateImageBody(irRequest)
// 	adt := oiGRPC.PreHook(c, iAuth, irRequest.GetAdditionalData(), irRequest.GetCredential().GetId(), "DEEP_INFRA", requestBody)
// 	// only calculate api call
// 	start := time.Now()
// 	res, err := oiGRPC.caller.Call(c, fmt.Sprintf("/v1/inference/%s", irRequest.GetModel()), "POST", headers, requestBody)
// 	timeTaken := int64(time.Since(start))

// 	if err == nil {
// 		oiGRPC.PostHook(c, iAuth, irRequest.GetAdditionalData(), irRequest.GetCredential().GetId(), adt, 200, timeTaken, res)
// 		return &integration_api.GenerateTextToImageResponse{
// 			Code:      200,
// 			Success:   true,
// 			Response:  res,
// 			RequestId: adt,
// 			TimeTaken: timeTaken,
// 		}, nil
// 	}

// 	oiGRPC.logger.Debugf("Exception occurred while calling completions %v", err)
// 	ex, ok := err.(callers.DeepInfraError)
// 	// can be used as defer
// 	if ok {
// 		errMessage := ex.Error()
// 		oiGRPC.PostHook(c, iAuth, irRequest.GetAdditionalData(), irRequest.GetCredential().GetId(), adt, 400, timeTaken, &errMessage)
// 		return &integration_api.GenerateTextToImageResponse{
// 			Code:    500,
// 			Success: false,
// 			Error: &integration_api.Error{
// 				ErrorCode:    500,
// 				ErrorMessage: errMessage,
// 				HumanMessage: ex.Detail.Error,
// 			},
// 			RequestId: adt,
// 			TimeTaken: timeTaken,
// 		}, nil

// 	}
// 	return utils.Error[integration_api.GenerateTextToImageResponse](errors.New("illegal token while processing request"), "Illegal request, please try again")
// }
