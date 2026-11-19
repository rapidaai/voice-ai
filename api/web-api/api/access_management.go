// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	internal_services "github.com/rapidaai/api/web-api/internal/service"
	accessmanagement "github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var customRoleKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
var roleReductionDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type accessManagementGRPC struct {
	logger  commons.Logger
	service internal_services.AccessManagementService
}

func NewAccessManagementGRPC(logger commons.Logger, postgres connectors.PostgresConnector) protos.AccessManagementServiceServer {
	return &accessManagementGRPC{logger: logger, service: accessmanagement.NewAccessManagementService(postgres)}
}

func (api *accessManagementGRPC) ListPermissions(ctx context.Context, request *protos.ListPermissionsRequest) (*protos.ListPermissionsResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	if len(request.GetCriterias()) > 1 {
		return &protos.ListPermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.ErrorMessage}}, nil
	}
	criterias := make([]*protos.Criteria, 0, len(request.GetCriterias()))
	seen := make(map[string]struct{}, len(request.GetCriterias()))
	for _, criteria := range request.GetCriterias() {
		if criteria == nil || criteria.GetKey() != "scope" || criteria.GetLogic() != "=" {
			return &protos.ListPermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.ErrorMessage}}, nil
		}
		if _, exists := seen[criteria.GetKey()]; exists {
			return &protos.ListPermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.ErrorMessage}}, nil
		}
		value := strings.TrimSpace(criteria.GetValue())
		if !validator.OneOf(value, accessmanagement.ScopeOrganization, accessmanagement.ScopeProject) {
			return &protos.ListPermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionCriteria.ErrorMessage}}, nil
		}
		seen[criteria.GetKey()] = struct{}{}
		criterias = append(criterias, &protos.Criteria{Key: criteria.GetKey(), Logic: criteria.GetLogic(), Value: value})
	}
	paginate := &protos.Paginate{Page: 1, PageSize: 25}
	if request.GetPaginate() != nil {
		paginate.Page = request.GetPaginate().GetPage()
		paginate.PageSize = request.GetPaginate().GetPageSize()
		if paginate.Page == 0 || paginate.PageSize == 0 || paginate.PageSize > 100 {
			return &protos.ListPermissionsResponse{Code: pkg_errors.AccessManagementInvalidPagination.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPagination.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPagination.Error, HumanMessage: pkg_errors.AccessManagementInvalidPagination.ErrorMessage}}, nil
		}
	}
	count, permissions, err := api.service.ListPermissions(ctx, principal, criterias, paginate)
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		default:
			correlationID, _ := ctx.Value(types.REQUEST_ID_KEY).(string)
			if api.logger != nil {
				api.logger.Errorf("access management list permissions failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, 0, correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	if count < 0 || count > math.MaxUint32 {
		return nil, status.Error(codes.Internal, "permission count exceeds response contract")
	}
	response := &protos.ListPermissionsResponse{
		Code: 200, Success: true,
		Permissions: make([]*protos.PermissionDefinition, 0, len(permissions)),
		Paginated: &protos.Paginated{
			CurrentPage: paginate.Page,
			TotalItem:   uint32(count),
		},
	}
	for _, permission := range permissions {
		if permission == nil {
			continue
		}
		permissionResponse := &protos.PermissionDefinition{
			Id: permission.Id, Key: permission.Key, Name: permission.Name,
			Description: permission.Description,
		}
		switch permission.ScopeType {
		case accessmanagement.ScopeOrganization:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		response.Permissions = append(response.Permissions, permissionResponse)
	}
	return response, nil
}

func (api *accessManagementGRPC) ListRoles(ctx context.Context, request *protos.ListRolesRequest) (*protos.ListRolesResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	if len(request.GetCriterias()) > 5 {
		return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
	}
	criterias := make([]*protos.Criteria, 0, len(request.GetCriterias()))
	seen := make(map[string]struct{}, len(request.GetCriterias()))
	for _, criteria := range request.GetCriterias() {
		if criteria == nil {
			return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
		}
		if _, exists := seen[criteria.GetKey()]; exists {
			return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
		}
		value := strings.TrimSpace(criteria.GetValue())
		if strings.ContainsRune(value, '\x00') || !validator.Between(len([]rune(value)), 1, 200) {
			return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
		}
		switch criteria.GetKey() {
		case "name", "key":
			if criteria.GetLogic() != "contains" {
				return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
			}
		case "scope":
			if criteria.GetLogic() != "=" || !validator.OneOf(value, accessmanagement.ScopeOrganization, accessmanagement.ScopeProject) {
				return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
			}
		case "subject":
			if criteria.GetLogic() != "=" || !validator.OneOf(value, accessmanagement.SubjectUser, accessmanagement.SubjectProjectToken) {
				return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
			}
		case "type":
			if criteria.GetLogic() != "=" || !validator.OneOf(value, accessmanagement.DefinitionBuiltIn, accessmanagement.DefinitionCustom) {
				return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
			}
		default:
			return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidRoleCriteria.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleCriteria.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleCriteria.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleCriteria.ErrorMessage}}, nil
		}
		seen[criteria.GetKey()] = struct{}{}
		criterias = append(criterias, &protos.Criteria{Key: criteria.GetKey(), Logic: criteria.GetLogic(), Value: value})
	}
	paginate := &protos.Paginate{Page: 1, PageSize: 25}
	if request.GetPaginate() != nil {
		paginate.Page = request.GetPaginate().GetPage()
		paginate.PageSize = request.GetPaginate().GetPageSize()
		if paginate.Page == 0 || paginate.PageSize == 0 || paginate.PageSize > 100 {
			return &protos.ListRolesResponse{Code: pkg_errors.AccessManagementInvalidPagination.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPagination.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPagination.Error, HumanMessage: pkg_errors.AccessManagementInvalidPagination.ErrorMessage}}, nil
		}
	}
	count, roles, err := api.service.ListRoles(ctx, principal, criterias, paginate)
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		default:
			correlationID, _ := ctx.Value(types.REQUEST_ID_KEY).(string)
			if api.logger != nil {
				api.logger.Errorf("access management list roles failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, 0, correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	if count < 0 || count > math.MaxUint32 {
		return nil, status.Error(codes.Internal, "role count exceeds response contract")
	}
	response := &protos.ListRolesResponse{
		Code: 200, Success: true,
		Roles: make([]*protos.RoleDefinition, 0, len(roles)),
		Paginated: &protos.Paginated{
			CurrentPage: paginate.Page,
			TotalItem:   uint32(count),
		},
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		roleResponse := &protos.RoleDefinition{
			Id: role.Id, Key: role.Key, Name: role.Name, Description: role.Description,
			CreatedDate: timestamppb.New(time.Time(role.CreatedDate)),
			Permissions: make([]*protos.PermissionDefinition, 0),
		}
		if updatedDate := time.Time(role.UpdatedDate); !updatedDate.IsZero() {
			roleResponse.UpdatedDate = timestamppb.New(updatedDate)
		}
		if role.CreatedActorID != 0 && role.CreatedActorType != "" && role.CreatedActorType != "unknown" {
			roleResponse.CreatedActor = &protos.AuditActor{Type: role.CreatedActorType, Id: role.CreatedActorID}
		}
		if role.UpdatedActorID != 0 && role.UpdatedActorType != "" && role.UpdatedActorType != "unknown" {
			roleResponse.UpdatedActor = &protos.AuditActor{Type: role.UpdatedActorType, Id: role.UpdatedActorID}
		}
		switch role.ScopeType {
		case accessmanagement.ScopeOrganization:
			roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		switch role.SubjectType {
		case accessmanagement.SubjectUser:
			roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
		case accessmanagement.SubjectProjectToken:
			roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
		}
		switch role.DefinitionType {
		case accessmanagement.DefinitionBuiltIn:
			roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
		case accessmanagement.DefinitionCustom:
			roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
		}
		response.Roles = append(response.Roles, roleResponse)
	}
	return response, nil
}

func (api *accessManagementGRPC) GetRole(ctx context.Context, request *protos.GetRoleRequest) (*protos.GetRoleResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	if !validator.Between(request.GetRoleDefinitionId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.GetRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleID.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleID.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleID.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleID.ErrorMessage}}, nil
	}
	role, err := api.service.GetRole(ctx, principal, request.GetRoleDefinitionId())
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		case errors.As(err, &roleNotFound):
			return &protos.GetRoleResponse{
				Code: pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(),
				Error: &protos.Error{
					ErrorCode:    uint64(pkg_errors.AccessManagementRoleNotFound.Code),
					ErrorMessage: pkg_errors.AccessManagementRoleNotFound.Error,
					HumanMessage: pkg_errors.AccessManagementRoleNotFound.ErrorMessage,
				},
			}, nil
		default:
			correlationID, _ := ctx.Value(types.REQUEST_ID_KEY).(string)
			if api.logger != nil {
				api.logger.Errorf("access management get role failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, request.GetRoleDefinitionId(), correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	roleResponse := &protos.RoleDefinition{
		Id: role.Id, Key: role.Key, Name: role.Name, Description: role.Description,
		CreatedDate: timestamppb.New(time.Time(role.CreatedDate)),
		Permissions: make([]*protos.PermissionDefinition, 0, len(role.RolePermissions)),
	}
	if updatedDate := time.Time(role.UpdatedDate); !updatedDate.IsZero() {
		roleResponse.UpdatedDate = timestamppb.New(updatedDate)
	}
	if role.CreatedActorID != 0 && role.CreatedActorType != "" && role.CreatedActorType != "unknown" {
		roleResponse.CreatedActor = &protos.AuditActor{Type: role.CreatedActorType, Id: role.CreatedActorID}
	}
	if role.UpdatedActorID != 0 && role.UpdatedActorType != "" && role.UpdatedActorType != "unknown" {
		roleResponse.UpdatedActor = &protos.AuditActor{Type: role.UpdatedActorType, Id: role.UpdatedActorID}
	}
	switch role.ScopeType {
	case accessmanagement.ScopeOrganization:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
	case accessmanagement.ScopeProject:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
	}
	switch role.SubjectType {
	case accessmanagement.SubjectUser:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
	case accessmanagement.SubjectProjectToken:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
	}
	switch role.DefinitionType {
	case accessmanagement.DefinitionBuiltIn:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
	case accessmanagement.DefinitionCustom:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
	}
	for _, rolePermission := range role.RolePermissions {
		if rolePermission == nil || rolePermission.PermissionDefinition == nil {
			continue
		}
		permissionResponse := &protos.PermissionDefinition{
			Id: rolePermission.PermissionDefinition.Id, Key: rolePermission.PermissionDefinition.Key, Name: rolePermission.PermissionDefinition.Name,
			Description: rolePermission.PermissionDefinition.Description,
		}
		switch rolePermission.PermissionDefinition.ScopeType {
		case accessmanagement.ScopeOrganization:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		roleResponse.Permissions = append(roleResponse.Permissions, permissionResponse)
	}
	return &protos.GetRoleResponse{Code: 200, Success: true, Role: roleResponse}, nil
}

func (api *accessManagementGRPC) CreateCustomRole(ctx context.Context, request *protos.CreateCustomRoleRequest) (*protos.CreateCustomRoleResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	scope := ""
	switch request.GetScope() {
	case protos.RoleScope_ROLE_SCOPE_ORGANIZATION:
		scope = accessmanagement.ScopeOrganization
	case protos.RoleScope_ROLE_SCOPE_PROJECT:
		scope = accessmanagement.ScopeProject
	case protos.RoleScope_ROLE_SCOPE_UNSPECIFIED:
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleScope.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleScope.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleScope.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleScope.ErrorMessage}}, nil
	default:
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleScope.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleScope.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleScope.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleScope.ErrorMessage}}, nil
	}
	subject := ""
	switch request.GetSubject() {
	case protos.RoleSubject_ROLE_SUBJECT_USER:
		subject = accessmanagement.SubjectUser
	case protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN:
		subject = accessmanagement.SubjectProjectToken
	case protos.RoleSubject_ROLE_SUBJECT_UNSPECIFIED:
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleSubject.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleSubject.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleSubject.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleSubject.ErrorMessage}}, nil
	default:
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleSubject.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleSubject.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleSubject.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleSubject.ErrorMessage}}, nil
	}
	if subject == accessmanagement.SubjectProjectToken && scope != accessmanagement.ScopeProject {
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidAPIRoleScope.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidAPIRoleScope.Code), ErrorMessage: pkg_errors.AccessManagementInvalidAPIRoleScope.Error, HumanMessage: pkg_errors.AccessManagementInvalidAPIRoleScope.ErrorMessage}}, nil
	}
	key := strings.TrimSpace(request.GetKey())
	if !customRoleKeyPattern.MatchString(key) {
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleKey.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleKey.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleKey.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleKey.ErrorMessage}}, nil
	}
	name := strings.TrimSpace(request.GetName())
	description := strings.TrimSpace(request.GetDescription())
	if strings.ContainsRune(name, '\x00') || !validator.Between(utf8.RuneCountInString(name), 1, 200) {
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleName.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleName.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleName.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleName.ErrorMessage}}, nil
	}
	if strings.ContainsRune(description, '\x00') || !validator.Between(utf8.RuneCountInString(description), 0, 400) {
		return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleDescription.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleDescription.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleDescription.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleDescription.ErrorMessage}}, nil
	}
	seenPermissionDefinitionIDs := make(map[uint64]struct{}, len(request.GetPermissionDefinitionIds()))
	for _, permissionDefinitionID := range request.GetPermissionDefinitionIds() {
		if !validator.Between(permissionDefinitionID, uint64(1), uint64(math.MaxInt64)) {
			return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
		}
		if _, exists := seenPermissionDefinitionIDs[permissionDefinitionID]; exists {
			return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
		}
		seenPermissionDefinitionIDs[permissionDefinitionID] = struct{}{}
	}
	role, err := api.service.CreateCustomRole(ctx, principal, key, name, description, scope, subject, request.GetPermissionDefinitionIds())
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		var roleKeyAlreadyExists pkg_errors.AccessManagementRoleKeyAlreadyExistsError
		var permissionSelectionInvalid pkg_errors.AccessManagementRolePermissionSelectionInvalidError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		case errors.As(err, &roleKeyAlreadyExists):
			return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementRoleKeyAlreadyExists.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleKeyAlreadyExists.Code), ErrorMessage: pkg_errors.AccessManagementRoleKeyAlreadyExists.Error, HumanMessage: pkg_errors.AccessManagementRoleKeyAlreadyExists.ErrorMessage}}, nil
		case errors.As(err, &permissionSelectionInvalid):
			return &protos.CreateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
		default:
			correlationID, _ := ctx.Value(types.REQUEST_ID_KEY).(string)
			if api.logger != nil {
				api.logger.Errorf("access management create role failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, 0, correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	if role == nil {
		return nil, status.Error(codes.Internal, "access management returned an empty role")
	}
	roleResponse := &protos.RoleDefinition{
		Id: role.Id, Key: role.Key, Name: role.Name, Description: role.Description,
		CreatedDate: timestamppb.New(time.Time(role.CreatedDate)),
		Permissions: make([]*protos.PermissionDefinition, 0, len(role.RolePermissions)),
	}
	if updatedDate := time.Time(role.UpdatedDate); !updatedDate.IsZero() {
		roleResponse.UpdatedDate = timestamppb.New(updatedDate)
	}
	if role.CreatedActorID != 0 && role.CreatedActorType != "" && role.CreatedActorType != "unknown" {
		roleResponse.CreatedActor = &protos.AuditActor{Type: role.CreatedActorType, Id: role.CreatedActorID}
	}
	if role.UpdatedActorID != 0 && role.UpdatedActorType != "" && role.UpdatedActorType != "unknown" {
		roleResponse.UpdatedActor = &protos.AuditActor{Type: role.UpdatedActorType, Id: role.UpdatedActorID}
	}
	switch role.ScopeType {
	case accessmanagement.ScopeOrganization:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
	case accessmanagement.ScopeProject:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
	}
	switch role.SubjectType {
	case accessmanagement.SubjectUser:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
	case accessmanagement.SubjectProjectToken:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
	}
	switch role.DefinitionType {
	case accessmanagement.DefinitionBuiltIn:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
	case accessmanagement.DefinitionCustom:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
	}
	for _, rolePermission := range role.RolePermissions {
		if rolePermission == nil || rolePermission.PermissionDefinition == nil {
			continue
		}
		permissionResponse := &protos.PermissionDefinition{
			Id: rolePermission.PermissionDefinition.Id, Key: rolePermission.PermissionDefinition.Key, Name: rolePermission.PermissionDefinition.Name,
			Description: rolePermission.PermissionDefinition.Description,
		}
		switch rolePermission.PermissionDefinition.ScopeType {
		case accessmanagement.ScopeOrganization:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		roleResponse.Permissions = append(roleResponse.Permissions, permissionResponse)
	}
	return &protos.CreateCustomRoleResponse{Code: 200, Success: true, Role: roleResponse}, nil
}

func (api *accessManagementGRPC) UpdateCustomRole(ctx context.Context, request *protos.UpdateCustomRoleRequest) (*protos.UpdateCustomRoleResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	if !validator.Between(request.GetRoleDefinitionId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.UpdateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleID.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleID.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleID.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleID.ErrorMessage}}, nil
	}
	name := strings.TrimSpace(request.GetName())
	description := strings.TrimSpace(request.GetDescription())
	if strings.ContainsRune(name, '\x00') || !validator.Between(utf8.RuneCountInString(name), 1, 200) {
		return &protos.UpdateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleName.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleName.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleName.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleName.ErrorMessage}}, nil
	}
	if strings.ContainsRune(description, '\x00') || !validator.Between(utf8.RuneCountInString(description), 0, 400) {
		return &protos.UpdateCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleDescription.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleDescription.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleDescription.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleDescription.ErrorMessage}}, nil
	}
	role, err := api.service.UpdateCustomRole(ctx, principal, request.GetRoleDefinitionId(), name, description)
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
		var roleChangeNotAllowed pkg_errors.AccessManagementRoleChangeNotAllowedError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		case errors.As(err, &roleNotFound):
			return &protos.UpdateCustomRoleResponse{Code: pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleNotFound.Code), ErrorMessage: pkg_errors.AccessManagementRoleNotFound.Error, HumanMessage: pkg_errors.AccessManagementRoleNotFound.ErrorMessage}}, nil
		case errors.As(err, &roleChangeNotAllowed):
			return &protos.UpdateCustomRoleResponse{Code: pkg_errors.AccessManagementRoleChangeNotAllowed.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleChangeNotAllowed.Code), ErrorMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.Error, HumanMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.ErrorMessage}}, nil
		default:
			correlationID, _ := ctx.Value(types.REQUEST_ID_KEY).(string)
			if api.logger != nil {
				api.logger.Errorf("access management update role failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, request.GetRoleDefinitionId(), correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	if role == nil {
		return nil, status.Error(codes.Internal, "access management returned an empty role")
	}
	roleResponse := &protos.RoleDefinition{
		Id: role.Id, Key: role.Key, Name: role.Name, Description: role.Description,
		CreatedDate: timestamppb.New(time.Time(role.CreatedDate)),
		Permissions: make([]*protos.PermissionDefinition, 0, len(role.RolePermissions)),
	}
	if updatedDate := time.Time(role.UpdatedDate); !updatedDate.IsZero() {
		roleResponse.UpdatedDate = timestamppb.New(updatedDate)
	}
	if role.CreatedActorID != 0 && role.CreatedActorType != "" && role.CreatedActorType != "unknown" {
		roleResponse.CreatedActor = &protos.AuditActor{Type: role.CreatedActorType, Id: role.CreatedActorID}
	}
	if role.UpdatedActorID != 0 && role.UpdatedActorType != "" && role.UpdatedActorType != "unknown" {
		roleResponse.UpdatedActor = &protos.AuditActor{Type: role.UpdatedActorType, Id: role.UpdatedActorID}
	}
	switch role.ScopeType {
	case accessmanagement.ScopeOrganization:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
	case accessmanagement.ScopeProject:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
	}
	switch role.SubjectType {
	case accessmanagement.SubjectUser:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
	case accessmanagement.SubjectProjectToken:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
	}
	switch role.DefinitionType {
	case accessmanagement.DefinitionBuiltIn:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
	case accessmanagement.DefinitionCustom:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
	}
	for _, rolePermission := range role.RolePermissions {
		if rolePermission == nil || rolePermission.PermissionDefinition == nil {
			continue
		}
		permissionResponse := &protos.PermissionDefinition{
			Id: rolePermission.PermissionDefinition.Id, Key: rolePermission.PermissionDefinition.Key, Name: rolePermission.PermissionDefinition.Name,
			Description: rolePermission.PermissionDefinition.Description,
		}
		switch rolePermission.PermissionDefinition.ScopeType {
		case accessmanagement.ScopeOrganization:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		roleResponse.Permissions = append(roleResponse.Permissions, permissionResponse)
	}
	return &protos.UpdateCustomRoleResponse{Code: 200, Success: true, Role: roleResponse}, nil
}

func (api *accessManagementGRPC) ReduceCustomRolePermissions(ctx context.Context, request *protos.ReduceCustomRolePermissionsRequest) (*protos.ReduceCustomRolePermissionsResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	correlationID, ok := ctx.Value(types.REQUEST_ID_KEY).(string)
	if !ok || strings.TrimSpace(correlationID) == "" {
		return nil, status.Error(codes.Internal, "request correlation is unavailable")
	}
	correlationID = strings.TrimSpace(correlationID)
	if strings.ContainsRune(correlationID, '\x00') || !validator.Between(utf8.RuneCountInString(correlationID), 1, 128) {
		return nil, status.Error(codes.Internal, "request correlation is unavailable")
	}
	if !validator.Between(request.GetRoleDefinitionId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidRoleID.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleID.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleID.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleID.ErrorMessage}}, nil
	}
	if !validator.NotEmpty(request.GetPermissionDefinitionIds()) {
		return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
	}
	seenPermissionDefinitionIDs := make(map[uint64]struct{}, len(request.GetPermissionDefinitionIds()))
	for _, permissionDefinitionID := range request.GetPermissionDefinitionIds() {
		if !validator.Between(permissionDefinitionID, uint64(1), uint64(math.MaxInt64)) {
			return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
		}
		if _, exists := seenPermissionDefinitionIDs[permissionDefinitionID]; exists {
			return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidPermissionSelection.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidPermissionSelection.Code), ErrorMessage: pkg_errors.AccessManagementInvalidPermissionSelection.Error, HumanMessage: pkg_errors.AccessManagementInvalidPermissionSelection.ErrorMessage}}, nil
		}
		seenPermissionDefinitionIDs[permissionDefinitionID] = struct{}{}
	}
	reason := strings.TrimSpace(request.GetReason())
	if strings.ContainsRune(reason, '\x00') || !validator.Between(utf8.RuneCountInString(reason), 1, 400) {
		return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidRoleChangeReason.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleChangeReason.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleChangeReason.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleChangeReason.ErrorMessage}}, nil
	}
	confirmationDigest := request.GetConfirmationDigest()
	if (request.GetPreview() && confirmationDigest != "") || (!request.GetPreview() && !roleReductionDigestPattern.MatchString(confirmationDigest)) {
		return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementInvalidConfirmationDigest.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidConfirmationDigest.Code), ErrorMessage: pkg_errors.AccessManagementInvalidConfirmationDigest.Error, HumanMessage: pkg_errors.AccessManagementInvalidConfirmationDigest.ErrorMessage}}, nil
	}
	role, reductionPreview, err := api.service.ReduceCustomRolePermissions(ctx, principal, request.GetRoleDefinitionId(), request.GetPermissionDefinitionIds(), reason, correlationID, confirmationDigest, request.GetPreview())
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
		var roleChangeNotAllowed pkg_errors.AccessManagementRoleChangeNotAllowedError
		var reductionStale pkg_errors.AccessManagementReductionStaleError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		case errors.As(err, &roleNotFound):
			return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleNotFound.Code), ErrorMessage: pkg_errors.AccessManagementRoleNotFound.Error, HumanMessage: pkg_errors.AccessManagementRoleNotFound.ErrorMessage}}, nil
		case errors.As(err, &roleChangeNotAllowed):
			return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementRoleChangeNotAllowed.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleChangeNotAllowed.Code), ErrorMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.Error, HumanMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.ErrorMessage}}, nil
		case errors.As(err, &reductionStale):
			return &protos.ReduceCustomRolePermissionsResponse{Code: pkg_errors.AccessManagementReductionStale.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementReductionStale.Code), ErrorMessage: pkg_errors.AccessManagementReductionStale.Error, HumanMessage: pkg_errors.AccessManagementReductionStale.ErrorMessage}}, nil
		default:
			if api.logger != nil {
				api.logger.Errorf("access management reduce role permissions failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, request.GetRoleDefinitionId(), correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	if request.GetPreview() {
		if reductionPreview == nil || reductionPreview.ResultingRole == nil {
			return nil, status.Error(codes.Internal, "access management returned an empty reduction preview")
		}
		resultingRole := reductionPreview.ResultingRole
		resultingRoleResponse := &protos.RoleDefinition{
			Id: resultingRole.Id, Key: resultingRole.Key, Name: resultingRole.Name, Description: resultingRole.Description,
			CreatedDate: timestamppb.New(time.Time(resultingRole.CreatedDate)),
			Permissions: make([]*protos.PermissionDefinition, 0, len(resultingRole.RolePermissions)),
		}
		if updatedDate := time.Time(resultingRole.UpdatedDate); !updatedDate.IsZero() {
			resultingRoleResponse.UpdatedDate = timestamppb.New(updatedDate)
		}
		if resultingRole.CreatedActorID != 0 && resultingRole.CreatedActorType != "" && resultingRole.CreatedActorType != "unknown" {
			resultingRoleResponse.CreatedActor = &protos.AuditActor{Type: resultingRole.CreatedActorType, Id: resultingRole.CreatedActorID}
		}
		if resultingRole.UpdatedActorID != 0 && resultingRole.UpdatedActorType != "" && resultingRole.UpdatedActorType != "unknown" {
			resultingRoleResponse.UpdatedActor = &protos.AuditActor{Type: resultingRole.UpdatedActorType, Id: resultingRole.UpdatedActorID}
		}
		switch resultingRole.ScopeType {
		case accessmanagement.ScopeOrganization:
			resultingRoleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			resultingRoleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		switch resultingRole.SubjectType {
		case accessmanagement.SubjectUser:
			resultingRoleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
		case accessmanagement.SubjectProjectToken:
			resultingRoleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
		}
		switch resultingRole.DefinitionType {
		case accessmanagement.DefinitionBuiltIn:
			resultingRoleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
		case accessmanagement.DefinitionCustom:
			resultingRoleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
		}
		for _, rolePermission := range resultingRole.RolePermissions {
			if rolePermission == nil || rolePermission.PermissionDefinition == nil {
				continue
			}
			permission := rolePermission.PermissionDefinition
			permissionResponse := &protos.PermissionDefinition{
				Id: permission.Id, Key: permission.Key, Name: permission.Name, Description: permission.Description,
			}
			switch permission.ScopeType {
			case accessmanagement.ScopeOrganization:
				permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
			case accessmanagement.ScopeProject:
				permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
			}
			resultingRoleResponse.Permissions = append(resultingRoleResponse.Permissions, permissionResponse)
		}
		impacts := make([]*protos.RoleReductionImpact, 0, len(reductionPreview.Impacts))
		for _, impact := range reductionPreview.Impacts {
			item := &protos.RoleReductionImpact{
				OrganizationId: impact.OrganizationID, ProjectId: impact.ProjectID,
				SubjectId: impact.SubjectID, AssignmentStatus: impact.AssignmentStatus,
			}
			switch impact.SubjectType {
			case accessmanagement.SubjectUser:
				item.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
			case accessmanagement.SubjectProjectToken:
				item.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
			}
			impacts = append(impacts, item)
		}
		return &protos.ReduceCustomRolePermissionsResponse{
			Code: 200, Success: true, CorrelationId: correlationID,
			Preview: &protos.RoleReductionPreview{
				ResultingRole: resultingRoleResponse,
				Impacts:       impacts, ConfirmationDigest: reductionPreview.ConfirmationDigest,
			},
		}, nil
	}
	if role == nil {
		return nil, status.Error(codes.Internal, "access management returned an empty role")
	}
	roleResponse := &protos.RoleDefinition{
		Id: role.Id, Key: role.Key, Name: role.Name, Description: role.Description,
		CreatedDate: timestamppb.New(time.Time(role.CreatedDate)),
		Permissions: make([]*protos.PermissionDefinition, 0, len(role.RolePermissions)),
	}
	if updatedDate := time.Time(role.UpdatedDate); !updatedDate.IsZero() {
		roleResponse.UpdatedDate = timestamppb.New(updatedDate)
	}
	if role.CreatedActorID != 0 && role.CreatedActorType != "" && role.CreatedActorType != "unknown" {
		roleResponse.CreatedActor = &protos.AuditActor{Type: role.CreatedActorType, Id: role.CreatedActorID}
	}
	if role.UpdatedActorID != 0 && role.UpdatedActorType != "" && role.UpdatedActorType != "unknown" {
		roleResponse.UpdatedActor = &protos.AuditActor{Type: role.UpdatedActorType, Id: role.UpdatedActorID}
	}
	switch role.ScopeType {
	case accessmanagement.ScopeOrganization:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
	case accessmanagement.ScopeProject:
		roleResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
	}
	switch role.SubjectType {
	case accessmanagement.SubjectUser:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_USER
	case accessmanagement.SubjectProjectToken:
		roleResponse.Subject = protos.RoleSubject_ROLE_SUBJECT_PROJECT_TOKEN
	}
	switch role.DefinitionType {
	case accessmanagement.DefinitionBuiltIn:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_BUILT_IN
	case accessmanagement.DefinitionCustom:
		roleResponse.DefinitionType = protos.RoleDefinitionType_ROLE_DEFINITION_TYPE_CUSTOM
	}
	for _, rolePermission := range role.RolePermissions {
		if rolePermission == nil || rolePermission.PermissionDefinition == nil {
			continue
		}
		permissionResponse := &protos.PermissionDefinition{
			Id: rolePermission.PermissionDefinition.Id, Key: rolePermission.PermissionDefinition.Key, Name: rolePermission.PermissionDefinition.Name,
			Description: rolePermission.PermissionDefinition.Description,
		}
		switch rolePermission.PermissionDefinition.ScopeType {
		case accessmanagement.ScopeOrganization:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_ORGANIZATION
		case accessmanagement.ScopeProject:
			permissionResponse.Scope = protos.RoleScope_ROLE_SCOPE_PROJECT
		}
		roleResponse.Permissions = append(roleResponse.Permissions, permissionResponse)
	}
	return &protos.ReduceCustomRolePermissionsResponse{
		Code: 200, Success: true, Role: roleResponse, CorrelationId: correlationID,
	}, nil
}

func (api *accessManagementGRPC) ArchiveCustomRole(ctx context.Context, request *protos.ArchiveCustomRoleRequest) (*protos.ArchiveCustomRoleResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	tenant := principal.Tenant()
	if tenant.OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}
	correlationID, ok := ctx.Value(types.REQUEST_ID_KEY).(string)
	if !ok || strings.TrimSpace(correlationID) == "" {
		return nil, status.Error(codes.Internal, "request correlation is unavailable")
	}
	correlationID = strings.TrimSpace(correlationID)
	if strings.ContainsRune(correlationID, '\x00') || !validator.Between(utf8.RuneCountInString(correlationID), 1, 128) {
		return nil, status.Error(codes.Internal, "request correlation is unavailable")
	}
	if !validator.Between(request.GetRoleDefinitionId(), uint64(1), uint64(math.MaxInt64)) {
		return &protos.ArchiveCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleID.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleID.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleID.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleID.ErrorMessage}}, nil
	}
	reason := strings.TrimSpace(request.GetReason())
	if strings.ContainsRune(reason, '\x00') || !validator.Between(utf8.RuneCountInString(reason), 1, 400) {
		return &protos.ArchiveCustomRoleResponse{Code: pkg_errors.AccessManagementInvalidRoleChangeReason.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementInvalidRoleChangeReason.Code), ErrorMessage: pkg_errors.AccessManagementInvalidRoleChangeReason.Error, HumanMessage: pkg_errors.AccessManagementInvalidRoleChangeReason.ErrorMessage}}, nil
	}
	err := api.service.ArchiveCustomRole(ctx, principal, request.GetRoleDefinitionId(), reason, correlationID)
	if err != nil {
		var permissionDenied pkg_errors.AccessManagementPermissionDeniedError
		var roleNotFound pkg_errors.AccessManagementRoleNotFoundError
		var roleChangeNotAllowed pkg_errors.AccessManagementRoleChangeNotAllowedError
		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "request deadline exceeded")
		case errors.As(err, &permissionDenied):
			return nil, status.Error(codes.PermissionDenied, "access management permission denied")
		case errors.As(err, &roleNotFound):
			return &protos.ArchiveCustomRoleResponse{Code: pkg_errors.AccessManagementRoleNotFound.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleNotFound.Code), ErrorMessage: pkg_errors.AccessManagementRoleNotFound.Error, HumanMessage: pkg_errors.AccessManagementRoleNotFound.ErrorMessage}}, nil
		case errors.As(err, &roleChangeNotAllowed):
			return &protos.ArchiveCustomRoleResponse{Code: pkg_errors.AccessManagementRoleChangeNotAllowed.HTTPStatusCodeInt32(), Error: &protos.Error{ErrorCode: uint64(pkg_errors.AccessManagementRoleChangeNotAllowed.Code), ErrorMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.Error, HumanMessage: pkg_errors.AccessManagementRoleChangeNotAllowed.ErrorMessage}}, nil
		default:
			if api.logger != nil {
				api.logger.Errorf("access management archive role failed organization=%d role=%d correlation=%q: %v", tenant.OrganizationID, request.GetRoleDefinitionId(), correlationID, err)
			}
			return nil, status.Error(codes.Internal, "access management operation failed")
		}
	}
	return &protos.ArchiveCustomRoleResponse{
		Code: 200, Success: true,
		RoleDefinitionId: request.GetRoleDefinitionId(), CorrelationId: correlationID,
	}, nil
}
