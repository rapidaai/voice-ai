"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

from typing import Optional, Tuple

from pydantic import BaseModel, ConfigDict, Field, model_validator
from starlette.authentication import BaseUser

from app.exceptions.authentication_exception import InvalidAuthorizationTokenException

MAX_ID = 9223372036854775807
IDENTITY_TYPES = {"user", "project", "organization", "service"}


class AuthenticationModel(BaseModel):
    model_config = ConfigDict(
        extra="forbid", frozen=True, populate_by_name=True
    )


class Identity(AuthenticationModel):
    type: str
    id: int
    issuer: str = ""

    @model_validator(mode="after")
    def validate_identity(self):
        if self.type not in IDENTITY_TYPES or not 1 <= self.id <= MAX_ID:
            raise ValueError("invalid authentication identity")
        if self.type == "service":
            if not self.issuer or self.issuer.strip() != self.issuer or "\x00" in self.issuer:
                raise ValueError("service identity requires issuer")
        elif self.issuer:
            raise ValueError("non-service identity forbids issuer")
        return self


class ServiceActor(AuthenticationModel):
    id: int
    issuer: str

    @model_validator(mode="after")
    def validate_actor(self):
        if (
            not 1 <= self.id <= MAX_ID
            or not self.issuer
            or self.issuer.strip() != self.issuer
            or "\x00" in self.issuer
        ):
            raise ValueError("invalid service actor")
        return self


class Tenant(AuthenticationModel):
    organization_id: int = Field(default=0, alias="organizationId")
    project_id: int = Field(default=0, alias="projectId")

    @model_validator(mode="after")
    def validate_tenant(self):
        if not 0 <= self.organization_id <= MAX_ID or not 0 <= self.project_id <= MAX_ID:
            raise ValueError("invalid tenant")
        if self.project_id and not self.organization_id:
            raise ValueError("project tenant requires organization")
        return self


class ProofReference(AuthenticationModel):
    type: str
    value: str

    @model_validator(mode="after")
    def validate_reference(self):
        if self.type not in {"user_session", "credential", "signing_key"}:
            raise ValueError("invalid proof reference type")
        if (
            not self.value
            or self.value.strip() != self.value
            or "\x00" in self.value
            or len(self.value) > 200
        ):
            raise ValueError("invalid proof reference")
        return self


class Proof(AuthenticationModel):
    method: str
    audience: str = ""
    reference: ProofReference


class Principal(AuthenticationModel):
    subject: Identity
    actor: Optional[ServiceActor] = None
    selected_tenant: Tenant = Field(alias="selectedTenant")
    proof: Proof
    authorities: Tuple[str, ...] = Field(default=(), exclude=True)

    @model_validator(mode="after")
    def validate_principal(self):
        if self.actor is not None:
            if self.subject.type == "service" or self.proof.method != "service_assertion":
                raise ValueError("invalid delegated principal")
            if self.proof.audience != "rapida_internal" or self.proof.reference.type != "signing_key":
                raise ValueError("invalid delegated proof")
        elif self.subject.type == "user":
            if (
                self.proof.method != "user_session"
                or self.proof.audience
                or self.proof.reference.type != "user_session"
            ):
                raise ValueError("invalid user proof")
        elif self.subject.type == "project":
            if (
                not self.selected_tenant.organization_id
                or not self.selected_tenant.project_id
                or self.proof.method != "project_credential"
                or self.proof.audience != "management"
                or self.proof.reference.type != "credential"
                or self.proof.reference.value != str(self.subject.id)
            ):
                raise ValueError("invalid project proof")
        elif self.subject.type == "organization":
            if (
                not self.selected_tenant.organization_id
                or self.selected_tenant.project_id
                or self.proof.method != "organization_credential"
                or self.proof.audience
                or self.proof.reference.type != "credential"
                or self.proof.reference.value != str(self.subject.id)
            ):
                raise ValueError("invalid organization proof")
        elif (
            self.proof.method != "service_assertion"
            or self.proof.audience != "rapida_internal"
            or self.proof.reference.type != "signing_key"
        ):
            raise ValueError("invalid service proof")

        if tuple(sorted(set(self.authorities))) != self.authorities:
            raise ValueError("authorities must be sorted and unique")
        if any(
            not authority
            or authority.strip() != authority
            or "\x00" in authority
            for authority in self.authorities
        ):
            raise ValueError("authority is invalid")
        tenant = self.selected_tenant
        if self.subject.type == "user":
            if self.authorities and not tenant.organization_id:
                raise ValueError("user authorities require organization")
            for authority in self.authorities:
                if tenant.project_id:
                    if not authority.startswith(("organization.", "project.")):
                        raise ValueError("user authority has invalid scope")
                elif not authority.startswith("organization."):
                    raise ValueError("user authority has invalid scope")
        elif self.subject.type == "project":
            if not tenant.organization_id or not tenant.project_id or any(
                not authority.startswith("project.") for authority in self.authorities
            ):
                raise ValueError("project principal has invalid scope")
        elif self.subject.type == "organization":
            if not tenant.organization_id or tenant.project_id or self.authorities:
                raise ValueError("organization principal has invalid scope")
        elif self.authorities:
            raise ValueError("service principal forbids authorities")
        return self

    @property
    def caller(self) -> Identity:
        if self.actor is None:
            return self.subject
        return Identity(type="service", id=self.actor.id, issuer=self.actor.issuer)


class Permission(AuthenticationModel):
    id: int
    key: str
    scope: str
    name: str
    description: str = ""


