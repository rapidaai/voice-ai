// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package authentication

import (
	authentication_proto "github.com/rapidaai/protos/authentication"

	pkg_errors "github.com/rapidaai/pkg/errors"
)

// Proto returns a detached wire representation of the Authentication aggregate.
func (authentication Authentication) Proto() *authentication_proto.Authentication {
	principal := authentication.Principal()
	subject := principal.Subject()
	tenant := principal.Tenant()
	proof := principal.Proof()
	value := &authentication_proto.Authentication{
		Principal: &authentication_proto.Principal{
			Subject: &authentication_proto.Identity{Type: string(subject.Type), Id: subject.ID, Issuer: subject.Issuer},
			SelectedTenant: &authentication_proto.Tenant{
				OrganizationId: tenant.OrganizationID,
				ProjectId:      tenant.ProjectID,
			},
			Proof: &authentication_proto.Proof{
				Method:   string(proof.Method),
				Audience: string(proof.Audience),
				Reference: &authentication_proto.ProofReference{
					Type:  string(proof.Reference.Type),
					Value: proof.Reference.Value,
				},
			},
		},
	}
	if actor, ok := principal.Actor(); ok {
		value.Principal.Actor = &authentication_proto.ServiceActor{Id: actor.ID, Issuer: actor.Issuer}
	}
	if user, ok := authentication.User(); ok {
		value.User = &authentication_proto.User{Id: user.ID, Name: user.Name, Email: user.Email, Status: user.Status}
	}
	for _, organization := range authentication.Organizations() {
		wireOrganization := &authentication_proto.Organization{Id: organization.ID, Name: organization.Name}
		for _, role := range organization.Roles {
			wireRole := &authentication_proto.Role{
				Id: role.ID, Key: role.Key, Name: role.Name, Description: role.Description,
				Scope: role.Scope, Subject: role.Subject, DefinitionType: role.DefinitionType,
			}
			for _, permission := range role.Permissions {
				wireRole.Permissions = append(wireRole.Permissions, &authentication_proto.Permission{
					Id: permission.ID, Key: permission.Key, Scope: permission.Scope,
					Name: permission.Name, Description: permission.Description,
				})
			}
			wireOrganization.Roles = append(wireOrganization.Roles, wireRole)
		}
		if organization.BillingPlan != nil {
			plan := organization.BillingPlan
			wireOrganization.BillingPlan = &authentication_proto.BillingPlan{
				Id: plan.ID, Name: plan.Name, Slug: plan.Slug, Description: plan.Description,
				IsDefault: plan.IsDefault, IsActive: plan.IsActive, SortOrder: plan.SortOrder,
				PriceMonthly: plan.PriceMonthly, PriceYearly: plan.PriceYearly,
				Currency: plan.Currency, StripeUrl: plan.StripeURL,
			}
			for _, quota := range plan.Quotas {
				wireOrganization.BillingPlan.Quotas = append(wireOrganization.BillingPlan.Quotas, &authentication_proto.BillingPlanQuota{
					Id: quota.ID, ResourceType: quota.ResourceType, QuotaLimit: quota.QuotaLimit, Unit: quota.Unit,
				})
			}
		}
		for _, project := range organization.Projects {
			wireProject := &authentication_proto.Project{Id: project.ID, Name: project.Name}
			for _, role := range project.Roles {
				wireRole := &authentication_proto.Role{
					Id: role.ID, Key: role.Key, Name: role.Name, Description: role.Description,
					Scope: role.Scope, Subject: role.Subject, DefinitionType: role.DefinitionType,
				}
				for _, permission := range role.Permissions {
					wireRole.Permissions = append(wireRole.Permissions, &authentication_proto.Permission{
						Id: permission.ID, Key: permission.Key, Scope: permission.Scope,
						Name: permission.Name, Description: permission.Description,
					})
				}
				wireProject.Roles = append(wireProject.Roles, wireRole)
			}
			wireOrganization.Projects = append(wireOrganization.Projects, wireProject)
		}
		value.Organizations = append(value.Organizations, wireOrganization)
	}
	return value
}

