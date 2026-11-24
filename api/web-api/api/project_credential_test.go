// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package web_api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type projectCredentialServiceStub struct {
	internal_service.ProjectService
	createCredential func(context.Context, authentication.Principal, string, uint64, uint64, uint64) (*internal_entity.ProjectCredential, error)
	getAllCredential func(context.Context, authentication.Principal, uint64, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error)
	revokeCredential func(context.Context, authentication.Principal, uint64, uint64, uint64) (uint64, error)
}

func (stub *projectCredentialServiceStub) CreateCredential(ctx context.Context, auth authentication.Principal, name string, projectID, organizationID, roleDefinitionID uint64) (*internal_entity.ProjectCredential, error) {
	return stub.createCredential(ctx, auth, name, projectID, organizationID, roleDefinitionID)
}

func (stub *projectCredentialServiceStub) GetAllCredential(ctx context.Context, auth authentication.Principal, projectID, organizationID uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
	return stub.getAllCredential(ctx, auth, projectID, organizationID, criterias, paginate)
}

func (stub *projectCredentialServiceStub) RevokeCredential(ctx context.Context, auth authentication.Principal, credentialID, projectID, organizationID uint64) (uint64, error) {
	return stub.revokeCredential(ctx, auth, credentialID, projectID, organizationID)
}

type projectCredentialUserServiceStub struct {
	internal_service.UserService
	role *internal_entity.UserOrganizationRole
	err  error
}

func (stub *projectCredentialUserServiceStub) HasOrganizationPermissions(context.Context, uint64, uint64, ...string) (bool, error) {
	return stub.role != nil && stub.role.Status == type_enums.RECORD_ACTIVE && (stub.role.RoleDefinitionID == 101 || stub.role.RoleDefinitionID == 102), stub.err
}

func projectCredentialContext() context.Context {
	return projectCredentialContextWithTenant(9, 100)
}

func projectCredentialContextWithTenant(organizationID, projectID uint64) context.Context {
	principal, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.Tenant{OrganizationID: organizationID, ProjectID: projectID},
		authentication.Proof{
			Method:    authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{Type: authentication.ProofReferenceUserSession, Value: "test-session"},
		},
		[]authentication.PermissionKey{"project.credential.manage"},
	)
	if err != nil {
		panic(err)
	}
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	return ctx
}

