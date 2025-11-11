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
from typing import Dict

import pytest
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import Request

from app.middlewares import RequestEnhancerMiddleware


class TestRequestEnhancerMiddleware:
    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        test_app.add_middleware(RequestEnhancerMiddleware)

        @pytest.mark.asyncio
        @test_app.get("/test/with-request-enhancer-all-header-middleware")
        async def with_all_header_set(request: Request):
            assert request.state.platform["device_id"] is not None
            assert request.state.platform["device_id"] == "test_user_id"

            assert request.state.platform["country_code"] is not None
            assert request.state.platform["country_code"] == "us"

            assert request.state.platform["country_locale"] is not None
            assert request.state.platform["country_locale"] == "us"

            assert request.state.platform["ip"] is not None
            assert request.state.platform["ip"] == "127.1.1"

            assert request.state.platform["accept_language"] is not None
            assert request.state.platform["accept_language"] == "en-us"

            assert request.state.platform["is_mobile"] is not None
            assert request.state.platform["is_mobile"] is True

            assert request.state.platform["is_lomotif_client"] is not None
            assert request.state.platform["is_lomotif_client"] is True

            assert request.state.platform["tier"] is not None
            assert request.state.platform["tier"] == "client"

            assert request.state.platform["product"] is not None
            assert request.state.platform["product"] == "ios"

            assert request.state.platform["version"] is not None
            assert request.state.platform["version"] == "5.4.30"

            assert request.state.platform["os"] is not None
            assert request.state.platform["os"] == "10.0.1"

            return {"test": "ok"}

        @pytest.mark.asyncio
        @test_app.get("/test/with-request-enhancer-combination-header-middleware")
        async def with_request_combination(request: Request):
            return {"test": "ok", "platform": request.state.platform}

    @pytest.mark.asyncio
    async def test_set_all_param(self, async_test_client: AsyncTestClient):
        async_test_client.headers = {
            "HTTP_X_USER_ID": "test_user_id",
            "HTTP_ACCEPT_LANGUAGE": "en-US",
            "HTTP_X_COUNTRY_CODE": "us",
            "HTTP_CF_CONNECTING_IP": "127.1.1",
            "HTTP_CF_IPCOUNTRY": "us",
            "HTTP_X_Lomotif_Agent": "Client/ios/5.4.30/10.0.1",
        }
        response = await async_test_client.get(
            "/test/with-request-enhancer-all-header-middleware"
        )
        assert response.status_code == 200
        assert response.json() == {"test": "ok"}

    @pytest.mark.asyncio
    @pytest.mark.parametrize(
        "platform_headers, expected_platform_state",
        [
            (
                {
                    "HTTP_X_USER_ID": "test_user_id",
                    "HTTP_ACCEPT_LANGUAGE": "en-US",
                    "HTTP_X_COUNTRY_CODE": "us",
                    "HTTP_CF_CONNECTING_IP": "127.1.1",
                    "HTTP_CF_IPCOUNTRY": "us",
                    "HTTP_X_Lomotif_Agent": "Client/ios/5.4.30/10.0.1",
                },
                {
                    "accept_language": "en-us",
                    "country_code": "us",
                    "country_locale": "us",
                    "device_id": "test_user_id",
                    "ip": "127.1.1",
                    "is_lomotif_client": True,
                    "is_mobile": True,
                    "os": "10.0.1",
                    "product": "ios",
                    "tier": "client",
                    "version": "5.4.30",
                },
            ),
            (
                {
                    "HTTP_X_USER_ID": "test_user_id",
                    "HTTP_ACCEPT_LANGUAGE": "en-US",
                    "HTTP_X_COUNTRY_CODE": "us",
                    "HTTP_CF_CONNECTING_IP": "127.1.1",
                    "HTTP_CF_IPCOUNTRY": "us",
                },
                {
                    "accept_language": "en-us",
                    "country_code": "us",
                    "country_locale": "us",
                    "device_id": "test_user_id",
                    "ip": "127.1.1",
                    "is_lomotif_client": False,
                    "is_mobile": False,
                    "os": "",
                    "product": "unknown",
                    "tier": None,
                    "version": "",
                },
            ),
            (
                {
                    "HTTP_X_USER_ID": "test_user_id",
                    "HTTP_ACCEPT_LANGUAGE": "en-US",
                    "HTTP_X_COUNTRY_CODE": "us",
                    "HTTP_CF_CONNECTING_IP": "127.1.1",
                    "HTTP_CF_IPCOUNTRY": "us",
                    "HTTP_X_Lomotif_Agent": "Client/android/5.4.30/10.0.1",
                },
                {
                    "accept_language": "en-us",
                    "country_code": "us",
                    "country_locale": "us",
                    "device_id": "test_user_id",
                    "ip": "127.1.1",
                    "is_lomotif_client": True,
                    "is_mobile": True,
                    "os": "10.0.1",
                    "product": "android",
                    "tier": "client",
                    "version": "5.4.30",
                },
            ),
            (
                {
                    "HTTP_x_user_id": "test_user_id",
                    "HTTP_accept_language": "en-us",
                    "HTTP_X_COUNTRY_CODE": "us",
                    "HTTP_CF_CONNECTING_IP": "127.1.1",
                    "HTTP_CF_IPCountry": "us",
                    "http_x_lomotif_agent": "Client/android/5.4.30/10.0.1",
                },
                {
                    "accept_language": "en-us",
                    "country_code": "us",
                    "country_locale": "us",
                    "device_id": "test_user_id",
                    "ip": "127.1.1",
                    "is_lomotif_client": True,
                    "is_mobile": True,
                    "os": "10.0.1",
                    "product": "android",
                    "tier": "client",
                    "version": "5.4.30",
                },
            ),
            (
                {},
                {
                    "accept_language": "",
                    "country_code": "",
                    "country_locale": None,
                    "device_id": None,
                    "ip": None,
                    "is_lomotif_client": False,
                    "is_mobile": False,
                    "os": "",
                    "product": "unknown",
                    "tier": None,
                    "version": "",
                },
            ),
        ],
    )
    async def test_combination_param(
        self,
        async_test_client: AsyncTestClient,
        platform_headers: Dict,
        expected_platform_state: Dict,
    ):
        async_test_client.headers = platform_headers
        response = await async_test_client.get(
            "/test/with-request-enhancer-combination-header-middleware"
        )
        assert response.status_code == 200
        assert response.json() == {"test": "ok", "platform": expected_platform_state}
