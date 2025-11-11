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
from typing import List

import pytest
from async_asgi_testclient import TestClient as AsyncTestClient
from fastapi import Request
from pydantic import BaseModel

from app.commons.j_response import JResponse
from app.commons.paginated_model import PaginatedModel
from app.utils.request import get_url_from_request


class TestPaginatedModel:
    @pytest.mark.asyncio
    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        class TestModel(BaseModel):
            key1: int
            key2: str
            key3: bool

        @pytest.mark.asyncio
        @test_app.get(
            "/test/simple-test-paginated_model-j-response",
            response_class=JResponse,
            response_model=PaginatedModel[TestModel],
        )
        async def simple_test_paginated_model_j_response(
            request: Request, query: str, offset: int, limit: int
        ):
            result_data: List = [
                {"key1": 1, "key2": "some-test-str", "key3": True},
                {"key1": 2, "key2": "some-test-str", "key3": True},
                {"key1": 3, "key2": "some-test-str", "key3": True},
                {"key1": 4, "key2": "some-test-str", "key3": True},
                {"key1": 5, "key2": "some-test-str", "key3": True},
                {"key1": 6, "key2": "some-test-str", "key3": True},
                {"key1": 7, "key2": "some-test-str", "key3": True},
                {"key1": 8, "key2": "some-test-str", "key3": True},
                {"key1": 9, "key2": "some-test-str", "key3": True},
                {"key1": 10, "key2": "some-test-str", "key3": True},
            ]
            return PaginatedModel(
                results=result_data[offset : offset + limit],
                param={"query": query, "offset": offset, "limit": limit},
                count=len(result_data),
                paginated_url=get_url_from_request(request),
            )

    # @pytest.mark.asyncio
    # async def test_pagination_model(test_app: FastAPI, async_test_client: AsyncTestClient):

    @pytest.mark.parametrize(
        "query, offset, limit, expected_result, expected_status",
        [
            (
                "q",
                0,
                2,
                [
                    {"key1": 1, "key2": "some-test-str", "key3": True},
                    {"key1": 2, "key2": "some-test-str", "key3": True},
                ],
                200,
            ),
            ("q", 2, 1, [{"key1": 3, "key2": "some-test-str", "key3": True}], 200),
            ("q", 0, 1, [{"key1": 1, "key2": "some-test-str", "key3": True}], 200),
            ("q", 9, 2, [{"key1": 10, "key2": "some-test-str", "key3": True}], 200),
            ("q", 10, 2, [], 200),
        ],
    )
    @pytest.mark.asyncio
    async def test_simple_test_paginated_model_j_response(
        self,
        query,
        offset,
        limit,
        expected_result,
        expected_status,
        async_test_client: AsyncTestClient,
    ):
        url = f"/test/simple-test-paginated_model-j-response?query={query}&limit={limit}&offset={offset}"
        response = await async_test_client.get(url)
        assert response.status_code == expected_status, response.text
        j_response = response.json()
        assert True.__eq__(j_response["success"])
        assert j_response["code"] == expected_status
        assert j_response["content"]["count"] == 10
        assert j_response["content"]["param"]["query"] == query
        assert j_response["content"]["param"]["offset"] == offset
        assert j_response["content"]["param"]["limit"] == limit
        assert j_response["content"]["results"] == expected_result
        if offset >= 10:
            assert j_response["content"]["next"] is None

        if offset <= 0:
            assert j_response["content"]["previous"] is None