func TestCreateProjectCredentialRequiresAndForwardsRole(t *testing.T) {
	callCount := 0
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
		createCredential: func(_ context.Context, _ authentication.Principal, name string, projectID, organizationID, roleDefinitionID uint64) (*internal_entity.ProjectCredential, error) {
			callCount++
			require.Equal(t, "publishable key", name)
			require.Equal(t, uint64(100), projectID)
			require.Equal(t, uint64(9), organizationID)
			require.Equal(t, uint64(117), roleDefinitionID)
			return &internal_entity.ProjectCredential{
				Name: name, Key: "rpx_secret", RoleDefinitionID: roleDefinitionID, RoleSubjectType: "PROJECT_TOKEN",
				RoleDefinition: &internal_entity.RoleDefinition{Name: "Runtime"},
			}, nil
		},
	}}}

	tests := []struct {
		name          string
		request       *protos.CreateProjectCredentialRequest
		platformError pkg_errors.PlatformError
	}{
		{name: "project is validated first", request: &protos.CreateProjectCredentialRequest{Name: " ", RoleDefinitionId: 0}, platformError: pkg_errors.CreateProjectCredentialInvalidProject},
		{name: "project exceeds signed bigint", request: &protos.CreateProjectCredentialRequest{ProjectId: uint64(math.MaxInt64) + 1, Name: "publishable key", RoleDefinitionId: 117}, platformError: pkg_errors.CreateProjectCredentialInvalidProject},
		{name: "trimmed name is validated before role", request: &protos.CreateProjectCredentialRequest{ProjectId: 100, Name: " \t "}, platformError: pkg_errors.CreateProjectCredentialMissingName},
		{name: "role is required", request: &protos.CreateProjectCredentialRequest{ProjectId: 100, Name: "publishable key"}, platformError: pkg_errors.CreateProjectCredentialInvalidRole},
		{name: "role exceeds signed bigint", request: &protos.CreateProjectCredentialRequest{ProjectId: 100, Name: "publishable key", RoleDefinitionId: uint64(math.MaxInt64) + 1}, platformError: pkg_errors.CreateProjectCredentialInvalidRole},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := api.CreateProjectCredential(projectCredentialContext(), test.request)
			require.NoError(t, err)
			require.False(t, response.GetSuccess())
			require.Nil(t, response.GetData())
			require.Equal(t, test.platformError.HTTPStatusCodeInt32(), response.GetCode())
			require.EqualValues(t, test.platformError.Code, response.GetError().GetErrorCode())
			require.Equal(t, test.platformError.Error, response.GetError().GetErrorMessage())
			require.Equal(t, test.platformError.ErrorMessage, response.GetError().GetHumanMessage())
		})
	}
	require.Zero(t, callCount)

	response, err := api.CreateProjectCredential(projectCredentialContext(), &protos.CreateProjectCredentialRequest{
		ProjectId: 100, Name: "  publishable key  ", RoleDefinitionId: 117,
	})
	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	require.Equal(t, int32(200), response.GetCode())
	require.True(t, response.GetSuccess())
	require.Equal(t, "rpx_secret", response.GetData().GetKey())
	require.Equal(t, uint64(117), response.GetData().GetRoleDefinitionId())
	require.Equal(t, "Runtime", response.GetData().GetRoleName())
}

func TestGetAllProjectCredentialValidatesAndMapsRoleMetadata(t *testing.T) {
	createdDate := gorm_models.TimeWrapper(time.Date(2026, time.November, 9, 10, 0, 0, 0, time.UTC))
	service := &projectCredentialServiceStub{
		getAllCredential: func(_ context.Context, _ authentication.Principal, projectID, organizationID uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
			require.Equal(t, uint64(100), projectID)
			require.Equal(t, uint64(9), organizationID)
			require.Equal(t, []*protos.Criteria{{Key: "role", Logic: "contains", Value: "runtime"}}, criterias)
			require.Equal(t, uint32(1), paginate.GetPage())
			require.Equal(t, uint32(25), paginate.GetPageSize())
			return 1, []*internal_entity.ProjectCredential{{
				Audited:          gorm_models.Audited{Id: 31, CreatedDate: createdDate},
				Organizational:   gorm_models.Organizational{ProjectId: 100, OrganizationId: 9},
				Name:             "Web widget",
				Key:              "rpx_secret",
				RoleDefinitionID: 117,
				Mutable:          gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE, CreatedActorType: string(authentication.IdentityUser), CreatedActorID: 7},
				RoleDefinition:   &internal_entity.RoleDefinition{Name: "Runtime"},
			}}, nil
		},
	}
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: service}}
	response, err := api.GetAllProjectCredential(projectCredentialContext(), &protos.GetAllProjectCredentialRequest{
		ProjectId: 100,
		Criterias: []*protos.Criteria{{Key: "role", Logic: "contains", Value: "  runtime  "}},
		Paginate:  &protos.Paginate{},
	})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, uint32(1), response.GetPaginated().GetCurrentPage())
	require.Equal(t, uint32(1), response.GetPaginated().GetTotalItem())
	require.Len(t, response.GetData(), 1)
	require.Equal(t, "rpx_secret", response.GetData()[0].GetKey())
	require.Equal(t, uint64(117), response.GetData()[0].GetRoleDefinitionId())
	require.Equal(t, "Runtime", response.GetData()[0].GetRoleName())
	require.Equal(t, uint64(7), response.GetData()[0].GetCreatedActor().GetId())
	require.Equal(t, createdDate, gorm_models.TimeWrapper(response.GetData()[0].GetCreatedDate().AsTime()))
}

