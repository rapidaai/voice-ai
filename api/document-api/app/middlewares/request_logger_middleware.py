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
import logging
import time

from fastapi import FastAPI
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request

from app.commons.j_response import JResponse
from app.exceptions import RapidaException
from app.utils.request import get_url_path_with_query_param

_log = logging.getLogger("app")


class RequestLoggerMiddleware(BaseHTTPMiddleware):
    """
    Request Logger
    Middleware to log request and auditing response time
    """

    def __init__(self, app: FastAPI, *args):
        super().__init__(app, *args)

    async def dispatch(self, request: Request, call_next):
        """
        Logging request and response time
        :param request:
        :param call_next:
        :return:
        """
        # start time
        start_time = time.time()
        status_code: int = 200
        try:
            response = await call_next(request)
            response.headers["X-Process-Time"] = str(time.time() - start_time)
            # update the http response code
            status_code = response.status_code
            return response
        except RapidaException as error:
            _log.error(f"exception while processing request {str(error)}")
            status_code = error.status_code
            return JResponse.default_on_error(
                exc=error, error_message=error.message, error_code=error.status_code
            )
        except Exception as err:
            _log.error(f"exception while processing request {str(err)}", exc_info=True)
            status_code = 500
            # only for unhandle exception
            # on generic error return valid json with error message and status code
            return JResponse.default_on_error(
                exc=err,
                error_message=f"Internal Server Error: {str(err)}",
                error_code=500,
            )
        finally:
            process_time = time.time() - start_time
            _log.info(
                f"{request.method} {get_url_path_with_query_param(request)} "
                f"[status:{status_code} request:{process_time}s]"
            )
