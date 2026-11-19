// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

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

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_service "github.com/rapidaai/api/web-api/internal/service"
	accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fakeAccessManagementService struct {
	permissions         []*internal_entity.PermissionDefinition
	role                *internal_entity.RoleDefinition
	permissionCriterias []*protos.Criteria
	roleCriterias       []*protos.Criteria
	permissionPaginate  *protos.Paginate
	rolePaginate        *protos.Paginate
	reduceCorrelationID string
	reductionPreview    *internal_service.RoleReductionPreview
	err                 error
}

type recordingAccessManagementLogger struct {
	commons.Logger
	message string
}

func (logger *recordingAccessManagementLogger) Errorf(template string, args ...interface{}) {
	logger.message = fmt.Sprintf(template, args...)
}

func (service *fakeAccessManagementService) ListPermissions(_ context.Context, _ authentication.Principal, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.PermissionDefinition, error) {
	service.permissionCriterias = criterias
	service.permissionPaginate = paginate
	return int64(len(service.permissions)), service.permissions, service.err
}
func (service *fakeAccessManagementService) ListRoles(_ context.Context, _ authentication.Principal, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, []*internal_entity.RoleDefinition, error) {
	service.roleCriterias = criterias
	service.rolePaginate = paginate
	return int64(len(service.roleDefinitions())), service.roleDefinitions(), service.err
}
func (service *fakeAccessManagementService) GetRole(context.Context, authentication.Principal, uint64) (*internal_entity.RoleDefinition, error) {
	return service.role, service.err
}
func (service *fakeAccessManagementService) CreateCustomRole(context.Context, authentication.Principal, string, string, string, string, string, []uint64) (*internal_entity.RoleDefinition, error) {
	return service.role, service.err
}
func (service *fakeAccessManagementService) UpdateCustomRole(context.Context, authentication.Principal, uint64, string, string) (*internal_entity.RoleDefinition, error) {
	return service.role, service.err
}
func (service *fakeAccessManagementService) ReduceCustomRolePermissions(_ context.Context, _ authentication.Principal, _ uint64, _ []uint64, _, correlationID, _ string, preview bool) (*internal_entity.RoleDefinition, *internal_service.RoleReductionPreview, error) {
	service.reduceCorrelationID = correlationID
	if preview && service.err == nil {
		if service.reductionPreview != nil {
			return nil, service.reductionPreview, nil
		}
		return nil, &internal_service.RoleReductionPreview{ResultingRole: service.role, ConfirmationDigest: strings.Repeat("a", 64)}, nil
	}
	return service.role, nil, service.err
}
func (service *fakeAccessManagementService) ArchiveCustomRole(context.Context, authentication.Principal, uint64, string, string) error {
	return service.err
}
func (service *fakeAccessManagementService) roleDefinitions() []*internal_entity.RoleDefinition {
	if service.role == nil {
		return nil
	}
	return []*internal_entity.RoleDefinition{service.role}
}

func accessManagementContext(requestID string) context.Context {
	principal, _ := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityUser, ID: 1},
		authentication.Tenant{OrganizationID: 10},
		authentication.Proof{
			Method: authentication.AuthenticationUserSession,
			Reference: authentication.ProofReference{
				Type: authentication.ProofReferenceUserSession, Value: "access-management-controller-test",
			},
		},
		[]authentication.PermissionKey{"organization.roles.read"},
	)
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), principal)
	if err != nil {
		panic(err)
	}
	if requestID != "" {
		ctx = context.WithValue(ctx, types.REQUEST_ID_KEY, requestID)
	}
	return ctx
}