func TestGetAllProjectCredentialRejectsInvalidRequests(t *testing.T) {
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
		getAllCredential: func(context.Context, authentication.Principal, uint64, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
			t.Fatal("GetAllCredential() should not be called")
			return 0, nil, nil
		},
	}}}
	tests := []struct {
		name          string
		request       *protos.GetAllProjectCredentialRequest
		platformError pkg_errors.PlatformError
	}{
		{name: "invalid project", request: &protos.GetAllProjectCredentialRequest{}, platformError: pkg_errors.ListProjectAPIKeyInvalidProject},
		{name: "partial pagination", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Paginate: &protos.Paginate{Page: 1}}, platformError: pkg_errors.ListProjectAPIKeyInvalidPagination},
		{name: "oversized page", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Paginate: &protos.Paginate{Page: 1, PageSize: 101}}, platformError: pkg_errors.ListProjectAPIKeyInvalidPagination},
		{name: "nil criteria", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{nil}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "unsupported key", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "key", Logic: "contains", Value: "secret"}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "unsupported logic", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "=", Value: "widget"}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "duplicate key", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "widget"}, {Key: "name", Logic: "contains", Value: "phone"}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "blank value", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "  "}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "nul value", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "web\x00widget"}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "long value", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "contains", Value: strings.Repeat("界", 201)}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
		{name: "too many criteria", request: &protos.GetAllProjectCredentialRequest{ProjectId: 100, Criterias: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "web"}, {Key: "role", Logic: "contains", Value: "runtime"}, {Key: "extra", Logic: "contains", Value: "value"}}}, platformError: pkg_errors.ListProjectAPIKeyInvalidCriteria},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := api.GetAllProjectCredential(projectCredentialContext(), test.request)
			require.NoError(t, err)
			require.False(t, response.GetSuccess())
			require.Equal(t, test.platformError.HTTPStatusCodeInt32(), response.GetCode())
			require.EqualValues(t, test.platformError.Code, response.GetError().GetErrorCode())
		})
	}
}

func TestGetAllProjectCredentialPreservesOmittedPagination(t *testing.T) {
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
		getAllCredential: func(_ context.Context, _ authentication.Principal, _ uint64, _ uint64, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
			require.Empty(t, criterias)
			require.Nil(t, paginate)
			return 0, nil, nil
		},
	}}}
	response, err := api.GetAllProjectCredential(projectCredentialContext(), &protos.GetAllProjectCredentialRequest{ProjectId: 100})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, uint32(1), response.GetPaginated().GetCurrentPage())
}

func TestGetAllProjectCredentialMapsTransportFailures(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		code       codes.Code
	}{
		{name: "canceled", serviceErr: context.Canceled, code: codes.Canceled},
		{name: "deadline", serviceErr: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "internal", serviceErr: errors.New("database detail"), code: codes.Internal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
				getAllCredential: func(context.Context, authentication.Principal, uint64, uint64, []*protos.Criteria, *protos.Paginate) (int64, []*internal_entity.ProjectCredential, error) {
					return 0, nil, test.serviceErr
				},
			}}}
			response, err := api.GetAllProjectCredential(projectCredentialContext(), &protos.GetAllProjectCredentialRequest{ProjectId: 100})
			require.Nil(t, response)
			require.Equal(t, test.code, status.Code(err))
		})
	}
}

