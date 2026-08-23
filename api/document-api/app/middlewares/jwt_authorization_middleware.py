"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import logging
import time
from typing import Dict, Optional, Tuple

import jwt
from fastapi import FastAPI
from starlette.authentication import AuthenticationBackend
from starlette.middleware.authentication import AuthenticationMiddleware
from starlette.requests import HTTPConnection

from app.configs.auth_config import JwtAuthenticationConfig
from app.exceptions.authentication_exception import (
    AuthenticationException,
    InvalidAuthorizationTokenException,
    MissingAuthorizationKeyException,
)
from app.middlewares.auth.user import (
    AnonymousUser,
    InternalAuthenticatedUser,
    User,
)

_log = logging.getLogger("app.middlewares.jwt_authorization_middleware")


class JwtAuthorizationMiddleware(AuthenticationMiddleware):
    """
    Authorize user for request using jwt token
    """

    def __init__(self, app: FastAPI, config: JwtAuthenticationConfig):
        super().__init__(backend=JwtAuthBackend(config=config), app=app)


class JwtAuthBackend(AuthenticationBackend):
    """
    starlette custom authentication backend to authenticate user using jwt.
    """

    def __init__(self, config: JwtAuthenticationConfig):
        self.config = config

    async def authenticate(self, conn: HTTPConnection) -> Tuple[bool, Optional[User]]:
        """
        Authenticate user from given jwt token
        :param conn:
        :return:
        """
        try:
            authorization: str = conn.headers.get(self.config.header_key)
            if not authorization:
                raise MissingAuthorizationKeyException(auth_type="JWT")
            unverified_payload: Dict = jwt.decode(
                authorization,
                options={"verify_signature": False, "verify_exp": False, "verify_aud": False},
            )
            actor_type = unverified_payload.get("actor_type")
            if actor_type == "service":
                payload = jwt.decode(
                    authorization,
                    self.config.secret_key.get_secret_value(),
                    algorithms=["HS256"],
                    audience="rapida-internal",
                    options={"require": ["exp", "iat", "aud", "iss"]},
                )
                actor_id = payload.get("actor_id")
                issued_at = payload.get("iat")
                expires_at = payload.get("exp")
                organization_id = payload.get("organizationId")
                project_id = payload.get("projectId")
                if (
                    payload.get("actor_type") != "service"
                    or not payload.get("iss")
                    or "userId" in payload
                    or type(actor_id) is not int
                    or actor_id <= 0
                    or actor_id > 9223372036854775807
                    or type(organization_id) is not int
                    or organization_id <= 0
                    or organization_id > 9223372036854775807
                    or (
                        project_id is not None
                        and (
                            type(project_id) is not int
                            or project_id <= 0
                            or project_id > 9223372036854775807
                        )
                    )
                    or isinstance(issued_at, bool)
                    or not isinstance(issued_at, (int, float))
                    or isinstance(expires_at, bool)
                    or not isinstance(expires_at, (int, float))
                    or expires_at <= issued_at
                    or expires_at - issued_at > 300
                    or issued_at > time.time()
                ):
                    raise InvalidAuthorizationTokenException("invalid service identity")
            else:
                payload = jwt.decode(
                    authorization,
                    self.config.secret_key.get_secret_value(),
                    algorithms=self.config.algorithms,
                    options={"verify_aud": False},
                )
                actor_type = payload.get("actor_type")
                user_id = payload.get("userId")
                actor_id = payload.get("actor_id")
                if (
                    type(user_id) is not int
                    or user_id <= 0
                    or user_id > 9223372036854775807
                    or actor_type not in {None, "user"}
                    or (
                        actor_type == "user"
                        and (
                            type(actor_id) is not int
                            or actor_id != user_id
                        )
                    )
                ):
                    raise InvalidAuthorizationTokenException("invalid token payload.")
            return True, InternalAuthenticatedUser.parse_obj(payload)
        except jwt.PyJWTError as err:
            _log.debug(f"Authentication Exception while decoding token: {err}")
            raise InvalidAuthorizationTokenException(
                f"unable to decode given token. {err}"
            )
        except AuthenticationException as ex:
            _log.debug(f"Authentication Exception while authorizing: {ex}")
            if self.config.strict:
                raise ex
            return False, AnonymousUser()