func TestAccessManagementListPermissionsMapsContract(t *testing.T) {
	service := &fakeAccessManagementService{permissions: []*internal_entity.PermissionDefinition{{
		Key: "project.read", ScopeType: accessmanagement.ScopeProject,
		Name: "Project Read", Description: "Allows project.read.",
	}}}
	service.permissions[0].Id = 34
	api := &accessManagementGRPC{service: service}
	response, err := api.ListPermissions(accessManagementContext("request-1"), &protos.ListPermissionsRequest{
		Criterias: []*protos.Criteria{{Key: "scope", Logic: "=", Value: accessmanagement.ScopeProject}},
	})
	require.NoError(t, err)
	require.Len(t, response.GetPermissions(), 1)
	require.Equal(t, int32(200), response.GetCode())
	require.True(t, response.GetSuccess())
	require.Nil(t, response.GetError())
	require.Equal(t, uint32(1), response.GetPaginated().GetCurrentPage())
	require.Equal(t, uint32(1), response.GetPaginated().GetTotalItem())
	require.Equal(t, protos.RoleScope_ROLE_SCOPE_PROJECT, response.GetPermissions()[0].GetScope())
	require.Equal(t, []*protos.Criteria{{Key: "scope", Logic: "=", Value: accessmanagement.ScopeProject}}, service.permissionCriterias)
	require.Equal(t, uint32(1), service.permissionPaginate.GetPage())
	require.Equal(t, uint32(25), service.permissionPaginate.GetPageSize())
}

func TestAccessManagementListRolesMapsCriteria(t *testing.T) {
	definition := &internal_entity.RoleDefinition{Name: "Support", ScopeType: accessmanagement.ScopeProject}
	service := &fakeAccessManagementService{role: definition}
	api := &accessManagementGRPC{service: service}
	response, err := api.ListRoles(accessManagementContext("request-role-list"), &protos.ListRolesRequest{
		Paginate: &protos.Paginate{Page: 2, PageSize: 50},
		Criterias: []*protos.Criteria{
			{Key: "name", Logic: "contains", Value: "  support  "},
			{Key: "scope", Logic: "=", Value: accessmanagement.ScopeProject},
			{Key: "subject", Logic: "=", Value: accessmanagement.SubjectUser},
			{Key: "type", Logic: "=", Value: accessmanagement.DefinitionCustom},
		},
	})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, []*protos.Criteria{
		{Key: "name", Logic: "contains", Value: "support"},
		{Key: "scope", Logic: "=", Value: accessmanagement.ScopeProject},
		{Key: "subject", Logic: "=", Value: accessmanagement.SubjectUser},
		{Key: "type", Logic: "=", Value: accessmanagement.DefinitionCustom},
	}, service.roleCriterias)
	require.Equal(t, uint32(2), service.rolePaginate.GetPage())
	require.Equal(t, uint32(50), service.rolePaginate.GetPageSize())
}

