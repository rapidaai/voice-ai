package web_api

import (
	"context"
	"errors"
	"fmt"

	clients "github.com/lexatic/web-backend/pkg/clients"
	endpoint_client "github.com/lexatic/web-backend/pkg/clients/endpoint"
	testing_client "github.com/lexatic/web-backend/pkg/clients/testing"
	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"

	config "github.com/lexatic/web-backend/config"
	commons "github.com/lexatic/web-backend/pkg/commons"
	"github.com/lexatic/web-backend/pkg/connectors"
	"github.com/lexatic/web-backend/pkg/types"
)

type webEndpointApi struct {
	cfg            *config.AppConfig
	logger         commons.Logger
	postgres       connectors.PostgresConnector
	endpointClient clients.EndpointServiceClient
	testingClient  clients.TestingServiceClient
}

type webEndpointGRPCApi struct {
	webEndpointApi
}

func NewEndpointGRPC(config *config.AppConfig, logger commons.Logger, postgres connectors.PostgresConnector) web_api.EndpointReaderServiceServer {
	return &webEndpointGRPCApi{
		webEndpointApi{
			cfg:            config,
			logger:         logger,
			postgres:       postgres,
			endpointClient: endpoint_client.NewEndpointServiceClientGRPC(config, logger),
			testingClient:  testing_client.NewTestingServiceClientGRPC(config, logger),
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
	return endpoint.endpointClient.GetEndpoint(c, iRequest.GetId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

func (endpoint *webEndpointGRPCApi) GetAllEndpoint(c context.Context, iRequest *web_api.GetAllEndpointRequest) (*web_api.GetAllEndpointResponse, error) {
	endpoint.logger.Debugf("GetAllEndpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return endpoint.endpointClient.GetAllEndpoint(c, iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iRequest.GetCriterias(), iRequest.GetPaginate())
}

func (endpoint *webEndpointGRPCApi) CreateEndpoint(c context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpoint.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, c)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpoint.endpointClient.CreateEndpoint(c, iRequest, iRequest.GetEndpoint().GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}

func (endpoint *webEndpointGRPCApi) CreateEndpointFromTestcase(c context.Context, iRequest *web_api.CreateEndpointFromTestcaseRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpoint.logger.Debugf("Create endpoint from test case grpc with requestPayload %v, %v", iRequest, c)

	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(c)
	if !isAuthenticated {
		endpoint.logger.Errorf("unauthenticated request for creating endpoint")
		return nil, errors.New("unauthenticated request")
	}

	res, err := endpoint.testingClient.GetTestSuite(c, iRequest.TestsuiteId)

	if err != nil || res.GetData() == nil || !res.Success {
		return &web_api.CreateEndpointProviderModelResponse{Code: 400, Success: false, Error: &web_api.Error{
			ErrorCode:    400,
			ErrorMessage: err.Error(),
			HumanMessage: "unable to create test suite from endpoint",
		}}, nil
	}

	var tc *web_api.TestSuiteCase

	tS := res.GetData()
	for _, testcase := range tS.GetTestsuiteCases() {
		if testcase.GetId() == iRequest.GetTestcaseId() {
			tc = testcase
			break
		}
	}

	if tc == nil {
		return &web_api.CreateEndpointProviderModelResponse{Code: 400, Success: false, Error: &web_api.Error{
			ErrorCode:    400,
			ErrorMessage: "unable to locate test suite to create test",
			HumanMessage: "unable to create test suite from endpoint",
		}}, nil
	}

	epName := fmt.Sprintf("endpoint-%s", tS.GetName())
	sysPrompt := tc.GetSystemPrompt()
	description := tS.GetDescription()

	epmp := make([]*web_api.EndpointProviderModelParameter, len(tc.TestCaseModelParameters))
	epmv := make([]*web_api.EndpointProviderModelVariable, len(tS.GetVariables()))

	for i, param := range tc.GetTestCaseModelParameters() {
		epmp[i] = &web_api.EndpointProviderModelParameter{
			ProviderModelVariableId: param.GetProviderModelVariableId(),
			Value:                   param.Value,
		}
	}

	for i, variable := range tS.GetVariables() {
		epmv[i] = &web_api.EndpointProviderModelVariable{
			Name:         variable,
			Type:         "any",
			DefaultValue: new(string),
		}
	}

	cer := &web_api.CreateEndpointRequest{EndpointAttributes: &web_api.EndpointAttributes{
		Name:                            &epName,
		CreatedBy:                       iAuth.GetUserInfo().Id,
		GlobalPrompt:                    tS.GetGlobalPrompt(),
		SystemPrompt:                    &sysPrompt,
		ProviderModelId:                 tc.GetProviderModelId(),
		Description:                     &description,
		EndpointProviderModelParameters: epmp,
		EndpointProviderModelVariable:   epmv,
	}, Endpoint: &web_api.EndpointParameter{
		ProjectId:        tS.GetProjectId(),
		OrganizationId:   iRequest.GetOrganizationId(),
		EndpointSource:   *web_api.EndpointSource_TEST_CASE.Enum(),
		SourceIdentifier: &tc.Id,
		Type:             tS.GetType(),
	}}
	return endpoint.endpointClient.CreateEndpoint(c, cer, tS.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}

func (endpointGRPCApi *webEndpointGRPCApi) GetAllEndpointProviderModel(ctx context.Context, iRequest *web_api.GetAllEndpointProviderModelRequest) (*web_api.GetAllEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}

	return endpointGRPCApi.endpointClient.GetAllEndpointProviderModel(ctx, iRequest.GetEndpointId(), iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iRequest.GetCriterias(), iRequest.GetPaginate())
}

func (endpointGRPCApi *webEndpointGRPCApi) UpdateEndpointVersion(ctx context.Context, iRequest *web_api.UpdateEndpointVersionRequest) (*web_api.UpdateEndpointVersionResponse, error) {
	endpointGRPCApi.logger.Debugf("Update endpoint from grpc with requestPayload %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request for get actvities")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.UpdateEndpointVersion(ctx, iRequest.GetEndpointId(), iRequest.GetEndpointProviderModelId(), iAuth.GetUserInfo().Id, iRequest.GetProjectId(), iAuth.GetOrganizationRole().OrganizationId)
}

func (endpointGRPCApi *webEndpointGRPCApi) CreateEndpointProviderModel(ctx context.Context, iRequest *web_api.CreateEndpointRequest) (*web_api.CreateEndpointProviderModelResponse, error) {
	endpointGRPCApi.logger.Debugf("Create endpoint provider model request %v, %v", iRequest, ctx)
	iAuth, isAuthenticated := types.GetAuthPrincipleGPRC(ctx)
	if !isAuthenticated {
		endpointGRPCApi.logger.Errorf("unauthenticated request to create endpoint provider model")
		return nil, errors.New("unauthenticated request")
	}
	return endpointGRPCApi.endpointClient.CreateEndpointProviderModel(ctx, iRequest, iRequest.GetEndpoint().GetProjectId(), iAuth.GetOrganizationRole().OrganizationId, iAuth.GetUserInfo().Id)
}
