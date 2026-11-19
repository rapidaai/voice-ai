"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import time
from unittest.mock import AsyncMock

import jwt
import pytest
import pytest_asyncio
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import FastAPI, Request
from starlette.authentication import AuthenticationError
from starlette.requests import HTTPConnection

from app.authentication import (
    Identity,
    ServiceAssertionIssuer,
    ServiceAssertionVerifier,
    Tenant,
)
from app.configs.auth_config import (
    AuthenticationConfig,
    UserAuthenticationConfig,
)
from app.middlewares.principal_authentication_middleware import (
    PrincipalAuthenticationBackend,
    PrincipalAuthenticationMiddleware,
)

SECRET = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="


@pytest.fixture
def test_app() -> FastAPI:
    return FastAPI()


@pytest_asyncio.fixture
async def async_test_client(test_app: FastAPI) -> AsyncTestClient:
    return AsyncTestClient(test_app)


def configs():
    authentication = AuthenticationConfig(
        user=UserAuthenticationConfig(auth_host="localhost:9001"),
    )
    issuer = ServiceAssertionIssuer(
        destination="document-api",
        secret=SECRET,
    )
    return authentication, issuer


def user_authentication():
    return {
        "principal": {
            "subject": {"type": "user", "id": 7},
            "selectedTenant": {"organizationId": 9, "projectId": 11},
            "proof": {
                "method": "user_session",
                "reference": {"type": "user_session", "value": "8"},
            },
        },
        "user": {
            "id": 7,
            "name": "Ada",
            "email": "ada@example.com",
            "status": "ACTIVE",
        },
        "organizations": [
            {
                "id": 9,
                "name": "Rapida",
                "roles": [],
                "projects": [{"id": 11, "name": "Voice", "roles": []}],
            }
        ],
    }


@pytest.mark.asyncio
async def test_service_assertion_attaches_direct_service_principal(
    test_app, async_test_client
):
    authentication, issuer = configs()
    test_app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(),
    )

    @test_app.get("/principal")
    async def principal(request: Request):
        return {
            "subject": request.user.actor,
            "organizationId": request.user.organization_id,
            "projectId": request.user.project_id,
        }

    token = issuer.issue(Tenant(organizationId=9, projectId=11))
    async_test_client.headers = {"x-internal-service-key": token}
    response = await async_test_client.get("/principal")

    assert response.status_code == 200
    assert response.json() == {
        "subject": {"type": "service", "id": 1},
        "organizationId": 9,
        "projectId": 11,
    }


@pytest.mark.asyncio
async def test_delegated_subject_owns_domain_audit_actor(
    test_app, async_test_client
):
    authentication, issuer = configs()
    test_app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(),
    )

    @test_app.get("/actor")
    async def actor(request: Request):
        return request.user.actor

    token = issuer.issue(
        Tenant(organizationId=9, projectId=11),
        subject=Identity(type="user", id=7),
        permissions=("organization.members.read", "project.knowledge.read"),
    )
    async_test_client.headers = {"x-internal-service-key": token}
    response = await async_test_client.get("/actor")

    assert response.status_code == 200
    assert response.json() == {"type": "user", "id": 7}


@pytest.mark.asyncio
async def test_user_credential_resolves_authentication_aggregate(
    test_app, async_test_client
):
    authentication, _ = configs()
    resolver = AsyncMock(return_value=user_authentication())
    test_app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=resolver,
    )

    @test_app.get("/user")
    async def user(request: Request):
        return {
            "id": request.user.user_id,
            "organizationId": request.user.organization_id,
            "projectId": request.user.project_id,
        }

    async_test_client.headers = {
        "authorization": "Bearer token",
        "x-auth-id": "7",
        "x-organization-id": "9",
        "x-project-id": "11",
    }
    response = await async_test_client.get("/user")

    assert response.status_code == 200
    assert response.json() == {"id": 7, "organizationId": 9, "projectId": 11}
    resolver.assert_awaited_once_with(
        auth_token="Bearer token",
        user_id="7",
        organization_id="9",
        project_id="11",
    )


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("headers", "resolved"),
    [
        ({"x-project-id": "11"}, user_authentication()),
        (
            {},
            {
                **user_authentication(),
                "principal": {
                    **user_authentication()["principal"],
                    "selectedTenant": {"organizationId": 9, "projectId": 0},
                },
            },
        ),
        (
            {},
            {
                **user_authentication(),
                "principal": {
                    **user_authentication()["principal"],
                    "selectedTenant": {"organizationId": 0, "projectId": 0},
                },
                "organizations": [
                    *user_authentication()["organizations"],
                    {"id": 10, "name": "Acme", "roles": [], "projects": []},
                ],
            },
        ),
    ],
)
async def test_user_credential_accepts_deterministic_tenant_resolution(
    headers, resolved
):
    authentication, _ = configs()
    backend = PrincipalAuthenticationBackend(
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(return_value=resolved),
    )
    request_headers = [
        (b"authorization", b"Bearer token"),
        (b"x-auth-id", b"7"),
    ]
    request_headers.extend((key.encode(), value.encode()) for key, value in headers.items())

    credentials, principal = await backend.authenticate(
        HTTPConnection({"type": "http", "headers": request_headers})
    )

    assert credentials.scopes == ["authenticated"]
    assert principal.is_authenticated