func TestAccessManagementRejectsInvalidCriteriaBeforeService(t *testing.T) {
	tests := []struct {
		name        string
		permissions []*protos.Criteria
		roles       []*protos.Criteria
		platformErr pkg_errors.PlatformError
	}{
		{name: "permission nil", permissions: []*protos.Criteria{nil}, platformErr: pkg_errors.AccessManagementInvalidPermissionCriteria},
		{name: "permission key", permissions: []*protos.Criteria{{Key: "name", Logic: "=", Value: "PROJECT"}}, platformErr: pkg_errors.AccessManagementInvalidPermissionCriteria},
		{name: "permission logic", permissions: []*protos.Criteria{{Key: "scope", Logic: "contains", Value: "PROJECT"}}, platformErr: pkg_errors.AccessManagementInvalidPermissionCriteria},
		{name: "permission value", permissions: []*protos.Criteria{{Key: "scope", Logic: "=", Value: "unknown"}}, platformErr: pkg_errors.AccessManagementInvalidPermissionCriteria},
		{name: "permission count", permissions: []*protos.Criteria{{Key: "scope", Logic: "=", Value: "PROJECT"}, {Key: "scope", Logic: "=", Value: "ORGANIZATION"}}, platformErr: pkg_errors.AccessManagementInvalidPermissionCriteria},
		{name: "role nil", roles: []*protos.Criteria{nil}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role key", roles: []*protos.Criteria{{Key: "unknown", Logic: "contains", Value: "value"}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role logic", roles: []*protos.Criteria{{Key: "name", Logic: "=", Value: "Support"}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role duplicate", roles: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "Support"}, {Key: "name", Logic: "contains", Value: "Sales"}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role blank", roles: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "  "}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role nul", roles: []*protos.Criteria{{Key: "name", Logic: "contains", Value: "bad\x00name"}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role long", roles: []*protos.Criteria{{Key: "name", Logic: "contains", Value: strings.Repeat("界", 201)}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role enum", roles: []*protos.Criteria{{Key: "subject", Logic: "=", Value: "unknown"}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
		{name: "role count", roles: []*protos.Criteria{{}, {}, {}, {}, {}, {}}, platformErr: pkg_errors.AccessManagementInvalidRoleCriteria},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeAccessManagementService{}
			api := &accessManagementGRPC{service: service}
			var err error
			var code int32
			var success bool
			var responseError *protos.Error
			if test.permissions != nil {
				response, responseErr := api.ListPermissions(accessManagementContext("request-invalid"), &protos.ListPermissionsRequest{Criterias: test.permissions})
				err, code, success, responseError = responseErr, response.GetCode(), response.GetSuccess(), response.GetError()
			} else {
				response, responseErr := api.ListRoles(accessManagementContext("request-invalid"), &protos.ListRolesRequest{Criterias: test.roles})
				err, code, success, responseError = responseErr, response.GetCode(), response.GetSuccess(), response.GetError()
			}
			require.NoError(t, err)
			require.False(t, success)
			require.Equal(t, test.platformErr.HTTPStatusCodeInt32(), code)
			require.EqualValues(t, test.platformErr.Code, responseError.GetErrorCode())
			require.Equal(t, test.platformErr.Error, responseError.GetErrorMessage())
			require.Equal(t, test.platformErr.ErrorMessage, responseError.GetHumanMessage())
			require.Nil(t, service.permissionCriterias)
			require.Nil(t, service.roleCriterias)
		})
	}
}

