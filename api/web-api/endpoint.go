package web_api

import (
	"context"
	"errors"

	endpoint_client "github.com/lexatic/web-backend/pkg/clients/endpoint"
	testing_client "github.com/lexatic/web-backend/pkg/clients/testing"
	"github.com/lexatic/web-backend/pkg/utils"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webEndpointApi struct {
	WebApi
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	redis          connectors.RedisConnector
	endpointClient endpoint_client.EndpointServiceClient
	testingClient  testing_client.TestingServiceClient
}

type webEndpointGRPCApi struct {
	webEndpointApi
}

func NewEndpointGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector, redis connectors.RedisConnector) web_api.EndpointServiceServer {
	return &webEndpointGRPCApi{
		webEndpointApi{
			WebApi:         NewWebApi(config, logger, postgres, redis),
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			redis:          redis,
			endpointClient: endpoint_client.NewEndpointServiceClientGRPC(config, logger, redis),
			testingClient:  testing_client.NewTestingServiceClientGRPC(config, logger, redis),
		},
	}
}

func (endpoint *webEndpointGRPCApi) GetEndpoint(c context.Context, iRequest *web_api.GetEndpointRequest) (*web_api.GetEndpointResponse, error) {
	endpoint.logger.Debugf("GetEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	_endpoint, err := endpoint.endpointClient.GetEndpoint(c, iAuth, iRequest)
	if err != nil {
		return utils.Error[web_api.GetEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	if _endpoint.EndpointProviderModel != nil {
		_endpoint.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _endpoint.EndpointProviderModel.GetCreatedBy())
		_endpoint.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _endpoint.EndpointProviderModel.GetProviderModelId())
	}

	return utils.Success[web_api.GetEndpointResponse, *web_api.Endpoint](_endpoint)

}

/*
 */

func (endpoint *webEndpointGRPCApi) GetAllPublicEndpoint(c context.Context, iRequest *web_api.GetAllEndpointRequest) (*web_api.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoint, err := endpoint.endpointClient.GetAllPublicEndpoint(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	for _, _ep := range _endpoint {
		if _ep.GetEndpointProviderModel() != nil {
			_ep.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _ep.EndpointProviderModel.GetCreatedBy())
			_ep.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _ep.EndpointProviderModel.GetProviderModelId())
		}
		_ep.Organization = endpoint.GetOrganization(c, iAuth, _ep.GetOrganizationId())
	}
	return utils.PaginatedSuccess[web_api.GetAllEndpointResponse, []*web_api.Endpoint](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

/*
 */
func (endpoint *webEndpointGRPCApi) GetAllEndpoint(c context.Context, iRequest *web_api.GetAllEndpointRequest) (*web_api.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	_page, _endpoint, err := endpoint.endpointClient.GetAllEndpoint(c, iAuth, iRequest.GetCriterias(), iRequest.GetPaginate())
	if err != nil {
		return utils.Error[web_api.GetAllEndpointResponse](
			err,
			"Unable to get your endpoint, please try again in sometime.")
	}

	for _, _ep := range _endpoint {
		if _ep.GetEndpointProviderModel() != nil {
			_ep.EndpointProviderModel.CreatedUser = endpoint.GetUser(c, iAuth, _ep.EndpointProviderModel.GetCreatedBy())
			_ep.EndpointProviderModel.ProviderModel = endpoint.GetProviderModel(c, iAuth, _ep.EndpointProviderModel.GetProviderModelId())
		}
	}
	return utils.PaginatedSuccess[web_api.GetAllEndpointResponse, []*web_api.Endpoint](
		_page.GetTotalItem(), _page.GetCurrentPage(),
		_endpoint)
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iAuth, iRequest)
}

// func (endpoint *webEndpointGRPCApi) CreateEndpointFromTestcase(c context.Context, iRequest *web_api.CreateEndpointFromTestcaseRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
// 	endpoint.logger.Debugf("Create endpoint from test case grpc with requestPayload %v, %v", iRequest, c)

// 	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
// 	if !isAuthenticated {
// 		endpoint.logger.Errorf("unauthenticated request for creating endpoint")
// 		return nil, errors.New("unauthenticated request")
// 	}

// 	res, err := endpoint.testingClient.GetTestSuite(c, iRequest.TestsuiteId)

// 	if err != nil || res.GetData() == nil || !res.Success {
// 		return &web_api.CreateEndpointProviderModelResponse{Code: 400, Success: false, Error: &web_api.Error{
// 			ErrorCode:    400,
// 			ErrorMessage: err.Error(),
// 			HumanMessage: "unable to create test suite from endpoint",
// 		}}, nil
// 	}

// 	var tc *web_api.TestSuiteCase

// 	tS := res.GetData()
// 	for _, testcase := range tS.GetTestsuiteCases() {
// 		if testcase.GetId() == iRequest.GetTestcaseId() {
// 			tc = testcase
// 			break
// 		}
// 	}

// 	if tc == nil {
// 		return &web_api.CreateEndpointProviderModelResponse{Code: 400, Success: false, Error: &web_api.Error{
// 			ErrorCode:    400,
// 			ErrorMessage: "unable to locate test suite to create test",
// 			HumanMessage: "unable to create test suite from endpoint",
// 		}}, nil
// 	}

// 	epName := fmt.Sprintf("endpoint-%s", tS.GetName())
// 	sysPrompt := tc.GetSystemPrompt()
// 	description := tS.GetDescription()

// 	epmp := make([]*web_api.EndpointProviderModelParameter, len(tc.TestCaseModelParameters))
// 	epmv := make([]*web_api.EndpointProviderModelVariable, len(tS.GetVariables()))

// 	for i, param := range tc.GetTestCaseModelParameters() {
// 		epmp[i] = &web_api.EndpointProviderModelParameter{
// 			ProviderModelVariableId: param.GetProviderModelVariableId(),
// 			Value:                   param.Value,
// 		}
// 	}

// 	for i, variable := range tS.GetVariables() {
// 		epmv[i] = &web_api.EndpointProviderModelVariable{
// 			Name:         variable,
// 			Type:         "any",
// 			DefaultValue: new(string),
// 		}
// 	}

// 	cer := &web_api.CreateEndpointRequest{EndpointAttributes: &web_api.EndpointAttributes{
// 		Name:                            &epName,
// 		CreatedBy:                       iAuth.GetUserInfo().Id,
// 		GlobalPrompt:                    tS.GetGlobalPrompt(),
// 		SystemPrompt:                    &sysPrompt,
// 		ProviderModelId:                 tc.GetProviderModelId(),
// 		Description:                     &description,
// 		EndpointProviderModelParameters: epmp,
// 		EndpointProviderModelVariable:   epmv,
// 	}, Endpoint: &web_api.EndpointParameter{
// 		ProjectId:        tS.GetProjectId(),
// 		OrganizationId:   iRequest.GetOrganizationId(),
// 		EndpointSource:   *web_api.EndpointSource_TEST_CASE.Enum(),
// 		SourceIdentifier: &tc.Id,
// 		Type:             tS.GetType(),
// 	}}
// 	return endpoint.endpointClient.CreateEndpoint(c, cer, tS.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
// }

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *web_api.GetAllEndpointProviderModelRequest) (*web_api.GetAllEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetCriterias(), iRequest.GetPaginate())
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *web_api.UpdateEndpointVersionRequest) (*web_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("Update endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, iAuth, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId())
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *web_api.CreateEndpointProviderModelRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint provider model")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, iAuth, iRequest)
}

// CreateEndpointCacheConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointCacheConfiguration(ctx context.Context, iRequest *web_api.CreateEndpointCacheConfigurationRequest) (*web_api.CreateEndpointCacheConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint caching configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointCacheConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointRetryConfiguration implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointRetryConfiguration(ctx context.Context, iRequest *web_api.CreateEndpointRetryConfigurationRequest) (*web_api.CreateEndpointRetryConfigurationResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint retry configuration")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointRetryConfiguration(ctx, iAuth, iRequest)
}

// CreateEndpointTag implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointTag(ctx context.Context, iRequest *web_api.CreateEndpointTagRequest) (*web_api.CreateEndpointTagResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint tag")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointTag(ctx, iAuth, iRequest)
}

// ForkEndpoint implements lexatic_backend.EndpointServiceServer.
func (endpointGRPCApi *webEndpointGRPCApi) ForkEndpoint(ctx context.Context, iRequest *web_api.ForkEndpointRequest) (*web_api.BaseResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, iAuth, iRequest)
}

func (endpointGRPCApi *webEndpointGRPCApi) Invoke(ctx context.Context, iRequest *web_api.ForkEndpointRequest) (*web_api.BaseResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to fork endpoint")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.ForkEndpoint(ctx, iAuth, iRequest)
}
