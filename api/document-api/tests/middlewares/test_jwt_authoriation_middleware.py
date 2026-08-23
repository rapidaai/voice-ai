"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""
import jwt
import pytest
import time

try:
    from app.middlewares import JwtAuthorizationMiddleware  # noqa: F401
except (ImportError, AttributeError):
    pytest.skip("JwtAuthorizationMiddleware not exported from app.middlewares", allow_module_level=True)
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import Request

from app.configs.auth_config import JwtConfig
from app.exceptions.authentication_exception import (
    InvalidAuthorizationTokenException,
    MissingAuthorizationKeyException,
)
from app.middlewares import JwtAuthorizationMiddleware
from app.middlewares.auth.user import AnonymousUser, AuthenticatedUser, InternalAuthenticatedUser


class TestStrictJWTAuthorizationMiddleware:
    config: JwtConfig = JwtConfig(secret_key="secret", strict=True, enable=True)

    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        test_app.add_middleware(JwtAuthorizationMiddleware, config=self.config)

        @pytest.mark.asyncio
        @test_app.get("/test/invalid-jwt-token-request")
        async def invalid_jwt_request():
            return {"test": "ok"}

        @pytest.mark.asyncio
        @test_app.get("/test/valid-jwt-token-request")
        async def valid_jwt_request(request: Request):
            assert bool(request.auth)
            assert request.user is not None
            user: AuthenticatedUser = request.user
            assert user.user_id is not None
            return {"test": "ok"}

    @pytest.mark.asyncio
    async def test_without_authorization_header(
        self, async_test_client: AsyncTestClient
    ):
        with pytest.raises(MissingAuthorizationKeyException):
            await async_test_client.get("/test/invalid-jwt-token-request")

    @pytest.mark.asyncio
    async def test_with_invalid_jwt_token(self, async_test_client: AsyncTestClient):
        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": "token my_token"}
            await async_test_client.get("/test/invalid-jwt-token-request")

        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {
                "authorization": "some-token-you-should-not-use"
            }
            await async_test_client.get("/test/invalid-jwt-token-request")

    @pytest.mark.asyncio
    async def test_with_invalid_secret(self, async_test_client: AsyncTestClient):
        with pytest.raises(InvalidAuthorizationTokenException):
            illegal_token = jwt.encode(
                {"user_id": 102},
                "some-secret-key-you-should-not-use",
                headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
                algorithm=list(self.config.algorithms)[0],
            )

            async_test_client.headers = {"authorization": illegal_token}

            await async_test_client.get("/test/invalid-jwt-token-request")

    @pytest.mark.asyncio
    async def test_with_invalid_payload(self, async_test_client: AsyncTestClient):
        with pytest.raises(InvalidAuthorizationTokenException):
            illegal_token = jwt.encode(
                {"some-key": "some-value"},
                self.config.secret_key.get_secret_value(),
                headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
                algorithm=list(self.config.algorithms)[0],
            )

            async_test_client.headers = {"authorization": illegal_token}

            await async_test_client.get("/test/invalid-jwt-token-request")

    # positive test case
    @pytest.mark.asyncio
    async def test_with_valid_authentication(self, async_test_client: AsyncTestClient):
        # Middleware checks payload.get("userId") and builds InternalAuthenticatedUser
        valid_token = jwt.encode(
            {"userId": 100, "projectId": 1, "organizationId": 1},
            self.config.secret_key.get_secret_value(),
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )

        async_test_client.headers = {"authorization": valid_token}

        response = await async_test_client.get("/test/valid-jwt-token-request")
        assert response.status_code == 200

        j_response = response.json()
        assert j_response == {"test": "ok"}