func TestAccessManagementMutationValidationUsesPlatformErrors(t *testing.T) {
	ctx := accessManagementContext("request-validation")
	tests := []struct {
		name          string
		platformError pkg_errors.PlatformError
		invoke        func(*accessManagementGRPC) (int32, bool, *protos.Error, error)
	}{
		{
			name: "get role identifier", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.GetRole(ctx, &protos.GetRoleRequest{})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "get role identifier exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.GetRole(ctx, &protos.GetRoleRequest{RoleDefinitionId: uint64(math.MaxInt64) + 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create scope", platformError: pkg_errors.AccessManagementInvalidRoleScope,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create subject", platformError: pkg_errors.AccessManagementInvalidRoleSubject,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Scope: protos.RoleScope_ROLE_SCOPE_PROJECT})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create API role scope", platformError: pkg_errors.AccessManagementInvalidAPIRoleScope,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Scope: protos.RoleScope_ROLE_SCOPE_ORGANIZATION, Subject: protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create key", platformError: pkg_errors.AccessManagementInvalidRoleKey,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Scope: protos.RoleScope_ROLE_SCOPE_PROJECT, Subject: protos.RoleSubject_ROLE_SUBJECT_USER})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create name", platformError: pkg_errors.AccessManagementInvalidRoleName,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Key: "custom_role", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT, Subject: protos.RoleSubject_ROLE_SUBJECT_USER})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create description", platformError: pkg_errors.AccessManagementInvalidRoleDescription,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Key: "custom_role", Name: "Custom Role", Description: strings.Repeat("x", 401), Scope: protos.RoleScope_ROLE_SCOPE_PROJECT, Subject: protos.RoleSubject_ROLE_SUBJECT_USER})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create permission", platformError: pkg_errors.AccessManagementInvalidPermissionSelection,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Key: "custom_role", Name: "Custom Role", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT, Subject: protos.RoleSubject_ROLE_SUBJECT_USER, PermissionDefinitionIds: []uint64{0}})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "create permission exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidPermissionSelection,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{Key: "custom_role", Name: "Custom Role", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT, Subject: protos.RoleSubject_ROLE_SUBJECT_USER, PermissionDefinitionIds: []uint64{uint64(math.MaxInt64) + 1}})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "update identifier", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.UpdateCustomRole(ctx, &protos.UpdateCustomRoleRequest{})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "update identifier exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.UpdateCustomRole(ctx, &protos.UpdateCustomRoleRequest{RoleDefinitionId: uint64(math.MaxInt64) + 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "update name", platformError: pkg_errors.AccessManagementInvalidRoleName,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.UpdateCustomRole(ctx, &protos.UpdateCustomRoleRequest{RoleDefinitionId: 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "reduce permission", platformError: pkg_errors.AccessManagementInvalidPermissionSelection,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{RoleDefinitionId: 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "reduce identifier exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{RoleDefinitionId: uint64(math.MaxInt64) + 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "reduce permission exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidPermissionSelection,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{RoleDefinitionId: 1, PermissionDefinitionIds: []uint64{uint64(math.MaxInt64) + 1}})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "reduce reason", platformError: pkg_errors.AccessManagementInvalidRoleChangeReason,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{RoleDefinitionId: 1, PermissionDefinitionIds: []uint64{1}})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "archive reason", platformError: pkg_errors.AccessManagementInvalidRoleChangeReason,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ArchiveCustomRole(ctx, &protos.ArchiveCustomRoleRequest{RoleDefinitionId: 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
		{
			name: "archive identifier exceeds signed bigint", platformError: pkg_errors.AccessManagementInvalidRoleID,
			invoke: func(api *accessManagementGRPC) (int32, bool, *protos.Error, error) {
				response, err := api.ArchiveCustomRole(ctx, &protos.ArchiveCustomRoleRequest{RoleDefinitionId: uint64(math.MaxInt64) + 1})
				return response.GetCode(), response.GetSuccess(), response.GetError(), err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &accessManagementGRPC{service: &fakeAccessManagementService{err: errors.New("service must not be called")}}
			code, success, responseError, err := test.invoke(api)
			require.NoError(t, err)
			require.False(t, success)
			require.Equal(t, test.platformError.HTTPStatusCodeInt32(), code)
			require.EqualValues(t, test.platformError.Code, responseError.GetErrorCode())
			require.Equal(t, test.platformError.Error, responseError.GetErrorMessage())
			require.Equal(t, test.platformError.ErrorMessage, responseError.GetHumanMessage())
		})
	}
}

func TestAccessManagementReduceUsesRequestCorrelation(t *testing.T) {
	definition := &internal_entity.RoleDefinition{
		Key: "custom_reader", Name: "Custom Reader", ScopeType: accessmanagement.ScopeProject,
		SubjectType: accessmanagement.SubjectUser, DefinitionType: accessmanagement.DefinitionCustom,
	}
	definition.Id = 201
	definition.CreatedActorType = "unknown"
	service := &fakeAccessManagementService{role: definition}
	api := &accessManagementGRPC{service: service}
	response, err := api.ReduceCustomRolePermissions(accessManagementContext("request-123"), &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: 201, PermissionDefinitionIds: []uint64{34}, Reason: "Remove access", ConfirmationDigest: strings.Repeat("a", 64),
	})
	require.NoError(t, err)
	require.Equal(t, int32(200), response.GetCode())
	require.True(t, response.GetSuccess())
	require.Nil(t, response.GetError())
	require.Equal(t, "request-123", response.GetCorrelationId())
	require.Equal(t, "request-123", service.reduceCorrelationID)
	require.Nil(t, response.GetRole().GetCreatedActor())
	require.Nil(t, response.GetRole().GetUpdatedDate())
}

func TestAccessManagementReductionEnforcesTwoPhaseContract(t *testing.T) {
	definition := &internal_entity.RoleDefinition{ScopeType: accessmanagement.ScopeProject, SubjectType: accessmanagement.SubjectUser, DefinitionType: accessmanagement.DefinitionCustom}
	definition.Id = 201
	service := &fakeAccessManagementService{role: definition}
	api := &accessManagementGRPC{service: service}
	ctx := accessManagementContext("request-two-phase")

	response, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: 201, PermissionDefinitionIds: []uint64{34}, Reason: "Reduce", Preview: true, ConfirmationDigest: strings.Repeat("a", 64),
	})
	require.NoError(t, err)
	require.EqualValues(t, pkg_errors.AccessManagementInvalidConfirmationDigest.Code, response.GetError().GetErrorCode())

	response, err = api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: 201, PermissionDefinitionIds: []uint64{34}, Reason: "Reduce",
	})
	require.NoError(t, err)
	require.EqualValues(t, pkg_errors.AccessManagementInvalidConfirmationDigest.Code, response.GetError().GetErrorCode())
}