// AuthenticationFromProto validates a wire aggregate and derives its Principal authorities once.
func AuthenticationFromProto(source *authentication_proto.Authentication) (Authentication, error) {
	if source == nil || source.Principal == nil || source.Principal.Subject == nil ||
		source.Principal.SelectedTenant == nil || source.Principal.Proof == nil || source.Principal.Proof.Reference == nil {
		return Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	principal := source.Principal
	subject := Identity{Type: IdentityType(principal.Subject.Type), ID: principal.Subject.Id, Issuer: principal.Subject.Issuer}
	tenant := Tenant{OrganizationID: principal.SelectedTenant.OrganizationId, ProjectID: principal.SelectedTenant.ProjectId}
	proof := Proof{
		Method: AuthenticationMethod(principal.Proof.Method), Audience: Audience(principal.Proof.Audience),
		Reference: ProofReference{Type: ProofReferenceType(principal.Proof.Reference.Type), Value: principal.Proof.Reference.Value},
	}
	var user *User
	if source.User != nil {
		user = &User{ID: source.User.Id, Name: source.User.Name, Email: source.User.Email, Status: source.User.Status}
	}
	organizations := make([]Organization, 0, len(source.Organizations))
	for _, wireOrganization := range source.Organizations {
		if wireOrganization == nil {
			return Authentication{}, pkg_errors.AuthenticationInvalidError{}
		}
		organization := Organization{ID: wireOrganization.Id, Name: wireOrganization.Name}
		for _, wireRole := range wireOrganization.Roles {
			if wireRole == nil {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			role := Role{
				ID: wireRole.Id, Key: wireRole.Key, Name: wireRole.Name, Description: wireRole.Description,
				Scope: wireRole.Scope, Subject: wireRole.Subject, DefinitionType: wireRole.DefinitionType,
			}
			for _, wirePermission := range wireRole.Permissions {
				if wirePermission == nil {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				role.Permissions = append(role.Permissions, Permission{
					ID: wirePermission.Id, Key: wirePermission.Key, Scope: wirePermission.Scope,
					Name: wirePermission.Name, Description: wirePermission.Description,
				})
			}
			organization.Roles = append(organization.Roles, role)
		}
		if wireOrganization.BillingPlan != nil {
			wirePlan := wireOrganization.BillingPlan
			plan := &BillingPlan{
				ID: wirePlan.Id, Name: wirePlan.Name, Slug: wirePlan.Slug, Description: wirePlan.Description,
				IsDefault: wirePlan.IsDefault, IsActive: wirePlan.IsActive, SortOrder: wirePlan.SortOrder,
				PriceMonthly: wirePlan.PriceMonthly, PriceYearly: wirePlan.PriceYearly,
				Currency: wirePlan.Currency, StripeURL: wirePlan.StripeUrl,
			}
			for _, wireQuota := range wirePlan.Quotas {
				if wireQuota == nil {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				plan.Quotas = append(plan.Quotas, BillingPlanQuota{
					ID: wireQuota.Id, ResourceType: wireQuota.ResourceType, QuotaLimit: wireQuota.QuotaLimit, Unit: wireQuota.Unit,
				})
			}
			organization.BillingPlan = plan
		}
		for _, wireProject := range wireOrganization.Projects {
			if wireProject == nil {
				return Authentication{}, pkg_errors.AuthenticationInvalidError{}
			}
			project := Project{ID: wireProject.Id, Name: wireProject.Name}
			for _, wireRole := range wireProject.Roles {
				if wireRole == nil {
					return Authentication{}, pkg_errors.AuthenticationInvalidError{}
				}
				role := Role{
					ID: wireRole.Id, Key: wireRole.Key, Name: wireRole.Name, Description: wireRole.Description,
					Scope: wireRole.Scope, Subject: wireRole.Subject, DefinitionType: wireRole.DefinitionType,
				}
				for _, wirePermission := range wireRole.Permissions {
					if wirePermission == nil {
						return Authentication{}, pkg_errors.AuthenticationInvalidError{}
					}
					role.Permissions = append(role.Permissions, Permission{
						ID: wirePermission.Id, Key: wirePermission.Key, Scope: wirePermission.Scope,
						Name: wirePermission.Name, Description: wirePermission.Description,
					})
				}
				project.Roles = append(project.Roles, role)
			}
			organization.Projects = append(organization.Projects, project)
		}
		organizations = append(organizations, organization)
	}
	if principal.Actor != nil {
		return NewDelegatedAuthentication(subject, ServiceActor{ID: principal.Actor.Id, Issuer: principal.Actor.Issuer}, tenant, proof, user, organizations)
	}
	return NewDirectAuthentication(subject, tenant, proof, user, organizations)
}