class Role(AuthenticationModel):
    id: int
    key: str
    name: str
    description: str = ""
    scope: str
    subject: str
    definition_type: str = Field(alias="definitionType")
    permissions: Tuple[Permission, ...] = ()


class Project(AuthenticationModel):
    id: int
    name: str
    roles: Tuple[Role, ...] = ()


class BillingPlanQuota(AuthenticationModel):
    id: int
    resource_type: str = Field(alias="resourceType")
    quota_limit: int = Field(alias="quotaLimit")
    unit: str


class BillingPlan(AuthenticationModel):
    id: int
    name: str
    slug: str
    description: str = ""
    is_default: bool = Field(alias="isDefault")
    is_active: bool = Field(alias="isActive")
    sort_order: int = Field(alias="sortOrder")
    price_monthly: int = Field(alias="priceMonthly")
    price_yearly: int = Field(alias="priceYearly")
    currency: str
    stripe_url: str = Field(alias="stripeUrl")
    quotas: Tuple[BillingPlanQuota, ...] = ()


class Organization(AuthenticationModel):
    id: int
    name: str
    roles: Tuple[Role, ...] = ()
    billing_plan: Optional[BillingPlan] = Field(default=None, alias="billingPlan")
    projects: Tuple[Project, ...] = ()


class AuthenticationUser(AuthenticationModel):
    id: int
    name: str
    email: str
    status: str


class Authentication(AuthenticationModel):
    principal: Principal
    user: Optional[AuthenticationUser] = None
    organizations: Tuple[Organization, ...] = ()

    @model_validator(mode="after")
    def validate_aggregate(self):
        subject = self.principal.subject
        if subject.type == "user":
            if self.user is None or self.user.id != subject.id:
                raise ValueError("user subject requires matching profile")
        elif self.user is not None:
            raise ValueError("non-user subject forbids profile")

        organization_ids = [organization.id for organization in self.organizations]
        if len(set(organization_ids)) != len(organization_ids):
            raise ValueError("duplicate organization")

        tenant = self.principal.selected_tenant
        selected_organization = next(
            (organization for organization in self.organizations if organization.id == tenant.organization_id),
            None,
        )
        if tenant.organization_id and selected_organization is None:
            raise ValueError("selected organization is absent")
        selected_project = None
        if selected_organization is not None:
            project_ids = [project.id for project in selected_organization.projects]
            if len(set(project_ids)) != len(project_ids):
                raise ValueError("duplicate project")
            selected_project = next(
                (project for project in selected_organization.projects if project.id == tenant.project_id),
                None,
            )
        if tenant.project_id and selected_project is None:
            raise ValueError("selected project is absent")

        authorities = set()
        if subject.type == "user":
            for organization in self.organizations:
                organization_authorities = (
                    authorities if organization is selected_organization else set()
                )
                for role in organization.roles:
                    self.validate_role(
                        role,
                        "ORGANIZATION",
                        "USER",
                        "organization.",
                        organization_authorities,
                    )
                for project in organization.projects:
                    project_authorities = (
                        authorities if project is selected_project else set()
                    )
                    for role in project.roles:
                        self.validate_role(
                            role,
                            "PROJECT",
                            "USER",
                            "project.",
                            project_authorities,
                        )
        elif subject.type == "project":
            if len(self.organizations) != 1 or selected_project is None:
                raise ValueError("project subject requires one selected project")
            if selected_organization.roles:
                raise ValueError("project subject forbids organization roles")
            for role in selected_project.roles:
                self.validate_role(role, "PROJECT", "PROJECT_TOKEN", "project.", authorities)
        else:
            for organization in self.organizations:
                if organization.roles or any(project.roles for project in organization.projects):
                    raise ValueError("subject forbids assigned roles")

        principal = self.principal.model_copy(update={"authorities": tuple(sorted(authorities))})
        object.__setattr__(self, "principal", principal)
        return self

    @staticmethod
    def validate_role(role, scope, subject, permission_prefix, authorities):
        if role.scope != scope or role.subject != subject:
            raise ValueError("role scope or subject mismatch")
        seen = set()
        for permission in role.permissions:
            if permission.scope != scope or not permission.key.startswith(permission_prefix):
                raise ValueError("permission scope mismatch")
            if permission.key in seen:
                raise ValueError("duplicate role permission")
            seen.add(permission.key)
            authorities.add(permission.key)


class AuthenticatedPrincipal(BaseUser):
    def __init__(
        self,
        authentication: Authentication = None,
        principal: Principal = None,
    ):
        if (authentication is None) == (principal is None):
            raise ValueError("exactly one authentication source is required")
        self.authentication = authentication
        self.principal = authentication.principal if authentication is not None else principal

    @property
    def is_authenticated(self) -> bool:
        return True

    @property
    def display_name(self) -> str:
        return f"{self.principal.subject.type}:{self.principal.subject.id}"

    @property
    def user_id(self) -> int:
        subject = self.principal.subject
        if subject.type != "user":
            raise InvalidAuthorizationTokenException("principal has no user identity")
        return subject.id

    @property
    def organization_id(self) -> int:
        organization_id = self.principal.selected_tenant.organization_id
        if not organization_id:
            raise InvalidAuthorizationTokenException("principal has no organization")
        return organization_id

    @property
    def project_id(self) -> Optional[int]:
        return self.principal.selected_tenant.project_id or None

    @property
    def actor(self) -> dict:
        subject = self.principal.subject
        return {"type": subject.type, "id": subject.id}