func TestAccessManagementReductionPreviewMapsImpacts(t *testing.T) {
	definition := &internal_entity.RoleDefinition{ScopeType: accessmanagement.ScopeProject, SubjectType: accessmanagement.SubjectUser, DefinitionType: accessmanagement.DefinitionCustom}
	definition.Id = 201
	definition.CreatedActorType = string(authentication.IdentityUser)
	definition.CreatedActorID = 7
	permission := &internal_entity.PermissionDefinition{Key: "project.read", Name: "Read projects", ScopeType: accessmanagement.ScopeProject}
	permission.Id = 34
	definition.RolePermissions = []*internal_entity.RolePermission{{PermissionDefinition: permission}}
	service := &fakeAccessManagementService{reductionPreview: &internal_service.RoleReductionPreview{
		ResultingRole: definition, ConfirmationDigest: strings.Repeat("b", 64),
		Impacts: []internal_service.RoleReductionImpact{{OrganizationID: 10, ProjectID: 20, SubjectType: accessmanagement.SubjectUser, SubjectID: 7, AssignmentStatus: "INVITED"}},
	}}
	api := &accessManagementGRPC{service: service}
	response, err := api.ReduceCustomRolePermissions(accessManagementContext("request-preview"), &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: 201, PermissionDefinitionIds: []uint64{34}, Reason: "Reduce", Preview: true,
	})
	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Nil(t, response.GetRole())
	require.Equal(t, uint64(201), response.GetPreview().GetResultingRole().GetId())
	require.Equal(t, protos.RoleScope_ROLE_SCOPE_PROJECT, response.GetPreview().GetResultingRole().GetScope())
	require.Equal(t, protos.RoleSubject_ROLE_SUBJECT_USER, response.GetPreview().GetResultingRole().GetSubject())
	require.Equal(t, protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM, response.GetPreview().GetResultingRole().GetDefinitionType())
	require.Equal(t, uint64(7), response.GetPreview().GetResultingRole().GetCreatedActor().GetId())
	require.Equal(t, "project.read", response.GetPreview().GetResultingRole().GetPermissions()[0].GetKey())
	require.Equal(t, protos.RoleScope_ROLE_SCOPE_PROJECT, response.GetPreview().GetResultingRole().GetPermissions()[0].GetScope())
	require.Equal(t, strings.Repeat("b", 64), response.GetPreview().GetConfirmationDigest())
	require.Equal(t, protos.RoleSubject_ROLE_SUBJECT_USER, response.GetPreview().GetImpacts()[0].GetSubject())
	require.Equal(t, "INVITED", response.GetPreview().GetImpacts()[0].GetAssignmentStatus())
}

func TestAccessManagementReturnsAuditActorIdentityWithoutDisplayName(t *testing.T) {
	definition := &internal_entity.RoleDefinition{}
	definition.CreatedActorType = string(authentication.IdentityUser)
	definition.CreatedActorID = 1

	api := &accessManagementGRPC{service: &fakeAccessManagementService{
		role: definition,
	}}
	response, err := api.GetRole(accessManagementContext("request-actor"), &protos.GetRoleRequest{RoleDefinitionId: 201})
	require.NoError(t, err)
	require.Equal(t, int32(200), response.GetCode())
	require.True(t, response.GetSuccess())
	require.Nil(t, response.GetError())
	require.Equal(t, uint64(1), response.GetRole().GetCreatedActor().GetId())
	require.Empty(t, response.GetRole().GetCreatedActor().GetDisplayName())
}