@pytest.mark.asyncio
async def test_user_credential_rejects_ambiguous_organization_selection():
    authentication, _ = configs()
    resolver = AsyncMock(return_value=user_authentication())
    backend = PrincipalAuthenticationBackend(
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=resolver,
    )

    for organization_headers in (
        (b" ",),
        (b"not-an-id",),
        (b"9223372036854775808",),
        (b"9", b"10"),
    ):
        headers = [
            (b"authorization", b"Bearer token"),
            (b"x-auth-id", b"7"),
        ]
        headers.extend(
            (b"x-organization-id", value) for value in organization_headers
        )
        with pytest.raises(AuthenticationError):
            await backend.authenticate(
                HTTPConnection({"type": "http", "headers": headers})
            )

    resolver.assert_not_awaited()


@pytest.mark.asyncio
async def test_organization_selection_without_user_credential_fails_closed():
    authentication, issuer = configs()
    resolver = AsyncMock(return_value=user_authentication())
    backend = PrincipalAuthenticationBackend(
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=resolver,
    )
    assertion = issuer.issue(Tenant(organizationId=9, projectId=0)).encode()
    for headers in (
        [(b"x-organization-id", b"9")],
        [
            (b"x-organization-id", b"9"),
            (b"x-internal-service-key", assertion),
        ],
    ):
        with pytest.raises(AuthenticationError):
            await backend.authenticate(
                HTTPConnection({"type": "http", "headers": headers})
            )

    resolver.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("headers", "resolved"),
    [
        (
            {"x-organization-id": "10", "x-project-id": "11"},
            user_authentication(),
        ),
        (
            {"x-organization-id": "9"},
            user_authentication(),
        ),
        (
            {},
            user_authentication(),
        ),
        (
            {},
            {
                **user_authentication(),
                "principal": {
                    **user_authentication()["principal"],
                    "subject": {"type": "user", "id": 8},
                },
                "user": {
                    **user_authentication()["user"],
                    "id": 8,
                },
            },
        ),
        (
            {"x-organization-id": "9", "x-project-id": "11"},
            {
                **user_authentication(),
                "principal": {
                    **user_authentication()["principal"],
                    "actor": {"id": 12, "issuer": "workflow-api"},
                    "proof": {
                        "method": "service_assertion",
                        "audience": "rapida_internal",
                        "reference": {"type": "signing_key", "value": "key-1"},
                    },
                },
            },
        ),
    ],
)
async def test_user_credential_rejects_resolved_principal_outside_request(
    headers, resolved
):
    authentication, _ = configs()
    backend = PrincipalAuthenticationBackend(
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(return_value=resolved),
    )
    request_headers = [
        (b"authorization", b"Bearer token"),
        (b"x-auth-id", b"7"),
    ]
    request_headers.extend((key.encode(), value.encode()) for key, value in headers.items())

    with pytest.raises(AuthenticationError):
        await backend.authenticate(
            HTTPConnection({"type": "http", "headers": request_headers})
        )


@pytest.mark.asyncio
async def test_conflicting_credentials_fail_closed(
    test_app, async_test_client
):
    authentication, issuer = configs()
    test_app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(),
    )

    @test_app.get("/conflict")
    async def conflict():
        return {"ok": True}

    async_test_client.headers = {
        "x-internal-service-key": issuer.issue(
            Tenant(organizationId=9, projectId=11)
        ),
        "authorization": "Bearer token",
        "x-auth-id": "7",
    }
    response = await async_test_client.get("/conflict")

    assert response.status_code == 401
    assert response.json()["content"]["error_message"] == "authentication rejected"


@pytest.mark.asyncio
async def test_invalid_or_duplicate_service_credentials_fail_closed():
    authentication, issuer = configs()
    backend = PrincipalAuthenticationBackend(
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(),
    )

    with pytest.raises(AuthenticationError):
        await backend.authenticate(
            HTTPConnection(
                {
                    "type": "http",
                    "headers": [(b"x-internal-service-key", b"invalid")],
                }
            )
        )

    token = issuer.issue(Tenant(organizationId=9, projectId=11)).encode()
    with pytest.raises(AuthenticationError):
        await backend.authenticate(
            HTTPConnection(
                {
                    "type": "http",
                    "headers": [
                        (b"x-internal-service-key", token),
                        (b"x-internal-service-key", token),
                    ],
                }
            )
        )


@pytest.mark.asyncio
async def test_legacy_schema_and_wrong_audience_are_rejected(
    test_app, async_test_client
):
    authentication, _ = configs()
    test_app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=authentication,
        service_assertion_verifier=ServiceAssertionVerifier("document-api", SECRET),
        user_authentication_resolver=AsyncMock(),
    )

    @test_app.get("/service")
    async def service_route():
        return {"ok": True}

    now = int(time.time())
    legacy = jwt.encode(
        {
            "iss": "workflow-api",
            "aud": "document-api",
            "sub": "service:9007",
            "tenant": {"organization_id": 9, "project_id": 11},
            "permissions": [],
            "iat": now,
            "exp": now + 60,
            "jti": "AAAAAAAAAAAAAAAAAAAAAA",
        },
        "legacy-secret",
        algorithm="HS256",
        headers={"kid": "workflow-local-1"},
    )
    async_test_client.headers = {"x-internal-service-key": legacy}
    response = await async_test_client.get("/service")
    assert response.status_code == 401

    integration_issuer = ServiceAssertionIssuer(
        destination="integration-api",
        secret=SECRET,
    )
    async_test_client.headers = {
        "x-internal-service-key": integration_issuer.issue(
            Tenant(organizationId=9, projectId=11)
        )
    }
    response = await async_test_client.get("/service")
    assert response.status_code == 401
