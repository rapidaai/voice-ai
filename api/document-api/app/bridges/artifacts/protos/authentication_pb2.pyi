from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Identity(_message.Message):
    __slots__ = ("type", "id", "issuer")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    type: str
    id: int
    issuer: str
    def __init__(self, type: _Optional[str] = ..., id: _Optional[int] = ..., issuer: _Optional[str] = ...) -> None: ...

class ServiceActor(_message.Message):
    __slots__ = ("id", "issuer")
    ID_FIELD_NUMBER: _ClassVar[int]
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    id: int
    issuer: str
    def __init__(self, id: _Optional[int] = ..., issuer: _Optional[str] = ...) -> None: ...

class Tenant(_message.Message):
    __slots__ = ("organizationId", "projectId")
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    organizationId: int
    projectId: int
    def __init__(self, organizationId: _Optional[int] = ..., projectId: _Optional[int] = ...) -> None: ...

class ProofReference(_message.Message):
    __slots__ = ("type", "value")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: str
    value: str
    def __init__(self, type: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Proof(_message.Message):
    __slots__ = ("method", "audience", "reference")
    METHOD_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    method: str
    audience: str
    reference: ProofReference
    def __init__(self, method: _Optional[str] = ..., audience: _Optional[str] = ..., reference: _Optional[_Union[ProofReference, _Mapping]] = ...) -> None: ...

class Principal(_message.Message):
    __slots__ = ("subject", "actor", "selectedTenant", "proof")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    SELECTEDTENANT_FIELD_NUMBER: _ClassVar[int]
    PROOF_FIELD_NUMBER: _ClassVar[int]
    subject: Identity
    actor: ServiceActor
    selectedTenant: Tenant
    proof: Proof
    def __init__(self, subject: _Optional[_Union[Identity, _Mapping]] = ..., actor: _Optional[_Union[ServiceActor, _Mapping]] = ..., selectedTenant: _Optional[_Union[Tenant, _Mapping]] = ..., proof: _Optional[_Union[Proof, _Mapping]] = ...) -> None: ...

class Permission(_message.Message):
    __slots__ = ("id", "key", "scope", "name", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: int
    key: str
    scope: str
    name: str
    description: str
    def __init__(self, id: _Optional[int] = ..., key: _Optional[str] = ..., scope: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Role(_message.Message):
    __slots__ = ("id", "key", "name", "description", "scope", "subject", "definitionType", "permissions")
    ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    DEFINITIONTYPE_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    id: int
    key: str
    name: str
    description: str
    scope: str
    subject: str
    definitionType: str
    permissions: _containers.RepeatedCompositeFieldContainer[Permission]
    def __init__(self, id: _Optional[int] = ..., key: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., scope: _Optional[str] = ..., subject: _Optional[str] = ..., definitionType: _Optional[str] = ..., permissions: _Optional[_Iterable[_Union[Permission, _Mapping]]] = ...) -> None: ...

class BillingPlanQuota(_message.Message):
    __slots__ = ("id", "resourceType", "quotaLimit", "unit")
    ID_FIELD_NUMBER: _ClassVar[int]
    RESOURCETYPE_FIELD_NUMBER: _ClassVar[int]
    QUOTALIMIT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    id: int
    resourceType: str
    quotaLimit: int
    unit: str
    def __init__(self, id: _Optional[int] = ..., resourceType: _Optional[str] = ..., quotaLimit: _Optional[int] = ..., unit: _Optional[str] = ...) -> None: ...

class BillingPlan(_message.Message):
    __slots__ = ("id", "name", "slug", "description", "isDefault", "isActive", "sortOrder", "priceMonthly", "priceYearly", "currency", "stripeUrl", "quotas")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ISDEFAULT_FIELD_NUMBER: _ClassVar[int]
    ISACTIVE_FIELD_NUMBER: _ClassVar[int]
    SORTORDER_FIELD_NUMBER: _ClassVar[int]
    PRICEMONTHLY_FIELD_NUMBER: _ClassVar[int]
    PRICEYEARLY_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    STRIPEURL_FIELD_NUMBER: _ClassVar[int]
    QUOTAS_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    slug: str
    description: str
    isDefault: bool
    isActive: bool
    sortOrder: int
    priceMonthly: int
    priceYearly: int
    currency: str
    stripeUrl: str
    quotas: _containers.RepeatedCompositeFieldContainer[BillingPlanQuota]
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., slug: _Optional[str] = ..., description: _Optional[str] = ..., isDefault: bool = ..., isActive: bool = ..., sortOrder: _Optional[int] = ..., priceMonthly: _Optional[int] = ..., priceYearly: _Optional[int] = ..., currency: _Optional[str] = ..., stripeUrl: _Optional[str] = ..., quotas: _Optional[_Iterable[_Union[BillingPlanQuota, _Mapping]]] = ...) -> None: ...

class Project(_message.Message):
    __slots__ = ("id", "name", "roles")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ...) -> None: ...

class Organization(_message.Message):
    __slots__ = ("id", "name", "roles", "billingPlan", "projects")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    BILLINGPLAN_FIELD_NUMBER: _ClassVar[int]
    PROJECTS_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    billingPlan: BillingPlan
    projects: _containers.RepeatedCompositeFieldContainer[Project]
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ..., billingPlan: _Optional[_Union[BillingPlan, _Mapping]] = ..., projects: _Optional[_Iterable[_Union[Project, _Mapping]]] = ...) -> None: ...

class User(_message.Message):
    __slots__ = ("id", "name", "email", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    email: str
    status: str
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., email: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class Authentication(_message.Message):
    __slots__ = ("principal", "user", "organizations")
    PRINCIPAL_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONS_FIELD_NUMBER: _ClassVar[int]
    principal: Principal
    user: User
    organizations: _containers.RepeatedCompositeFieldContainer[Organization]
    def __init__(self, principal: _Optional[_Union[Principal, _Mapping]] = ..., user: _Optional[_Union[User, _Mapping]] = ..., organizations: _Optional[_Iterable[_Union[Organization, _Mapping]]] = ...) -> None: ...
