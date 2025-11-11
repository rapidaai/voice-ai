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
import pytest_asyncio
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import FastAPI
from fastapi.testclient import TestClient
from mock.mock import MagicMock

from app.connectors.redis_connector import RedisConnector
from app.main import app


@pytest.fixture
def api_client() -> TestClient:
    """
    Returns a fastapi.test-client.TestClient.
    The test client uses the requests' library for making http requests.
    :return: TestClient
    """
    return TestClient(app)


@pytest_asyncio.fixture
async def async_api_client() -> AsyncTestClient:
    """
    Returns an async_asgi_testclient.TestClient.
    :return: AsyncTestClient
    """
    return AsyncTestClient(app)


@pytest.fixture
def test_app() -> FastAPI:
    """
    Create test purpose FastAPi app
    :return: FastAPI
    """
    return FastAPI()


@pytest_asyncio.fixture(scope="function")
async def async_test_client(test_app: FastAPI) -> AsyncTestClient:
    """
    Returns an async_asgi_testclient.TestClient with Test App
    :param: application
    :return: AsyncTestClient
    """
    return AsyncTestClient(test_app)


@pytest_asyncio.fixture
async def redis_test_connector() -> RedisConnector:
    """
    redis connector can be used for testing
    :return:
    """
    return MagicMock()
