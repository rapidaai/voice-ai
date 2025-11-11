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
import pytest
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import Request

from app.configs.auth_config import TokenConfig
from app.exceptions.authentication_exception import (
    InvalidAuthorizationTokenException,
    MissingAuthorizationKeyException,
)
from app.middlewares import TokenAuthorizationMiddleware
from app.middlewares.auth.user import AuthenticatedUser


class TestStrictTokenAuthorizationMiddleware:
    config: TokenConfig = TokenConfig(strict=True, enable=True)

    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        async def user_info(token: str):
            assert token is not None
            if token == "valid-token":
                return {"user_id": 1, "is_staff": True, "email": ""}
            else:
                return {}

        test_app.add_middleware(
            TokenAuthorizationMiddleware,
            config=self.config,
            user_info_resolver=user_info,
        )

        @pytest.mark.asyncio
        @test_app.get("/test/invalid-token-request")
        async def invalid_token_request(request: Request):
            return {"test": "ok"}

        @pytest.mark.asyncio
        @test_app.get("/test/valid-token-request")
        async def valid_token_request(request: Request):
            assert bool(request.auth)
            assert request.user is not None
            assert isinstance(request.user, AuthenticatedUser)
            user: AuthenticatedUser = request.user
            assert user.user_id is not None
            return {"test": "ok"}

    @pytest.mark.asyncio
    async def test_without_authorization_header(
        self, async_test_client: AsyncTestClient
    ):
        with pytest.raises(MissingAuthorizationKeyException):
            await async_test_client.get("/test/invalid-token-request")

    @pytest.mark.asyncio
    async def test_with_invalid_authorization_header(
        self, async_test_client: AsyncTestClient
    ):
        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"Authorization": "my-token"}
            await async_test_client.get("/test/invalid-token-request")

        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"Authorization": "Token invalid-token"}
            await async_test_client.get("/test/invalid-token-request")

        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": "token invalid-token"}
            await async_test_client.get("/test/invalid-token-request")

    @pytest.mark.asyncio
    async def test_with_valid_authentication(self, async_test_client: AsyncTestClient):
        async_test_client.headers = {"Authorization": "Token valid-token"}
        response = await async_test_client.get("/test/valid-token-request")
        assert response.status_code == 200
        j_response = response.json()
        assert j_response == {"test": "ok"}

    @pytest.mark.asyncio
    async def test_with_valid_authentication_small_case(
        self, async_test_client: AsyncTestClient
    ):
        async_test_client.headers = {"authorization": "token valid-token"}
        response = await async_test_client.get("/test/valid-token-request")
        assert response.status_code == 200
        j_response = response.json()
        assert j_response == {"test": "ok"}


class TestLooseTokenAuthorizationMiddleware:
    config: TokenConfig = TokenConfig(strict=False, enable=True)

    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        async def user_info(token: str):
            assert token is not None
            if token == "valid-token":
                return {"user_id": 1, "is_staff": True, "email": ""}
            else:
                return {}

        test_app.add_middleware(
            TokenAuthorizationMiddleware,
            config=self.config,
            user_info_resolver=user_info,
        )

        @pytest.mark.asyncio
        @test_app.get("/test/invalid-token-request")
        async def invalid_token_request(request: Request):
            return {"test": "ok"}

        @pytest.mark.asyncio
        @test_app.get("/test/valid-token-request")
        async def valid_token_request(request: Request):
            assert bool(request.auth)
            assert request.user is not None
            assert isinstance(request.user, AuthenticatedUser)
            user: AuthenticatedUser = request.user
            assert user.user_id is not None
            return {"test": "ok"}

    @pytest.mark.asyncio
    async def test_with_invalid_tokens(self, async_test_client: AsyncTestClient):
        m_response = await async_test_client.get("/test/invalid-token-request")
        assert m_response.status_code == 200
        assert m_response.json() == {"test": "ok"}

        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": "token my_token"}
            await async_test_client.get("/test/invalid-token-request")

        # invalid secret
        async_test_client.headers = {"authorization": ""}
        await async_test_client.get("/test/invalid-token-request")
        # invalid payload
        with pytest.raises(InvalidAuthorizationTokenException):
            async_test_client.headers = {"authorization": "xyz.abc.123"}
            await async_test_client.get("/test/invalid-token-request")

    # positive test case
    @pytest.mark.asyncio
    async def test_with_valid_authentication(self, async_test_client: AsyncTestClient):
        # with pytest.raises(InvalidAuthorizationTokenException):
        async_test_client.headers = {"authorization": "token valid-token"}
        response = await async_test_client.get("/test/valid-token-request")
        assert response.status_code == 200
        j_response = response.json()
        assert j_response == {"test": "ok"}
