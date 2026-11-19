"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

from pydantic import BaseModel, ConfigDict, field_validator


class UserAuthenticationConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    auth_host: str
    authorization_header: str = "authorization"
    user_header: str = "x-auth-id"
    organization_header: str = "x-organization-id"
    project_header: str = "x-project-id"

    @field_validator(
        "auth_host",
        "authorization_header",
        "user_header",
        "organization_header",
        "project_header",
    )
    @classmethod
    def validate_required_string(cls, value):
        if not value or value.strip() != value or "\x00" in value:
            raise ValueError("authentication configuration contains an invalid string")
        return value


class AuthenticationConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    user: UserAuthenticationConfig