func TestAccessManagementRemainingSuccessResponsesUseEnvelope(t *testing.T) {
	definition := &internal_entity.RoleDefinition{
		Key: "custom_reader", Name: "Custom Reader", ScopeType: accessmanagement.ScopeProject,
		SubjectType: accessmanagement.SubjectUser, DefinitionType: accessmanagement.DefinitionCustom,
	}
	definition.Id = 201
	definition.CreatedActorType = "unknown"
	api := &accessManagementGRPC{service: &fakeAccessManagementService{
		role: definition,
	}}
	ctx := accessManagementContext("request-envelope")

	listed, err := api.ListRoles(ctx, &protos.ListRolesRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(200), listed.GetCode())
	require.True(t, listed.GetSuccess())
	require.Nil(t, listed.GetError())

	created, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{
		Key: "custom_reader", Name: "Custom Reader", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT,
		Subject: protos.RoleSubject_ROLE_SUBJECT_USER,
	})
	require.NoError(t, err)
	require.Equal(t, int32(200), created.GetCode())
	require.True(t, created.GetSuccess())
	require.Nil(t, created.GetError())

	updated, err := api.UpdateCustomRole(ctx, &protos.UpdateCustomRoleRequest{
		RoleDefinitionId: 201, Name: "Reader", Description: "Reads projects",
	})
	require.NoError(t, err)
	require.Equal(t, int32(200), updated.GetCode())
	require.True(t, updated.GetSuccess())
	require.Nil(t, updated.GetError())

	archived, err := api.ArchiveCustomRole(ctx, &protos.ArchiveCustomRoleRequest{
		RoleDefinitionId: 201, Reason: "No longer needed",
	})
	require.NoError(t, err)
	require.Equal(t, int32(200), archived.GetCode())
	require.True(t, archived.GetSuccess())
	require.Nil(t, archived.GetError())
}

func TestAccessManagementResponseEnvelopeFieldNumbers(t *testing.T) {
	tests := []struct {
		message   protoreflect.MessageDescriptor
		payload   protoreflect.Name
		secondary protoreflect.Name
	}{
		{message: (&protos.ListPermissionsResponse{}).ProtoReflect().Descriptor(), payload: "permissions", secondary: "paginated"},
		{message: (&protos.ListRolesResponse{}).ProtoReflect().Descriptor(), payload: "roles", secondary: "paginated"},
		{message: (&protos.GetRoleResponse{}).ProtoReflect().Descriptor(), payload: "role"},
		{message: (&protos.CreateCustomRoleResponse{}).ProtoReflect().Descriptor(), payload: "role"},
		{message: (&protos.UpdateCustomRoleResponse{}).ProtoReflect().Descriptor(), payload: "role"},
		{message: (&protos.ReduceCustomRolePermissionsResponse{}).ProtoReflect().Descriptor(), payload: "role", secondary: "correlationId"},
		{message: (&protos.ArchiveCustomRoleResponse{}).ProtoReflect().Descriptor(), payload: "roleDefinitionId", secondary: "correlationId"},
	}
	for _, test := range tests {
		require.Equal(t, protoreflect.FieldNumber(1), test.message.Fields().ByName("code").Number(), test.message.FullName())
		require.Equal(t, protoreflect.FieldNumber(2), test.message.Fields().ByName("success").Number(), test.message.FullName())
		require.Equal(t, protoreflect.FieldNumber(3), test.message.Fields().ByName(test.payload).Number(), test.message.FullName())
		require.Equal(t, protoreflect.FieldNumber(4), test.message.Fields().ByName("error").Number(), test.message.FullName())
		if test.secondary != "" {
			require.Equal(t, protoreflect.FieldNumber(5), test.message.Fields().ByName(test.secondary).Number(), test.message.FullName())
		}
	}
}

