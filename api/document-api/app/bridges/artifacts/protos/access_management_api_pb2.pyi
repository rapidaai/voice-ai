import datetime

import app.bridges.artifacts.protos.common_pb2 as _common_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RoleScope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROLE_SCOPE_UNSPECIFIED: _ClassVar[RoleScope]
    ROLE_SCOPE_ORGANIZATION: _ClassVar[RoleScope]
    ROLE_SCOPE_PROJECT: _ClassVar[RoleScope]

class RoleSubject(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROLE_SUBJECT_UNSPECIFIED: _ClassVar[RoleSubject]
    ROLE_SUBJECT_USER: _ClassVar[RoleSubject]
    ROLE_SUBJECT_PROJECT_TOKEN: _ClassVar[RoleSubject]

class RoleDefinitionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROLE_DEFINITION_TYPE_UNSPECIFIED: _ClassVar[RoleDefinitionType]
    ROLE_DEFINITION_TYPE_BUILT_IN: _ClassVar[RoleDefinitionType]
    ROLE_DEFINITION_TYPE_CUSTOM: _ClassVar[RoleDefinitionType]
ROLE_SCOPE_UNSPECIFIED: RoleScope
ROLE_SCOPE_ORGANIZATION: RoleScope
ROLE_SCOPE_PROJECT: RoleScope
ROLE_SUBJECT_UNSPECIFIED: RoleSubject
ROLE_SUBJECT_USER: RoleSubject
ROLE_SUBJECT_PROJECT_TOKEN: RoleSubject
ROLE_DEFINITION_TYPE_UNSPECIFIED: RoleDefinitionType
ROLE_DEFINITION_TYPE_BUILT_IN: RoleDefinitionType
ROLE_DEFINITION_TYPE_CUSTOM: RoleDefinitionType

class PermissionDefinition(_message.Message):
    __slots__ = ("id", "key", "scope", "name", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: int
    key: str
    scope: RoleScope
    name: str
    description: str
    def __init__(self, id: _Optional[int] = ..., key: _Optional[str] = ..., scope: _Optional[_Union[RoleScope, str]] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class RoleDefinition(_message.Message):
    __slots__ = ("id", "key", "name", "description", "scope", "subject", "definitionType", "createdDate", "updatedDate", "createdActor", "updatedActor", "permissions")
    ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    DEFINITIONTYPE_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    id: int
    key: str
    name: str
    description: str
    scope: RoleScope
    subject: RoleSubject
    definitionType: RoleDefinitionType
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    permissions: _containers.RepeatedCompositeFieldContainer[PermissionDefinition]
    def __init__(self, id: _Optional[int] = ..., key: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., scope: _Optional[_Union[RoleScope, str]] = ..., subject: _Optional[_Union[RoleSubject, str]] = ..., definitionType: _Optional[_Union[RoleDefinitionType, str]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., permissions: _Optional[_Iterable[_Union[PermissionDefinition, _Mapping]]] = ...) -> None: ...

class ListPermissionsRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class ListPermissionsResponse(_message.Message):
    __slots__ = ("code", "success", "permissions", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    permissions: _containers.RepeatedCompositeFieldContainer[PermissionDefinition]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., permissions: _Optional[_Iterable[_Union[PermissionDefinition, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class ListRolesRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class ListRolesResponse(_message.Message):
    __slots__ = ("code", "success", "roles", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    roles: _containers.RepeatedCompositeFieldContainer[RoleDefinition]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., roles: _Optional[_Iterable[_Union[RoleDefinition, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class GetRoleRequest(_message.Message):
    __slots__ = ("roleDefinitionId",)
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    roleDefinitionId: int
    def __init__(self, roleDefinitionId: _Optional[int] = ...) -> None: ...

class GetRoleResponse(_message.Message):
    __slots__ = ("code", "success", "role", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    role: RoleDefinition
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., role: _Optional[_Union[RoleDefinition, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreateCustomRoleRequest(_message.Message):
    __slots__ = ("key", "name", "description", "scope", "subject", "permissionDefinitionIds")
    KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONDEFINITIONIDS_FIELD_NUMBER: _ClassVar[int]
    key: str
    name: str
    description: str
    scope: RoleScope
    subject: RoleSubject
    permissionDefinitionIds: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, key: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., scope: _Optional[_Union[RoleScope, str]] = ..., subject: _Optional[_Union[RoleSubject, str]] = ..., permissionDefinitionIds: _Optional[_Iterable[int]] = ...) -> None: ...

class CreateCustomRoleResponse(_message.Message):
    __slots__ = ("code", "success", "role", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    role: RoleDefinition
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., role: _Optional[_Union[RoleDefinition, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class UpdateCustomRoleRequest(_message.Message):
    __slots__ = ("roleDefinitionId", "name", "description")
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    roleDefinitionId: int
    name: str
    description: str
    def __init__(self, roleDefinitionId: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class UpdateCustomRoleResponse(_message.Message):
    __slots__ = ("code", "success", "role", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    role: RoleDefinition
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., role: _Optional[_Union[RoleDefinition, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class RoleReductionImpact(_message.Message):
    __slots__ = ("organizationId", "projectId", "subject", "subjectId", "assignmentStatus")
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    SUBJECTID_FIELD_NUMBER: _ClassVar[int]
    ASSIGNMENTSTATUS_FIELD_NUMBER: _ClassVar[int]
    organizationId: int
    projectId: int
    subject: RoleSubject
    subjectId: int
    assignmentStatus: str
    def __init__(self, organizationId: _Optional[int] = ..., projectId: _Optional[int] = ..., subject: _Optional[_Union[RoleSubject, str]] = ..., subjectId: _Optional[int] = ..., assignmentStatus: _Optional[str] = ...) -> None: ...

class RoleReductionPreview(_message.Message):
    __slots__ = ("resultingRole", "impacts", "confirmationDigest")
    RESULTINGROLE_FIELD_NUMBER: _ClassVar[int]
    IMPACTS_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATIONDIGEST_FIELD_NUMBER: _ClassVar[int]
    resultingRole: RoleDefinition
    impacts: _containers.RepeatedCompositeFieldContainer[RoleReductionImpact]
    confirmationDigest: str
    def __init__(self, resultingRole: _Optional[_Union[RoleDefinition, _Mapping]] = ..., impacts: _Optional[_Iterable[_Union[RoleReductionImpact, _Mapping]]] = ..., confirmationDigest: _Optional[str] = ...) -> None: ...

class ReduceCustomRolePermissionsRequest(_message.Message):
    __slots__ = ("roleDefinitionId", "permissionDefinitionIds", "reason", "confirmationDigest", "preview")
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONDEFINITIONIDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATIONDIGEST_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    roleDefinitionId: int
    permissionDefinitionIds: _containers.RepeatedScalarFieldContainer[int]
    reason: str
    confirmationDigest: str
    preview: bool
    def __init__(self, roleDefinitionId: _Optional[int] = ..., permissionDefinitionIds: _Optional[_Iterable[int]] = ..., reason: _Optional[str] = ..., confirmationDigest: _Optional[str] = ..., preview: bool = ...) -> None: ...

class ReduceCustomRolePermissionsResponse(_message.Message):
    __slots__ = ("code", "success", "role", "error", "correlationId", "preview")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CORRELATIONID_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    role: RoleDefinition
    error: _common_pb2.Error
    correlationId: str
    preview: RoleReductionPreview
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., role: _Optional[_Union[RoleDefinition, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., correlationId: _Optional[str] = ..., preview: _Optional[_Union[RoleReductionPreview, _Mapping]] = ...) -> None: ...

class ArchiveCustomRoleRequest(_message.Message):
    __slots__ = ("roleDefinitionId", "reason")
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    roleDefinitionId: int
    reason: str
    def __init__(self, roleDefinitionId: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ArchiveCustomRoleResponse(_message.Message):
    __slots__ = ("code", "success", "roleDefinitionId", "error", "correlationId")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CORRELATIONID_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    roleDefinitionId: int
    error: _common_pb2.Error
    correlationId: str
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., roleDefinitionId: _Optional[int] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., correlationId: _Optional[str] = ...) -> None: ...
