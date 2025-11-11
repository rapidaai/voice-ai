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
import jwt
import pytest
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import Request

from app.configs.auth_config import JwtConfig
from app.exceptions.authentication_exception import (
    InvalidAuthorizationTokenException,
    MissingAuthorizationKeyException,
)
from app.middlewares import JwtAuthorizationMiddleware
from app.middlewares.auth.user import AnonymousUser, AuthenticatedUser


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
        # with pytest.raises(InvalidAuthorizationTokenException):
        valid_token = jwt.encode(
            {"user_id": 100},
            self.config.secret_key.get_secret_value(),
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )

        async_test_client.headers = {"authorization": valid_token}

        response = await async_test_client.get("/test/valid-jwt-token-request")
        assert response.status_code == 200

        j_response = response.json()
        assert j_response == {"test": "ok"}


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
            assert isinstance(request.user, AuthenticatedUser)
            user: AuthenticatedUser = request.user
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
        # with pytest.raises(InvalidAuthorizationTokenException):
        valid_token = jwt.encode(
            {"user_id": 100},
            self.config.secret_key.get_secret_value(),
            headers={"alg": list(self.config.algorithms)[0], "typ": "jwt"},
            algorithm=list(self.config.algorithms)[0],
        )

        async_test_client.headers = {"authorization": valid_token}
        response = await async_test_client.get("/test/valid-jwt-token-request")
        assert response.status_code == 200
        j_response = response.json()
        assert j_response == {"test": "ok"}
