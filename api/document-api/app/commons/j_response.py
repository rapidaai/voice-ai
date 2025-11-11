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
import json
import typing
from collections import OrderedDict

from starlette.responses import JSONResponse


class JResponse(JSONResponse):
    """
    JResponse extended from starlette to add simplify in service context.
    Should be used as `response_class` argument to routes of your app:
        @app.get("/", response_class=JResponse)
    """

    def render(self, content: typing.Any) -> bytes:
        body = OrderedDict(
            [
                ("success", self.ok),
                ("content", content),
                ("code", self.status_code),
            ]
        )
        return json.dumps(
            body,
            ensure_ascii=False,
            allow_nan=False,
            indent=None,
            separators=(",", ":"),
        ).encode("utf-8")

    @property
    def ok(self) -> bool:
        return 200 <= self.status_code <= 299

    @staticmethod
    def default_on_error(
        exc: Exception, error_message: str, error_code: int
    ) -> JSONResponse:
        """
        Default support method to construct JResponse
        on error it can invoke
        JResponse.default_on_error(exc=exc, ,error_message ="some str", error_code=422)
        :param exc: exception
        :param error_message: human readable message
        :param error_code: any particular error code
        """
        return JResponse(
            content={"error_message": error_message, "detail": str(exc)},
            status_code=error_code,
        )

    @staticmethod
    def default_ok(data: typing.Any, code: int = 200) -> JSONResponse:
        """
        Default support method to construct JResponse
        JResponse.default_ok(data={}, code=200)
        JResponse.default_ok(data={})
        """
        return JResponse(content=data, status_code=code)
