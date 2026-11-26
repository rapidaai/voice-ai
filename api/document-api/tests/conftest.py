"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import pytest
import pytest_asyncio
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import FastAPI
from fastapi.testclient import TestClient
from mock.mock import MagicMock

from app.connectors.redis_connector import RedisConnector  # noqa: E402
from app.main import app  # noqa: E402


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