func TestProjectCredentialCriteriaDescriptor(t *testing.T) {
	descriptor := (&protos.GetAllProjectCredentialRequest{}).ProtoReflect().Descriptor()
	criterias := descriptor.Fields().ByName("criterias")
	require.NotNil(t, criterias)
	require.Equal(t, protoreflect.FieldNumber(2), criterias.Number())
	require.True(t, criterias.IsList())
	require.Nil(t, descriptor.Fields().ByName("query"))
	for index := 0; index < descriptor.ReservedRanges().Len(); index++ {
		reserved := descriptor.ReservedRanges().Get(index)
		require.False(t, protoreflect.FieldNumber(4) >= reserved[0] && protoreflect.FieldNumber(4) < reserved[1])
	}
	for index := 0; index < descriptor.ReservedNames().Len(); index++ {
		require.NotEqual(t, protoreflect.Name("query"), descriptor.ReservedNames().Get(index))
	}
}

func TestRevokeProjectCredentialUsesRouteAuthorizedPrincipalAndIsSecretFree(t *testing.T) {
	service := &projectCredentialServiceStub{
		revokeCredential: func(_ context.Context, _ authentication.Principal, credentialID, projectID, organizationID uint64) (uint64, error) {
			require.Equal(t, uint64(31), credentialID)
			require.Equal(t, uint64(100), projectID)
			require.Equal(t, uint64(9), organizationID)
			return credentialID, nil
		},
	}
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: service}}
	response, err := api.RevokeProjectCredential(projectCredentialContext(), &protos.RevokeProjectCredentialRequest{ProjectId: 100, CredentialId: 31})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, uint64(31), response.GetId())
	require.Nil(t, response.GetError())
}

func TestRevokeProjectCredentialMapsValidationAndServiceFailures(t *testing.T) {
	role := &internal_entity.UserOrganizationRole{RoleDefinitionID: 101}
	role.Status = type_enums.RECORD_ACTIVE
	tests := []struct {
		name          string
		request       *protos.RevokeProjectCredentialRequest
		serviceErr    error
		platformError *pkg_errors.PlatformError
		grpcCode      codes.Code
	}{
		{name: "invalid project", request: &protos.RevokeProjectCredentialRequest{CredentialId: 31}, platformError: &pkg_errors.RevokeProjectAPIKeyInvalidProject},
		{name: "invalid credential", request: &protos.RevokeProjectCredentialRequest{ProjectId: 100}, platformError: &pkg_errors.RevokeProjectAPIKeyInvalidCredential},
		{name: "hidden credential", request: &protos.RevokeProjectCredentialRequest{ProjectId: 100, CredentialId: 31}, serviceErr: fmt.Errorf("wrapped: %w", pkg_errors.ProjectAPIKeyNotFoundError{}), platformError: &pkg_errors.RevokeProjectAPIKeyNotFound},
		{name: "canceled", request: &protos.RevokeProjectCredentialRequest{ProjectId: 100, CredentialId: 31}, serviceErr: context.Canceled, grpcCode: codes.Canceled},
		{name: "deadline", request: &protos.RevokeProjectCredentialRequest{ProjectId: 100, CredentialId: 31}, serviceErr: context.DeadlineExceeded, grpcCode: codes.DeadlineExceeded},
		{name: "internal", request: &protos.RevokeProjectCredentialRequest{ProjectId: 100, CredentialId: 31}, serviceErr: errors.New("database detail"), grpcCode: codes.Internal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webProjectGRPCApi{webProjectApi: webProjectApi{
				userService: &projectCredentialUserServiceStub{role: role},
				projectService: &projectCredentialServiceStub{revokeCredential: func(context.Context, authentication.Principal, uint64, uint64, uint64) (uint64, error) {
					return 0, test.serviceErr
				}},
			}}
			response, err := api.RevokeProjectCredential(projectCredentialContext(), test.request)
			if test.platformError != nil {
				require.NoError(t, err)
				require.False(t, response.GetSuccess())
				require.EqualValues(t, test.platformError.Code, response.GetError().GetErrorCode())
				return
			}
			require.Nil(t, response)
			require.Equal(t, test.grpcCode, status.Code(err))
		})
	}
}

