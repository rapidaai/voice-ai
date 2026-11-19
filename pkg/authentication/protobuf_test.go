// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication_test

import (
	"reflect"
	"testing"

	"github.com/rapidaai/pkg/authentication"
	pkg_errors "github.com/rapidaai/pkg/errors"
	authentication_proto "github.com/rapidaai/protos/authentication"
)

func TestAuthenticationProtoRoundTripDerivesPermissionedDelegatedPrincipal(t *testing.T) {
	value, err := authentication.NewDelegatedAuthentication(
		authentication.Identity{Type: authentication.IdentityUser, ID: 7},
		authentication.ServiceActor{ID: 8, Issuer: "workflow-api"},
		authentication.Tenant{OrganizationID: 9, ProjectID: 10},
		authentication.Proof{Method: authentication.AuthenticationServiceAssertion, Audience: authentication.AudienceRapidaInternal, Reference: authentication.ProofReference{Type: authentication.ProofReferenceSigningKey, Value: "workflow-key-1"}},
		&authentication.User{ID: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
		[]authentication.Organization{{
			ID: 9, Name: "Organization",
			Roles: []authentication.Role{{
				ID: 11, Key: "organization_reader", Name: "Organization Reader", Scope: authentication.RoleScopeOrganization,
				Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionBuiltIn,
				Permissions: []authentication.Permission{{ID: 1, Key: "organization.projects.read", Scope: authentication.RoleScopeOrganization, Name: "Read projects"}},
			}},
			BillingPlan: &authentication.BillingPlan{ID: 12, Name: "Growth", Slug: "growth", IsActive: true, Currency: "USD", Quotas: []authentication.BillingPlanQuota{{ID: 13, ResourceType: "projects", QuotaLimit: 10, Unit: "project"}}},
			Projects: []authentication.Project{{
				ID: 10, Name: "Project", Roles: []authentication.Role{{
					ID: 14, Key: "project_reader", Name: "Project Reader", Scope: authentication.RoleScopeProject,
					Subject: authentication.RoleSubjectUser, DefinitionType: authentication.RoleDefinitionCustom,
					Permissions: []authentication.Permission{{ID: 2, Key: "project.resources.read", Scope: authentication.RoleScopeProject, Name: "Read resources"}},
				}},
			}},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	wire := value.Proto()
	if wire.GetPrincipal().GetActor().GetIssuer() != "workflow-api" {
		t.Fatalf("Proto() actor = %+v", wire.GetPrincipal().GetActor())
	}
	decoded, err := authentication.AuthenticationFromProto(wire)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Principal().Subject() != value.Principal().Subject() || decoded.Principal().Tenant() != value.Principal().Tenant() ||
		!reflect.DeepEqual(decoded.Principal().Authorities(), []authentication.PermissionKey{"organization.projects.read", "project.resources.read"}) ||
		!reflect.DeepEqual(decoded.Organizations(), value.Organizations()) {
		t.Fatalf("round trip = %+v", decoded)
	}
}

func TestAuthenticationFromProtoRejectsMalformedGraph(t *testing.T) {
	tests := []*authentication_proto.Authentication{
		nil,
		{},
		{Principal: &authentication_proto.Principal{}},
		{
			Principal: &authentication_proto.Principal{
				Subject:        &authentication_proto.Identity{Type: "user", Id: 7},
				SelectedTenant: &authentication_proto.Tenant{OrganizationId: 9},
				Proof:          &authentication_proto.Proof{Method: "user_session", Reference: &authentication_proto.ProofReference{Type: "user_session", Value: "8"}},
			},
			User: &authentication_proto.User{Id: 7, Name: "Ada", Email: "ada@example.com", Status: "ACTIVE"},
			Organizations: []*authentication_proto.Organization{{
				Id: 9, Name: "Organization", Roles: []*authentication_proto.Role{{
					Id: 11, Key: "bad", Name: "Bad", Scope: "ORGANIZATION", Subject: "USER", DefinitionType: "BUILT_IN",
					Permissions: []*authentication_proto.Permission{{Id: 1, Key: "project.wrong", Scope: "ORGANIZATION", Name: "Wrong"}},
				}},
			}},
		},
	}
	for index, source := range tests {
		if _, err := authentication.AuthenticationFromProto(source); err == nil {
			t.Fatalf("case %d accepted", index)
		} else if _, ok := err.(pkg_errors.AuthenticationInvalidError); !ok {
			t.Fatalf("case %d error = %T", index, err)
		}
	}
}
