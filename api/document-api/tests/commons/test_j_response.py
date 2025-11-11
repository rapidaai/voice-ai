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
from pydantic import BaseModel

from app.commons.j_response import JResponse


class TestJResponse:
    @pytest.mark.asyncio
    @pytest.fixture(autouse=True)
    def setup_method(self, test_app):
        class TestModel(BaseModel):
            key1: int
            key2: str
            key3: bool

        @pytest.mark.asyncio
        @test_app.get(
            "/test/simple-test-j-response-with-dict", response_class=JResponse
        )
        async def simple_test_j_response_with_dict():
            return {"test": "ok", "nest-test": {"nest": "yup"}}

        @pytest.mark.asyncio
        @test_app.get(
            "/test/simple-test-j-response-with-model-exclude-and-include",
            response_class=JResponse,
            response_model=TestModel,
            response_model_include={"key1"},
            response_model_exclude={"key2"},
        )
        async def simple_test_j_response_with_model_exclude_and_include():
            return {"key1": 1, "key2": "some-test-str", "key3": True}

        @pytest.mark.asyncio
        @test_app.get(
            "/test/simple-test-j-response-default-ok", response_class=JResponse
        )
        async def simple_test_j_response_default_ok():
            return JResponse.default_ok({"key1": "value"})

        @pytest.mark.asyncio
        @test_app.get(
            "/test/simple-test-j-response-default-error", response_class=JResponse
        )
        async def simple_test_j_response_default_error():
            try:
                raise Exception("something-bad-happen")
            except Exception as e:
                return JResponse.default_on_error(
                    exc=e, error_code=422, error_message="something-bad-happen"
                )

    @pytest.mark.parametrize(
        "expected_data, expected_status",
        [
            ({"test": "ok", "nest-test": {"nest": "yup"}}, 200),
        ],
    )
    @pytest.mark.asyncio
    async def test_simple_test_j_response_with_dict(
        self, async_test_client: AsyncTestClient, expected_data, expected_status
    ):
        response = await async_test_client.get("/test/simple-test-j-response-with-dict")
        assert response.status_code == expected_status, response.text
        j_response = response.json()
        assert True.__eq__(j_response["success"])
        assert j_response["code"] == expected_status
        assert j_response["content"] == expected_data

    @pytest.mark.parametrize(
        "expected_data, expected_status",
        [
            ({"key1": 1}, 200),
        ],
    )
    @pytest.mark.asyncio
    async def test_simple_test_j_response_with_model_exclude_and_include(
        self, async_test_client: AsyncTestClient, expected_data, expected_status
    ):
        response = await async_test_client.get(
            "/test/simple-test-j-response-with-model-exclude-and-include"
        )
        assert response.status_code == expected_status, response.text
        j_response = response.json()
        assert True.__eq__(j_response["success"])
        assert j_response["code"] == expected_status
        assert j_response["content"] == expected_data

    @pytest.mark.parametrize(
        "expected_data, expected_status",
        [
            ({"key1": "value"}, 200),
        ],
    )
    @pytest.mark.asyncio
    async def test_simple_test_j_response_default_ok(
        self, async_test_client: AsyncTestClient, expected_data, expected_status
    ):
        response = await async_test_client.get(
            "/test/simple-test-j-response-default-ok"
        )
        assert response.status_code == expected_status, response.text
        j_response = response.json()
        assert True.__eq__(j_response["success"])
        assert j_response["code"] == expected_status
        assert j_response["content"] == expected_data

    @pytest.mark.parametrize(
        "expected_data, expected_status",
        [
            (
                {
                    "detail": "something-bad-happen",
                    "error_message": "something-bad-happen",
                },
                422,
            ),
        ],
    )
    @pytest.mark.asyncio
    async def test_simple_test_j_response_default_error(
        self, async_test_client: AsyncTestClient, expected_data, expected_status
    ):
        response = await async_test_client.get(
            "/test/simple-test-j-response-default-error"
        )
        assert response.status_code == expected_status, response.text
        j_response = response.json()
        assert False.__eq__(j_response["success"])
        assert j_response["code"] == expected_status
        assert j_response["content"] == expected_data
