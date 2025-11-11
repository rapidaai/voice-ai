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
from typing import Union

from starlette.requests import Request

from app.config import ApplicationSettings
from app.exceptions.connector_exception import (
    ConnectorNotThereException,
)
from app.storage.storage import Storage


def attach_storage(setting: ApplicationSettings) -> Union[Storage, None]:
    if setting.storage:
        return Storage(setting.storage)


async def get_me_storage(request) -> Storage:
    """
    Return elastic search connection wrapper class from request context
    :param request: request context
    :return: :class:`ElasticSearchConnector`.
    """
    key = "storage"
    try:
        if isinstance(request, Request):
            return request.state.datasource[key]
        return request.state["datasource"][key]
    except KeyError:
        raise ConnectorNotThereException(key, f"{key} is not enable in env.")


async def get_storage(request: Request) -> Storage:
    """
    Return elastic search connection wrapper class from request context
    :param request: request context
    :return: :class:`ElasticSearchConnector`.
    """
    return await get_me_storage(request)
