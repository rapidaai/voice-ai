import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
import app.bridges.artifacts.protos.common_pb2 as _common_pb2
import app.bridges.artifacts.protos.authentication_pb2 as _authentication_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuthenticateRequest(_message.Message):
    __slots__ = ("email", "password")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class RegisterUserRequest(_message.Message):
    __slots__ = ("email", "password", "name")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    name: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class Token(_message.Message):
    __slots__ = ("id", "token", "tokenType", "isExpired")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    TOKENTYPE_FIELD_NUMBER: _ClassVar[int]
    ISEXPIRED_FIELD_NUMBER: _ClassVar[int]
    id: int
    token: str
    tokenType: str
    isExpired: bool
    def __init__(self, id: _Optional[int] = ..., token: _Optional[str] = ..., tokenType: _Optional[str] = ..., isExpired: bool = ...) -> None: ...

class AuthenticateResponse(_message.Message):
    __slots__ = ("code", "success", "error", "authentication", "token")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATION_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    authentication: _authentication_pb2.Authentication
    token: Token
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., authentication: _Optional[_Union[_authentication_pb2.Authentication, _Mapping]] = ..., token: _Optional[_Union[Token, _Mapping]] = ...) -> None: ...

class ForgotPasswordRequest(_message.Message):
    __slots__ = ("email",)
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    email: str
    def __init__(self, email: _Optional[str] = ...) -> None: ...

class ForgotPasswordResponse(_message.Message):
    __slots__ = ("code", "success", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class ChangePasswordRequest(_message.Message):
    __slots__ = ("oldPassword", "password")
    OLDPASSWORD_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    oldPassword: str
    password: str
    def __init__(self, oldPassword: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class ChangePasswordResponse(_message.Message):
    __slots__ = ("code", "success", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreatePasswordRequest(_message.Message):
    __slots__ = ("token", "password")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    token: str
    password: str
    def __init__(self, token: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class CreatePasswordResponse(_message.Message):
    __slots__ = ("code", "success", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class VerifyTokenRequest(_message.Message):
    __slots__ = ("user", "project", "organization")
    class UserCredential(_message.Message):
        __slots__ = ("token", "userId", "selectedOrganizationId", "selectedProjectId")
        TOKEN_FIELD_NUMBER: _ClassVar[int]
        USERID_FIELD_NUMBER: _ClassVar[int]
        SELECTEDORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
        SELECTEDPROJECTID_FIELD_NUMBER: _ClassVar[int]
        token: str
        userId: int
        selectedOrganizationId: int
        selectedProjectId: int
        def __init__(self, token: _Optional[str] = ..., userId: _Optional[int] = ..., selectedOrganizationId: _Optional[int] = ..., selectedProjectId: _Optional[int] = ...) -> None: ...
    class ProjectCredential(_message.Message):
        __slots__ = ("token",)
        TOKEN_FIELD_NUMBER: _ClassVar[int]
        token: str
        def __init__(self, token: _Optional[str] = ...) -> None: ...
    class OrganizationCredential(_message.Message):
        __slots__ = ("token",)
        TOKEN_FIELD_NUMBER: _ClassVar[int]
        token: str
        def __init__(self, token: _Optional[str] = ...) -> None: ...
    USER_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATION_FIELD_NUMBER: _ClassVar[int]
    user: VerifyTokenRequest.UserCredential
    project: VerifyTokenRequest.ProjectCredential
    organization: VerifyTokenRequest.OrganizationCredential
    def __init__(self, user: _Optional[_Union[VerifyTokenRequest.UserCredential, _Mapping]] = ..., project: _Optional[_Union[VerifyTokenRequest.ProjectCredential, _Mapping]] = ..., organization: _Optional[_Union[VerifyTokenRequest.OrganizationCredential, _Mapping]] = ...) -> None: ...

class VerifyTokenResponse(_message.Message):
    __slots__ = ("authentication",)
    AUTHENTICATION_FIELD_NUMBER: _ClassVar[int]
    authentication: _authentication_pb2.Authentication
    def __init__(self, authentication: _Optional[_Union[_authentication_pb2.Authentication, _Mapping]] = ...) -> None: ...

class AuthorizeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetUserRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetUserResponse(_message.Message):
    __slots__ = ("code", "success", "data")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.User
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.User, _Mapping]] = ...) -> None: ...

class UpdateUserRequest(_message.Message):
    __slots__ = ("email", "name")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    email: str
    name: str
    def __init__(self, email: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class UpdateUserResponse(_message.Message):
    __slots__ = ("code", "success", "data")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.User
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.User, _Mapping]] = ...) -> None: ...

