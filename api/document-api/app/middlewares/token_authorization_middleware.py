"""
Copyright (c) 2024 Prashant Srivastav <prashant@rapida.ai>
All rights reserved.

This code is licensed under the MIT License. You may obtain a copy of the License at
https://opensource.org/licenses/MIT.

Unless required by applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions
and limitations under the License.

"""

import logging
from typing import Callable, Optional, Tuple

from fastapi import FastAPI
from starlette.authentication import AuthenticationBackend
from starlette.middleware.authentication import AuthenticationMiddleware
from starlette.requests import HTTPConnection

from app.exceptions import RapidaException
from app.exceptions.authentication_exception import (
    AuthenticationException,
    InvalidAuthorizationTokenException,
    MissingAuthorizationKeyException,
)
from app.exceptions.bridges_exceptions import BridgeException
from app.middlewares.auth.user import AnonymousUser, AuthenticatedUser, User

_log = logging.getLogger("app.middlewares.token_authorization_middleware")


class TokenAuthorizationMiddleware(AuthenticationMiddleware):
    """
    Token Authorization Middleware
    auth-token based authentication for requests
    """

    def __init__(self, app: FastAPI, user_info_resolver: Callable):
        super().__init__(
            backend=TokenAuthBackend(user_info_resolver=user_info_resolver),
            app=app,
        )


class TokenAuthBackend(AuthenticationBackend):
    """
    Authorize user for request
    """

    user_info_resolver: Callable
    authorization_header_key = "authorization"
    auth_header_key = "x-auth-id"
    project_header_key = "x-project-id"

    def __init__(self, user_info_resolver):
        self.user_info_resolver = user_info_resolver

    async def authenticate(self, conn: HTTPConnection) -> Tuple[bool, Optional[User]]:
        """
        Authenticate user from given token
        All the authentication exceptions will be handled with flag strict
        if strict is True, then it will raise exception if any failure occurs while authenticating
        if not then it will handle gracefully and return False and Unknown user object
        :type conn: HTTPConnection
        :param conn:
        :return: User object and is_authenticated or not (True/False)
        """
        try:
            authorization_token: str = conn.headers.get(self.authorization_header_key)
            auth_id: str = conn.headers.get(self.auth_header_key)
            if not authorization_token or not auth_id:
                raise MissingAuthorizationKeyException(auth_type="token-auth")
            try:
                user_info = await self.user_info_resolver(
                    auth_token=authorization_token, user_id=auth_id
                )
                if not user_info or "user" not in user_info:
                    raise InvalidAuthorizationTokenException("illegal token payload.")
                project_id: str = conn.headers.get(self.project_header_key)

                _log.debug(f"got the user {user_info}")
                user = AuthenticatedUser(**user_info)
                if not project_id:
                    return True, user

                user.select_project(project_id)
                return True, user
            except BridgeException as err:
                _log.debug(f"Authentication Exception while resolving user-info: {err}")
                raise InvalidAuthorizationTokenException(
                    f"Token request is not valid. {err}"
                )
        except AuthenticationException as ex:
            _log.debug(f"Authentication Exception while authorizing: {ex}")
            # raise ex
            # if not strict then return false and unknown user object
            return False, AnonymousUser()
        except RapidaException as ex:
            _log.debug(f"Authentication Exception while authorizing: {ex}")
            # if self.config.strict:
            #     raise ex
            return False, AnonymousUser()