@pytest.mark.asyncio
async def test_service_assertion_uses_shared_secret(test_app, async_test_client):
    config = JwtConfig(secret_key="secret", strict=True, enable=True)
    test_app.add_middleware(JwtAuthorizationMiddleware, config=config)

    @test_app.get("/service")
    async def service_request(request: Request):
        return request.user.actor

    now = int(time.time())
    token = jwt.encode(
        {
            "actor_type": "service",
            "actor_id": 41,
            "iss": "assistant-api",
            "aud": "rapida-internal",
            "iat": now,
            "exp": now + 60,
            "organizationId": 7,
            "delegated_auth_type": "user",
            "delegated_actor_id": 99,
        },
        config.secret_key.get_secret_value(),
        algorithm="HS256",
    )
    async_test_client.headers = {"authorization": token}
    response = await async_test_client.get("/service")
    assert response.status_code == 200
    assert response.json() == {"type": "service", "id": 41}

    rejected = jwt.encode(
        {
            "actor_type": "service",
            "actor_id": 41,
            "iss": "assistant-api",
            "aud": "rapida-internal",
            "iat": now,
            "exp": now + 60,
            "organizationId": 7,
        },
        "wrong-secret",
        algorithm="HS256",
    )
    async_test_client.headers = {"authorization": rejected}
    with pytest.raises(InvalidAuthorizationTokenException):
        await async_test_client.get("/service")

    long_lived_token = jwt.encode(
        {
            "actor_type": "service",
            "actor_id": 41,
            "iss": "assistant-api",
            "aud": "rapida-internal",
            "iat": now,
            "exp": now + 301,
            "organizationId": 7,
        },
        config.secret_key.get_secret_value(),
        algorithm="HS256",
    )
    async_test_client.headers = {"authorization": long_lived_token}
    with pytest.raises(InvalidAuthorizationTokenException):
        await async_test_client.get("/service")

    forwarded_user_token = jwt.encode(
        {
            "actor_type": "service",
            "actor_id": 41,
            "iss": "assistant-api",
            "aud": "rapida-internal",
            "iat": now,
            "exp": now + 60,
            "organizationId": 7,
            "userId": 9,
        },
        config.secret_key.get_secret_value(),
        algorithm="HS256",
    )
    async_test_client.headers = {"authorization": forwarded_user_token}
    with pytest.raises(InvalidAuthorizationTokenException):
        await async_test_client.get("/service")


@pytest.mark.asyncio
async def test_non_service_actor_cannot_bypass_service_validation(test_app, async_test_client):
    config = JwtConfig(secret_key="secret", strict=True, enable=True)
    test_app.add_middleware(JwtAuthorizationMiddleware, config=config)

    @test_app.get("/actor")
    async def actor_request(request: Request):
        return request.user.actor

    for actor_type in ("project", "organization", "system"):
        token = jwt.encode(
            {
                "actor_type": actor_type,
                "actor_id": 41,
                "organizationId": 7,
            },
            config.secret_key.get_secret_value(),
            algorithm="HS256",
        )
        async_test_client.headers = {"authorization": token}
        with pytest.raises(InvalidAuthorizationTokenException):
            await async_test_client.get("/actor")

class TestLooseJWTAuthorizationMiddleware:
    config: JwtConfig = JwtConfig(secret_key="secret", strict=False, enable=True)

    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        test_app.add_middleware(JwtAuthorizationMiddleware, config=self.config)

        @pytest.mark.asyncio
        @test_app.get("/test/invalid-jwt-token-request")
        async def invalid_jwt_request(request: Request):
            assert request.user is not None
            assert isinstance(request.user, AnonymousUser)
            return {"test": "ok"}

        @pytest.mark.asyncio
        @test_app.get("/test/valid-jwt-token-request")
        async def valid_jwt_request(request: Request):
            assert bool(request.auth)
            assert request.user is not None
            # Middleware returns InternalAuthenticatedUser (not AuthenticatedUser)
            assert isinstance(request.user, InternalAuthenticatedUser)
            user: InternalAuthenticatedUser = request.user
            assert user.user_id is not None
            return {"test": "ok"}

    @pytest.mark.asyncio
    async def test_with_invalid_jwt_tokens(self, async_test_client: AsyncTestClient):
        m_response = await async_test_client.get("/test/invalid-jwt-token-request")
        assert m_response.status_code == 200
        assert m_response.json() == {"test": "ok"}

        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": "token my_token"}
            await async_test_client.get("/test/invalid-jwt-token-request")

        # invalid secret
        illegal_token = jwt.encode(
            {"user_id": 102},
            "some-secret-key-you-should-not-use",
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )
        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": illegal_token}
            await async_test_client.get("/test/invalid-jwt-token-request")
        # invalid payload
        illegal_token = jwt.encode(
            {"some-key": "some-value"},
            self.config.secret_key.get_secret_value(),
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )
        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": illegal_token}
            await async_test_client.get("/test/invalid-jwt-token-request")

    # positive test case
    @pytest.mark.asyncio
    async def test_with_valid_authentication(self, async_test_client: AsyncTestClient):
        # Middleware checks payload.get("userId") and builds InternalAuthenticatedUser
        valid_token = jwt.encode(
            {"userId": 100, "projectId": 1, "organizationId": 1},
            self.config.secret_key.get_secret_value(),
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )

        async_test_client.headers = {"authorization": valid_token}
        response = await async_test_client.get("/test/valid-jwt-token-request")
        assert response.status_code == 200
        j_response = response.json()
        assert j_response == {"test": "ok"}