func TestAccessManagementCriteriaRequestDescriptors(t *testing.T) {
	permissions := (&protos.ListPermissionsRequest{}).ProtoReflect().Descriptor()
	permissionCriterias := permissions.Fields().ByName("criterias")
	require.NotNil(t, permissionCriterias)
	require.Equal(t, protoreflect.FieldNumber(3), permissionCriterias.Number())
	require.True(t, permissionCriterias.IsList())
	require.Nil(t, permissions.Fields().ByName("scope"))
	permissionFieldReserved := false
	for index := 0; index < permissions.ReservedRanges().Len(); index++ {
		reserved := permissions.ReservedRanges().Get(index)
		permissionFieldReserved = permissionFieldReserved || protoreflect.FieldNumber(2) >= reserved[0] && protoreflect.FieldNumber(2) < reserved[1]
	}
	require.True(t, permissionFieldReserved)
	permissionNameReserved := false
	for index := 0; index < permissions.ReservedNames().Len(); index++ {
		permissionNameReserved = permissionNameReserved || permissions.ReservedNames().Get(index) == protoreflect.Name("scope")
	}
	require.True(t, permissionNameReserved)

	roles := (&protos.ListRolesRequest{}).ProtoReflect().Descriptor()
	roleCriterias := roles.Fields().ByName("criterias")
	require.NotNil(t, roleCriterias)
	require.Equal(t, protoreflect.FieldNumber(6), roleCriterias.Number())
	require.True(t, roleCriterias.IsList())
	for _, name := range []protoreflect.Name{"query", "scope", "subject", "definitionType"} {
		require.Nil(t, roles.Fields().ByName(name))
		reserved := false
		for index := 0; index < roles.ReservedNames().Len(); index++ {
			reserved = reserved || roles.ReservedNames().Get(index) == name
		}
		require.True(t, reserved, name)
	}
	for fieldNumber := protoreflect.FieldNumber(2); fieldNumber <= 5; fieldNumber++ {
		reserved := false
		for index := 0; index < roles.ReservedRanges().Len(); index++ {
			reservedRange := roles.ReservedRanges().Get(index)
			reserved = reserved || fieldNumber >= reservedRange[0] && fieldNumber < reservedRange[1]
		}
		require.True(t, reserved, fieldNumber)
	}
}

func TestAccessManagementMapsAuthenticationAndDomainErrors(t *testing.T) {
	api := &accessManagementGRPC{service: &fakeAccessManagementService{}}
	_, err := api.ListPermissions(context.Background(), &protos.ListPermissionsRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	api = &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleChangeNotAllowedError{}}}
	response, err := api.UpdateCustomRole(accessManagementContext("request-1"), &protos.UpdateCustomRoleRequest{
		RoleDefinitionId: 201, Name: "Custom Reader",
	})
	require.NoError(t, err)
	require.EqualValues(t, pkg_errors.AccessManagementRoleChangeNotAllowed.Code, response.GetError().GetErrorCode())

	permissions, err := api.ListPermissions(accessManagementContext("request-1"), &protos.ListPermissionsRequest{
		Paginate: &protos.Paginate{Page: 0, PageSize: 25},
	})
	require.NoError(t, err)
	require.EqualValues(t, pkg_errors.AccessManagementInvalidPagination.Code, permissions.GetError().GetErrorCode())
}

func TestAccessManagementGetRoleReturnsDomainEnvelope(t *testing.T) {
	api := &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleNotFoundError{}}}
	response, err := api.GetRole(accessManagementContext("request-domain"), &protos.GetRoleRequest{RoleDefinitionId: 201})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), response.GetCode())
	require.False(t, response.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementRoleNotFound.Code, response.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.Error, response.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.ErrorMessage, response.GetError().GetHumanMessage())
}

