package assistant_api

import (
	"context"
	"errors"
	"testing"

	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAssistant_CreateAssistantErrorUsesStandardError(t *testing.T) {
	assistantApi := &assistantGrpcApi{
		assistantApi: assistantApi{
			assistantService: &createAssistantRestAssistantServiceStub{
				createAssistantErr: errors.New("database password leaked"),
			},
		},
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), createAssistantRestAuth())
	require.NoError(t, err)

	response, err := assistantApi.CreateAssistant(ctx, &protos.CreateAssistantRequest{
		Name: "Support Assistant",
		AssistantProvider: &protos.CreateAssistantProviderRequest{
			AssistantProvider: &protos.CreateAssistantProviderRequest_Model{
				Model: &protos.CreateAssistantProviderRequest_CreateAssistantProviderModel{
					ModelProviderName: "openai",
				},
			},
		},
	})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, int32(pkg_errors.CreateAssistantCreateAssistant.HTTPStatusCode), response.Code)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantCreateAssistant.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateAssistantCreateAssistant.Error, response.Error.ErrorMessage)
	assert.Equal(t, pkg_errors.CreateAssistantCreateAssistant.ErrorMessage, response.Error.HumanMessage)
	assert.NotContains(t, response.Error.ErrorMessage, "database password leaked")
	assert.NotContains(t, err.Error(), "database password leaked")
}

func TestCreateAssistant_MissingNameValidation(t *testing.T) {
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantGrpcApi{
		assistantApi: assistantApi{
			assistantService: assistantService,
		},
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), createAssistantRestAuth())
	require.NoError(t, err)

	response, err := assistantApi.CreateAssistant(ctx, &protos.CreateAssistantRequest{
		AssistantProvider: &protos.CreateAssistantProviderRequest{
			AssistantProvider: &protos.CreateAssistantProviderRequest_Model{
				Model: &protos.CreateAssistantProviderRequest_CreateAssistantProviderModel{
					ModelProviderName: "openai",
				},
			},
		},
	})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantMissingName.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateAssistantMissingName.Error, response.Error.ErrorMessage)
	assert.False(t, assistantService.createAssistantCalled)
}

func TestCreateAssistant_MissingProviderValidation(t *testing.T) {
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantGrpcApi{
		assistantApi: assistantApi{
			assistantService: assistantService,
		},
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), createAssistantRestAuth())
	require.NoError(t, err)

	response, err := assistantApi.CreateAssistant(ctx, &protos.CreateAssistantRequest{
		Name: "Support Assistant",
	})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantMissingProvider.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateAssistantMissingProvider.Error, response.Error.ErrorMessage)
	assert.False(t, assistantService.createAssistantCalled)
}

func TestCreateAssistant_MissingModelProviderNameValidation(t *testing.T) {
	assistantService := &createAssistantRestAssistantServiceStub{}
	assistantApi := &assistantGrpcApi{
		assistantApi: assistantApi{
			assistantService: assistantService,
		},
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), createAssistantRestAuth())
	require.NoError(t, err)

	response, err := assistantApi.CreateAssistant(ctx, &protos.CreateAssistantRequest{
		Name: "Support Assistant",
		AssistantProvider: &protos.CreateAssistantProviderRequest{
			AssistantProvider: &protos.CreateAssistantProviderRequest_Model{
				Model: &protos.CreateAssistantProviderRequest_CreateAssistantProviderModel{},
			},
		},
	})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, uint64(pkg_errors.CreateAssistantMissingModelProviderName.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateAssistantMissingModelProviderName.Error, response.Error.ErrorMessage)
	assert.False(t, assistantService.createAssistantCalled)
}