class SocialAuthenticationRequest(_message.Message):
    __slots__ = ("state", "code")
    STATE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    state: str
    code: str
    def __init__(self, state: _Optional[str] = ..., code: _Optional[str] = ...) -> None: ...

class GetAllUserRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class GetAllUserResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[_common_pb2.User]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[_common_pb2.User, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class CreateOrganizationRequest(_message.Message):
    __slots__ = ("organizationName", "organizationSize", "organizationIndustry", "organizationContact")
    ORGANIZATIONNAME_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONSIZE_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONINDUSTRY_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONCONTACT_FIELD_NUMBER: _ClassVar[int]
    organizationName: str
    organizationSize: str
    organizationIndustry: str
    organizationContact: str
    def __init__(self, organizationName: _Optional[str] = ..., organizationSize: _Optional[str] = ..., organizationIndustry: _Optional[str] = ..., organizationContact: _Optional[str] = ...) -> None: ...

class UpdateOrganizationRequest(_message.Message):
    __slots__ = ("organizationId", "organizationName", "organizationIndustry", "organizationContact")
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONNAME_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONINDUSTRY_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONCONTACT_FIELD_NUMBER: _ClassVar[int]
    organizationId: int
    organizationName: str
    organizationIndustry: str
    organizationContact: str
    def __init__(self, organizationId: _Optional[int] = ..., organizationName: _Optional[str] = ..., organizationIndustry: _Optional[str] = ..., organizationContact: _Optional[str] = ...) -> None: ...

class GetOrganizationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetOrganizationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.Organization
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.Organization, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class CreateOrganizationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.Organization
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.Organization, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class UpdateOrganizationResponse(_message.Message):
    __slots__ = ("code", "success", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class UpdateBillingInformationRequest(_message.Message):
    __slots__ = ("paymentMethod", "billingInterval", "taxInformation", "address", "email")
    class BillingInterval(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        Annually: _ClassVar[UpdateBillingInformationRequest.BillingInterval]
        Monthly: _ClassVar[UpdateBillingInformationRequest.BillingInterval]
    Annually: UpdateBillingInformationRequest.BillingInterval
    Monthly: UpdateBillingInformationRequest.BillingInterval
    PAYMENTMETHOD_FIELD_NUMBER: _ClassVar[int]
    BILLINGINTERVAL_FIELD_NUMBER: _ClassVar[int]
    TAXINFORMATION_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    paymentMethod: str
    billingInterval: UpdateBillingInformationRequest.BillingInterval
    taxInformation: str
    address: str
    email: str
    def __init__(self, paymentMethod: _Optional[str] = ..., billingInterval: _Optional[_Union[UpdateBillingInformationRequest.BillingInterval, str]] = ..., taxInformation: _Optional[str] = ..., address: _Optional[str] = ..., email: _Optional[str] = ...) -> None: ...

class Project(_message.Message):
    __slots__ = ("id", "name", "description", "members", "status", "createdDate")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    description: str
    members: _containers.RepeatedCompositeFieldContainer[_common_pb2.User]
    status: str
    createdDate: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., members: _Optional[_Iterable[_Union[_common_pb2.User, _Mapping]]] = ..., status: _Optional[str] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateProjectRequest(_message.Message):
    __slots__ = ("projectName", "projectDescription")
    PROJECTNAME_FIELD_NUMBER: _ClassVar[int]
    PROJECTDESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    projectName: str
    projectDescription: str
    def __init__(self, projectName: _Optional[str] = ..., projectDescription: _Optional[str] = ...) -> None: ...

class CreateProjectResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Project
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Project, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class UpdateProjectRequest(_message.Message):
    __slots__ = ("projectId", "projectName", "projectDescription")
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    PROJECTNAME_FIELD_NUMBER: _ClassVar[int]
    PROJECTDESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    projectId: int
    projectName: str
    projectDescription: str
    def __init__(self, projectId: _Optional[int] = ..., projectName: _Optional[str] = ..., projectDescription: _Optional[str] = ...) -> None: ...

class UpdateProjectResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Project
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Project, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetProjectRequest(_message.Message):
    __slots__ = ("projectId",)
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    projectId: int
    def __init__(self, projectId: _Optional[int] = ...) -> None: ...

class GetProjectResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: Project
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[Project, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllProjectRequest(_message.Message):
    __slots__ = ("paginate", "criterias")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ...) -> None: ...

class GetAllProjectResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[Project]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[Project, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class ProjectRoleAssignment(_message.Message):
    __slots__ = ("projectId", "roleDefinitionIds")
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ROLEDEFINITIONIDS_FIELD_NUMBER: _ClassVar[int]
    projectId: int
    roleDefinitionIds: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, projectId: _Optional[int] = ..., roleDefinitionIds: _Optional[_Iterable[int]] = ...) -> None: ...

class InviteUserToOrganizationRequest(_message.Message):
    __slots__ = ("email", "projectRoles", "organizationRoleDefinitionIds")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PROJECTROLES_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONROLEDEFINITIONIDS_FIELD_NUMBER: _ClassVar[int]
    email: str
    projectRoles: _containers.RepeatedCompositeFieldContainer[ProjectRoleAssignment]
    organizationRoleDefinitionIds: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, email: _Optional[str] = ..., projectRoles: _Optional[_Iterable[_Union[ProjectRoleAssignment, _Mapping]]] = ..., organizationRoleDefinitionIds: _Optional[_Iterable[int]] = ...) -> None: ...

class InviteUserToOrganizationResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _common_pb2.User
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[_common_pb2.User, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class AddUserToProjectsRequest(_message.Message):
    __slots__ = ("userId", "projectRoles")
    USERID_FIELD_NUMBER: _ClassVar[int]
    PROJECTROLES_FIELD_NUMBER: _ClassVar[int]
    userId: int
    projectRoles: _containers.RepeatedCompositeFieldContainer[ProjectRoleAssignment]
    def __init__(self, userId: _Optional[int] = ..., projectRoles: _Optional[_Iterable[_Union[ProjectRoleAssignment, _Mapping]]] = ...) -> None: ...

class UpdateUserOrganizationRoleRequest(_message.Message):
    __slots__ = ("userId", "roleDefinitionIds")
    USERID_FIELD_NUMBER: _ClassVar[int]
    ROLEDEFINITIONIDS_FIELD_NUMBER: _ClassVar[int]
    userId: int
    roleDefinitionIds: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, userId: _Optional[int] = ..., roleDefinitionIds: _Optional[_Iterable[int]] = ...) -> None: ...

class UpdateUserOrganizationRoleResponse(_message.Message):
    __slots__ = ("code", "success", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class DeleteUserFromOrganizationRequest(_message.Message):
    __slots__ = ("userId",)
    USERID_FIELD_NUMBER: _ClassVar[int]
    userId: int
    def __init__(self, userId: _Optional[int] = ...) -> None: ...

class DeleteUserFromOrganizationResponse(_message.Message):
    __slots__ = ("code", "success", "id", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    id: int
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., id: _Optional[int] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class DeleteUserFromProjectRequest(_message.Message):
    __slots__ = ("userId", "projectId")
    USERID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    userId: int
    projectId: int
    def __init__(self, userId: _Optional[int] = ..., projectId: _Optional[int] = ...) -> None: ...

class DeleteUserFromProjectResponse(_message.Message):
    __slots__ = ("code", "success", "id", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    id: int
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., id: _Optional[int] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class ArchiveProjectRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class ArchiveProjectResponse(_message.Message):
    __slots__ = ("code", "success", "id", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    id: int
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., id: _Optional[int] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class AddUserToProjectsResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[Project]
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[Project, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class ProjectCredential(_message.Message):
    __slots__ = ("id", "projectId", "organizationId", "name", "key", "status", "createdActor", "updatedActor", "createdDate", "updatedDate", "roleDefinitionId", "roleName")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    ORGANIZATIONID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    UPDATEDACTOR_FIELD_NUMBER: _ClassVar[int]
    CREATEDDATE_FIELD_NUMBER: _ClassVar[int]
    UPDATEDDATE_FIELD_NUMBER: _ClassVar[int]
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    ROLENAME_FIELD_NUMBER: _ClassVar[int]
    id: int
    projectId: int
    organizationId: int
    name: str
    key: str
    status: str
    createdActor: _common_pb2.AuditActor
    updatedActor: _common_pb2.AuditActor
    createdDate: _timestamp_pb2.Timestamp
    updatedDate: _timestamp_pb2.Timestamp
    roleDefinitionId: int
    roleName: str
    def __init__(self, id: _Optional[int] = ..., projectId: _Optional[int] = ..., organizationId: _Optional[int] = ..., name: _Optional[str] = ..., key: _Optional[str] = ..., status: _Optional[str] = ..., createdActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., updatedActor: _Optional[_Union[_common_pb2.AuditActor, _Mapping]] = ..., createdDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updatedDate: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., roleDefinitionId: _Optional[int] = ..., roleName: _Optional[str] = ...) -> None: ...

class CreateProjectCredentialRequest(_message.Message):
    __slots__ = ("projectId", "name", "roleDefinitionId")
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROLEDEFINITIONID_FIELD_NUMBER: _ClassVar[int]
    projectId: int
    name: str
    roleDefinitionId: int
    def __init__(self, projectId: _Optional[int] = ..., name: _Optional[str] = ..., roleDefinitionId: _Optional[int] = ...) -> None: ...

class GetAllProjectCredentialRequest(_message.Message):
    __slots__ = ("paginate", "criterias", "projectId")
    PAGINATE_FIELD_NUMBER: _ClassVar[int]
    CRITERIAS_FIELD_NUMBER: _ClassVar[int]
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    paginate: _common_pb2.Paginate
    criterias: _containers.RepeatedCompositeFieldContainer[_common_pb2.Criteria]
    projectId: int
    def __init__(self, paginate: _Optional[_Union[_common_pb2.Paginate, _Mapping]] = ..., criterias: _Optional[_Iterable[_Union[_common_pb2.Criteria, _Mapping]]] = ..., projectId: _Optional[int] = ...) -> None: ...

class RevokeProjectCredentialRequest(_message.Message):
    __slots__ = ("projectId", "credentialId")
    PROJECTID_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALID_FIELD_NUMBER: _ClassVar[int]
    projectId: int
    credentialId: int
    def __init__(self, projectId: _Optional[int] = ..., credentialId: _Optional[int] = ...) -> None: ...

class CreateProjectCredentialResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: ProjectCredential
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Union[ProjectCredential, _Mapping]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...

class GetAllProjectCredentialResponse(_message.Message):
    __slots__ = ("code", "success", "data", "error", "paginated")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAGINATED_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    data: _containers.RepeatedCompositeFieldContainer[ProjectCredential]
    error: _common_pb2.Error
    paginated: _common_pb2.Paginated
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., data: _Optional[_Iterable[_Union[ProjectCredential, _Mapping]]] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ..., paginated: _Optional[_Union[_common_pb2.Paginated, _Mapping]] = ...) -> None: ...

class RevokeProjectCredentialResponse(_message.Message):
    __slots__ = ("code", "success", "id", "error")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    code: int
    success: bool
    id: int
    error: _common_pb2.Error
    def __init__(self, code: _Optional[int] = ..., success: bool = ..., id: _Optional[int] = ..., error: _Optional[_Union[_common_pb2.Error, _Mapping]] = ...) -> None: ...
