"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

from abc import ABC, abstractmethod
from typing import List, Optional, Union

from pydantic import BaseModel, Field

from app.exceptions.authentication_exception import InvalidAuthorizationTokenException


class Account(BaseModel):
    id: int
    name: str
    email: str


class Token(BaseModel):
    id: int
    token: str
    tokenType: str


class OrganizationRole(BaseModel):
    id: int
    organizationId: int
    role: str
    organizationName: str


class ProjectRole(BaseModel):
    id: int
    projectId: int
    role: str
    projectName: str


class User(ABC, BaseModel):

    @abstractmethod
    def user_id(self):
        raise NotImplementedError("illegal authenticated user")

    @abstractmethod
    def project_id(self):
        raise NotImplementedError("illegal authenticated user")

    @abstractmethod
    def organization_id(self):
        raise NotImplementedError("illegal authenticated user")

    @property
    @abstractmethod
    def actor(self) -> dict:
        raise NotImplementedError("illegal authenticated user")


class AuthenticatedUser(User):
    user: Account
    token: Token
    organizationRole: OrganizationRole
    projectRoles: List[ProjectRole]
    currentProject: Optional[ProjectRole] = Field(None)
    actorType: Optional[str] = None
    actorId: Optional[int] = None

    def select_project(self, project_id: str) -> Optional[ProjectRole]:
        for project in self.projectRoles:
            if project.projectId == int(project_id):
                self.currentProject = project
                return project
        return None

    @property
    def user_id(self) -> int:
        return self.user.id

    @property
    def project_id(self) -> Union[int, None]:
        if not self.currentProject:
            return None
        return self.currentProject.projectId

    @property
    def organization_id(self) -> int:
        return self.organizationRole.organizationId

    @property
    def actor(self) -> dict:
        actor_type = self.actorType or "user"
        actor_id = self.actorId or self.user.id
        return _validated_actor(actor_type, actor_id)


class InternalAuthenticatedUser(User):
    userId: Optional[int] = None
    projectId: Optional[int] = None
    organizationId: int
    actorType: Optional[str] = Field(default=None, alias="actor_type")
    actorId: Optional[int] = Field(default=None, alias="actor_id")

    @property
    def user_id(self) -> int:
        if self.userId is None:
            raise InvalidAuthorizationTokenException("service authentication has no user identity")
        return self.userId

    @property
    def project_id(self) -> Union[int, None]:
        return self.projectId

    @property
    def organization_id(self) -> int:
        return self.organizationId

    @property
    def actor(self) -> dict:
        if self.actorType is not None and self.actorId is not None:
            return _validated_actor(self.actorType, self.actorId)
        if self.userId is not None:
            return _validated_actor("user", self.userId)
        raise InvalidAuthorizationTokenException("authenticated request is missing actor identity")


class AnonymousUser(User):
    @property
    def user_id(self) -> int:
        raise InvalidAuthorizationTokenException(
            "anonymous user doen't have any attribute."
        )

    @property
    def project_id(self) -> Union[int, None]:
        raise InvalidAuthorizationTokenException(
            "anonymous user doen't have any attribute."
        )

    @property
    def organization_id(self) -> int:
        raise InvalidAuthorizationTokenException(
            "anonymous user doen't have any attribute."
        )

    @property
    def actor(self) -> dict:
        raise InvalidAuthorizationTokenException(
            "anonymous user doesn't have an actor identity."
        )


def _validated_actor(actor_type: str, actor_id: int) -> dict:
    if actor_type not in {"user", "project", "organization", "service", "system"}:
        raise InvalidAuthorizationTokenException("unsupported actor type")
    if actor_id <= 0 or actor_id > 9223372036854775807:
        raise InvalidAuthorizationTokenException("actor id must be a positive bigint")
    return {"type": actor_type, "id": actor_id}