func TestCreateProjectCredentialAcceptsSignedBigIntBoundaries(t *testing.T) {
	tests := []struct {
		name string
		id   uint64
	}{
		{name: "minimum", id: 1},
		{name: "maximum", id: math.MaxInt64},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callCount := 0
			api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
				createCredential: func(_ context.Context, _ authentication.Principal, name string, projectID, organizationID, roleDefinitionID uint64) (*internal_entity.ProjectCredential, error) {
					callCount++
					require.Equal(t, "publishable key", name)
					require.Equal(t, test.id, projectID)
					require.Equal(t, test.id, organizationID)
					require.Equal(t, test.id, roleDefinitionID)
					return &internal_entity.ProjectCredential{
						Name: name, Key: "rpx_secret", RoleDefinitionID: roleDefinitionID, RoleSubjectType: "PROJECT_TOKEN",
						RoleDefinition: &internal_entity.RoleDefinition{Name: "Runtime"},
					}, nil
				},
			}}}

			response, err := api.CreateProjectCredential(
				projectCredentialContextWithTenant(test.id, test.id),
				&protos.CreateProjectCredentialRequest{ProjectId: test.id, Name: "  publishable key  ", RoleDefinitionId: test.id},
			)
			require.NoError(t, err)
			require.True(t, response.GetSuccess())
			require.Equal(t, int32(200), response.GetCode())
			require.Equal(t, 1, callCount)
		})
	}
}

func TestCreateProjectCredentialRejectsMissingPrincipalBeforeService(t *testing.T) {
	called := false
	api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
		createCredential: func(context.Context, authentication.Principal, string, uint64, uint64, uint64) (*internal_entity.ProjectCredential, error) {
			called = true
			return nil, nil
		},
	}}}

	response, err := api.CreateProjectCredential(
		context.Background(),
		&protos.CreateProjectCredentialRequest{ProjectId: 100, Name: "publishable key", RoleDefinitionId: 117},
	)
	require.Nil(t, response)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.False(t, called)
}

func TestCreateProjectCredentialMapsSelectionAndTransportFailures(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		code       codes.Code
		envelope   bool
	}{
		{name: "hidden selection", serviceErr: fmt.Errorf("wrapped selection: %w", pkg_errors.CredentialSelectionNotFoundError{}), code: codes.OK, envelope: true},
		{name: "canceled", serviceErr: context.Canceled, code: codes.Canceled},
		{name: "deadline", serviceErr: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "internal", serviceErr: errors.New("database detail"), code: codes.Internal},
		{name: "defensive invalid organization", serviceErr: fmt.Errorf("wrapped organization: %w", pkg_errors.CredentialOrganizationInvalidError{}), code: codes.Internal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &webProjectGRPCApi{webProjectApi: webProjectApi{projectService: &projectCredentialServiceStub{
				createCredential: func(context.Context, authentication.Principal, string, uint64, uint64, uint64) (*internal_entity.ProjectCredential, error) {
					return nil, test.serviceErr
				},
			}}}
			response, err := api.CreateProjectCredential(projectCredentialContext(), &protos.CreateProjectCredentialRequest{
				ProjectId: 100, Name: "publishable key", RoleDefinitionId: 117,
			})
			require.Equal(t, test.code, status.Code(err))
			if test.envelope {
				require.NotNil(t, response)
				require.False(t, response.GetSuccess())
				require.Nil(t, response.GetData())
				require.Equal(t, pkg_errors.CreateProjectCredentialSelectionNotFound.HTTPStatusCodeInt32(), response.GetCode())
				require.EqualValues(t, pkg_errors.CreateProjectCredentialSelectionNotFound.Code, response.GetError().GetErrorCode())
				require.Equal(t, pkg_errors.CreateProjectCredentialSelectionNotFound.Error, response.GetError().GetErrorMessage())
				require.Equal(t, pkg_errors.CreateProjectCredentialSelectionNotFound.ErrorMessage, response.GetError().GetHumanMessage())
			} else {
				require.Nil(t, response)
			}
		})
	}
}