func TestAccessManagementMutationsMapDomainEnvelopes(t *testing.T) {
	ctx := accessManagementContext("request-domain")

	api := &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleKeyAlreadyExistsError{}}}
	created, err := api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{
		Key: "custom_reader", Name: "Custom Reader", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT,
		Subject: protos.RoleSubject_ROLE_SUBJECT_USER,
	})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementRoleKeyAlreadyExists.HTTPStatusCodeInt32(), created.GetCode())
	require.False(t, created.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementRoleKeyAlreadyExists.Code, created.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementRoleKeyAlreadyExists.Error, created.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementRoleKeyAlreadyExists.ErrorMessage, created.GetError().GetHumanMessage())

	api = &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRolePermissionSelectionInvalidError{}}}
	created, err = api.CreateCustomRole(ctx, &protos.CreateCustomRoleRequest{
		Key: "custom_reader", Name: "Custom Reader", Scope: protos.RoleScope_ROLE_SCOPE_PROJECT,
		Subject: protos.RoleSubject_ROLE_SUBJECT_USER,
	})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), created.GetCode())
	require.False(t, created.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementInvalidPermissionSelection.Code, created.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementInvalidPermissionSelection.Error, created.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage, created.GetError().GetHumanMessage())

	api = &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleNotFoundError{}}}
	updated, err := api.UpdateCustomRole(ctx, &protos.UpdateCustomRoleRequest{RoleDefinitionId: 201, Name: "Custom Reader"})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), updated.GetCode())
	require.False(t, updated.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementRoleNotFound.Code, updated.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.Error, updated.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.ErrorMessage, updated.GetError().GetHumanMessage())

	api = &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleChangeNotAllowedError{}}}
	reduced, err := api.ReduceCustomRolePermissions(ctx, &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: 201, PermissionDefinitionIds: []uint64{34}, Reason: "Remove access", ConfirmationDigest: strings.Repeat("a", 64),
	})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementRoleChangeNotAllowed.HTTPStatusCodeInt32(), reduced.GetCode())
	require.False(t, reduced.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementRoleChangeNotAllowed.Code, reduced.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementRoleChangeNotAllowed.Error, reduced.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementRoleChangeNotAllowed.ErrorMessage, reduced.GetError().GetHumanMessage())

	api = &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementRoleNotFoundError{}}}
	archived, err := api.ArchiveCustomRole(ctx, &protos.ArchiveCustomRoleRequest{RoleDefinitionId: 201, Reason: "Retire role"})
	require.NoError(t, err)
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), archived.GetCode())
	require.False(t, archived.GetSuccess())
	require.EqualValues(t, pkg_errors.AccessManagementRoleNotFound.Code, archived.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.Error, archived.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AccessManagementRoleNotFound.ErrorMessage, archived.GetError().GetHumanMessage())
}

func TestAccessManagementGetRoleKeepsPermissionDeniedStatus(t *testing.T) {
	api := &accessManagementGRPC{service: &fakeAccessManagementService{err: pkg_errors.AccessManagementPermissionDeniedError{}}}
	_, err := api.GetRole(accessManagementContext("request-denied"), &protos.GetRoleRequest{RoleDefinitionId: 201})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestAccessManagementMutationRequiresCorrelationContext(t *testing.T) {
	api := &accessManagementGRPC{service: &fakeAccessManagementService{}}
	_, err := api.ArchiveCustomRole(accessManagementContext(""), &protos.ArchiveCustomRoleRequest{
		RoleDefinitionId: 201, Reason: "Archive role",
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestAccessManagementInternalLogIncludesCorrelation(t *testing.T) {
	logger := &recordingAccessManagementLogger{}
	api := &accessManagementGRPC{logger: logger, service: &fakeAccessManagementService{err: errors.New("database unavailable")}}
	_, err := api.GetRole(accessManagementContext("request-log-123"), &protos.GetRoleRequest{RoleDefinitionId: 201})
	require.Equal(t, codes.Internal, status.Code(err))
	require.True(t, strings.Contains(logger.message, `correlation="request-log-123"`), "log = %q", logger.message)
}
