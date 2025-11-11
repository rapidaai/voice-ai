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

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware as _CORSMiddleware

from app.config import ApplicationSettings

_log = logging.getLogger("app.middlewares.cors_middleware")


class CORSMiddleware(_CORSMiddleware):
    """
    CORS middleware for service
    Extension of starlette.middleware.cors.CORSMiddleware with default parameters
    """

    def __init__(self, app: FastAPI, settings: ApplicationSettings):
        super().__init__(
            app=app,
            allow_origins=settings.cors_allow_origins,
            allow_credentials=settings.cors_allow_credentials,
            allow_methods=settings.cors_allow_methods,
            allow_headers=settings.cors_allow_headers,
        )
