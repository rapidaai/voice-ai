"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import logging
from typing import Callable

from fastapi import FastAPI
from starlette.authentication import (
    AuthCredentials,
    AuthenticationBackend,
    AuthenticationError,
    UnauthenticatedUser,
)
from starlette.middleware.authentication import AuthenticationMiddleware
from starlette.requests import HTTPConnection

from app.authentication import (
    Authentication,
    AuthenticatedPrincipal,
    ServiceAssertionVerifier,
)
from app.commons.j_response import JResponse
from app.configs.auth_config import AuthenticationConfig
from app.exceptions.authentication_exception import (
    AuthenticationException,
    InvalidAuthorizationTokenException,
)
from app.exceptions.bridges_exceptions import BridgeException

_log = logging.getLogger("app.middlewares.principal_authentication_middleware")
SERVICE_ASSERTION_HEADER = "x-internal-service-key"


class PrincipalAuthenticationMiddleware(AuthenticationMiddleware):
    def __init__(
        self,
        app: FastAPI,
        authentication_config: AuthenticationConfig,
        service_assertion_verifier: ServiceAssertionVerifier,
        user_authentication_resolver: Callable,
    ):
        super().__init__(
            app=app,
            backend=PrincipalAuthenticationBackend(
                service_assertion_verifier=service_assertion_verifier,
                authentication_config=authentication_config,
                user_authentication_resolver=user_authentication_resolver,
            ),
            on_error=lambda _, error: JResponse.default_on_error(
                exc=error,
                error_message="authentication rejected",
                error_code=401,
            ),
        )


class PrincipalAuthenticationBackend(AuthenticationBackend):
    def __init__(
        self,
        service_assertion_verifier: ServiceAssertionVerifier,
        authentication_config: AuthenticationConfig,
        user_authentication_resolver: Callable,
    ):
        self.service_assertion_verifier = service_assertion_verifier
        self.authentication_config = authentication_config
        self.user_authentication_resolver = user_authentication_resolver

    async def authenticate(self, conn: HTTPConnection):
        user_config = self.authentication_config.user
        try:
            service_assertions = conn.headers.getlist(SERVICE_ASSERTION_HEADER)
            authorizations = conn.headers.getlist(user_config.authorization_header)
            user_ids = conn.headers.getlist(user_config.user_header)
            organization_ids = conn.headers.getlist(user_config.organization_header)
            project_ids = conn.headers.getlist(user_config.project_header)
            if any(
                len(values) > 1
                for values in (
                    service_assertions,
                    authorizations,
                    user_ids,
                    organization_ids,
                    project_ids,
                )
            ):
                raise InvalidAuthorizationTokenException(
                    "authentication credential header is ambiguous"
                )
            service_assertion = service_assertions[0] if service_assertions else None
            authorization = authorizations[0] if authorizations else None
            user_id = user_ids[0] if user_ids else None
            organization_id = organization_ids[0] if organization_ids else None
            project_id = project_ids[0] if project_ids else None
            user_present = (
                authorization is not None
                or user_id is not None
                or organization_id is not None
                or project_id is not None
            )
            if service_assertion is not None and user_present:
                raise InvalidAuthorizationTokenException("authentication credentials conflict")
            if service_assertion is not None:
                principal = self.service_assertion_verifier.verify(service_assertion)
                return AuthCredentials(["authenticated"]), AuthenticatedPrincipal(principal=principal)
            if user_present:
                if not authorization or not user_id:
                    raise InvalidAuthorizationTokenException("user authentication is incomplete")
                if organization_id is not None:
                    try:
                        selected_organization_id = int(organization_id)
                    except ValueError as error:
                        raise InvalidAuthorizationTokenException(
                            "organization selection is invalid"
                        ) from error
                    if (
                        not organization_id
                        or organization_id.strip() != organization_id
                        or selected_organization_id < 1
                        or selected_organization_id > 9223372036854775807
                    ):
                        raise InvalidAuthorizationTokenException(
                            "organization selection is invalid"
                        )
                response = await self.user_authentication_resolver(
                    auth_token=authorization,
                    user_id=user_id,
                    organization_id=organization_id,
                    project_id=project_id,
                )
                authentication = Authentication.model_validate(response)
                tenant = authentication.principal.selected_tenant
                organizations = authentication.organizations
                subject = authentication.principal.subject
                proof = authentication.principal.proof
                if (
                    subject.type != "user"
                    or str(subject.id) != user_id
                    or authentication.principal.actor is not None
                    or proof.method != "user_session"
                    or proof.audience
                    or proof.reference.type != "user_session"
                ):
                    raise InvalidAuthorizationTokenException(
                        "resolved authentication does not match the request"
                    )
                if organization_id is not None:
                    if tenant.organization_id != selected_organization_id:
                        raise InvalidAuthorizationTokenException(
                            "resolved authentication does not match the request"
                        )
                    if project_id is None and tenant.project_id:
                        raise InvalidAuthorizationTokenException(
                            "resolved authentication does not match the request"
                        )
                if project_id is not None and str(tenant.project_id) != project_id:
                    raise InvalidAuthorizationTokenException(
                        "resolved authentication does not match the request"
                    )
                if organization_id is None and project_id is None:
                    if tenant.project_id:
                        raise InvalidAuthorizationTokenException(
                            "resolved authentication does not match the request"
                        )
                    if len(organizations) == 1:
                        if tenant.organization_id != organizations[0].id:
                            raise InvalidAuthorizationTokenException(
                                "resolved authentication does not match the request"
                            )
                    elif tenant.organization_id:
                        raise InvalidAuthorizationTokenException(
                            "resolved authentication does not match the request"
                        )
                return AuthCredentials(["authenticated"]), AuthenticatedPrincipal(authentication=authentication)
            return AuthCredentials(), UnauthenticatedUser()
        except (
            AuthenticationException,
            InvalidAuthorizationTokenException,
            AuthenticationError,
            BridgeException,
            ValueError,
        ) as error:
            _log.debug("Authentication rejected: %s", error)
            raise AuthenticationError("authentication rejected") from error
