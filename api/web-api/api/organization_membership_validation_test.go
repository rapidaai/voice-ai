// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_api

import (
	"context"
	"testing"

	internal_service "github.com/rapidaai/api/web-api/internal/service"
	pkg_errors "github.com/rapidaai/pkg/errors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/protos"
)

func TestOrganizationMembershipControllersValidateRoleSelectionsInline(t *testing.T) {
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), onboardingAuthentication(t, 1, 10))
	if err != nil {
		t.Fatal(err)
	}

	inviteTests := []struct {
		name string
		req  *protos.InviteUserToOrganizationRequest
		want pkg_errors.PlatformError
	}{
		{name: "empty organization roles", req: &protos.InviteUserToOrganizationRequest{Email: "new@example.com"}, want: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole},
		{name: "duplicate organization role", req: &protos.InviteUserToOrganizationRequest{Email: "new@example.com", OrganizationRoleDefinitionIds: []uint64{108, 108}}, want: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole},
		{name: "nil project assignment", req: &protos.InviteUserToOrganizationRequest{Email: "new@example.com", OrganizationRoleDefinitionIds: []uint64{108}, ProjectRoles: []*protos.ProjectRoleAssignment{nil}}, want: pkg_errors.InviteUserToOrganizationInvalidProjects},
		{name: "empty project roles", req: &protos.InviteUserToOrganizationRequest{Email: "new@example.com", OrganizationRoleDefinitionIds: []uint64{108}, ProjectRoles: []*protos.ProjectRoleAssignment{{ProjectId: 100}}}, want: pkg_errors.InviteUserToOrganizationInvalidProjectRole},
		{name: "duplicate project role", req: &protos.InviteUserToOrganizationRequest{Email: "new@example.com", OrganizationRoleDefinitionIds: []uint64{108}, ProjectRoles: []*protos.ProjectRoleAssignment{{ProjectId: 100, RoleDefinitionIds: []uint64{112, 112}}}}, want: pkg_errors.InviteUserToOrganizationInvalidProjectRole},
	}
	for _, test := range inviteTests {
		t.Run("invite "+test.name, func(t *testing.T) {
			response, err := (&webOrganizationGRPCApi{}).InviteUserToOrganization(ctx, test.req)
			if err != nil {
				t.Fatal(err)
			}
			if response.GetCode() != test.want.HTTPStatusCodeInt32() || response.GetError().GetErrorCode() != uint64(test.want.Code) {
				t.Fatalf("response = %+v, want error %+v", response, test.want)
			}
		})
	}

	updateTests := []struct {
		name string
		ids  []uint64
	}{
		{name: "empty roles"},
		{name: "zero role", ids: []uint64{0}},
		{name: "duplicate role", ids: []uint64{108, 108}},
		{name: "owner role", ids: []uint64{internal_service.OrganizationOwnerRoleDefinitionID}},
	}
	for _, test := range updateTests {
		t.Run("update "+test.name, func(t *testing.T) {
			response, err := (&webOrganizationGRPCApi{}).UpdateUserOrganizationRole(ctx, &protos.UpdateUserOrganizationRoleRequest{UserId: 2, RoleDefinitionIds: test.ids})
			if err != nil {
				t.Fatal(err)
			}
			if response.GetCode() != pkg_errors.UpdateUserOrganizationRoleInvalidRole.HTTPStatusCodeInt32() || response.GetError().GetErrorCode() != uint64(pkg_errors.UpdateUserOrganizationRoleInvalidRole.Code) {
				t.Fatalf("response = %+v", response)
			}
		})
	}
}

func TestAddUserToProjectsValidatesRoleSelectionsInline(t *testing.T) {
	ctx, err := middleware_authentication.ContextWithPrincipal(context.Background(), onboardingAuthentication(t, 1, 10))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		ids  []uint64
	}{
		{name: "empty roles"},
		{name: "zero role", ids: []uint64{0}},
		{name: "duplicate role", ids: []uint64{112, 112}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := (&webProjectGRPCApi{}).AddUserToProjects(ctx, &protos.AddUserToProjectsRequest{
				UserId: 2,
				ProjectRoles: []*protos.ProjectRoleAssignment{{
					ProjectId:         100,
					RoleDefinitionIds: test.ids,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if response.GetCode() != pkg_errors.AddUserToProjectsInvalidProjectRole.HTTPStatusCodeInt32() || response.GetError().GetErrorCode() != uint64(pkg_errors.AddUserToProjectsInvalidProjectRole.Code) {
				t.Fatalf("response = %+v", response)
			}
		})
	}
}
