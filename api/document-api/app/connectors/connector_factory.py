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

from typing import Callable, Dict, List

from sqlalchemy.orm import Session
from starlette.requests import Request

from app.config import ApplicationSettings
from app.configs.auth.aws_auth import AWSAuth
from app.configs.elastic_search_config import ElasticSearchConfig
from app.configs.postgres_config import PostgresConfig
from app.configs.redis_config import RedisConfig
from app.connectors import Connector
from app.connectors.aws.s3_connector import S3Connector
from app.connectors.elstic_search_connector import ElasticSearchConnector
from app.connectors.postgres_connector import PostgresConnector
from app.connectors.redis_connector import RedisConnector
from app.exceptions.connector_exception import (
    ConnectorIllegalNameException,
    ConnectorNotThereException,
)


def attach_connectors(setting: ApplicationSettings) -> List[Connector]:
    """
    Given a setting provide all connector being valued
    Iterating every property of setting class and instantiate appropriated connector for that
    :param setting: app configurations
    :return: :class:`List[Connector]`.
    """
    enabled_connector: List[Connector] = []

    for key in setting.model_dump():
        if type(getattr(setting, key)) is RedisConfig:
            enabled_connector.append(
                RedisConnector(config=getattr(setting, key), name=key)
            )

        if type(getattr(setting, key)) is ElasticSearchConfig:
            enabled_connector.append(
                ElasticSearchConnector(config=getattr(setting, key), name=key)
            )

        if type(getattr(setting, key)) is PostgresConfig:
            enabled_connector.append(
                PostgresConnector(config=getattr(setting, key), name=key)
            )

    return enabled_connector


async def get_elastic_search(request: Request) -> "PostgresConnector":
    return await get_me_elastic_search(request)


async def get_me_elastic_search(request: Request) -> ElasticSearchConnector:
    """
    Return elastic search connection wrapper class from request context
    :param request: request context
    :return: :class:`ElasticSearchConnector`.
    """
    key = "elastic_search"
    try:
        if isinstance(request, Request):
            return request.state.datasource[key]
        return request.state["datasource"][key]
    except KeyError:
        raise ConnectorNotThereException(
            connector_name=key, message=f"{key} is not enable in env."
        )


async def get_me_redis(request: Request) -> RedisConnector:
    """
    Return redis connection wrapper class from request context
    :param request: request context
    :return: :class:`RedisConnector`.
    """
    key = "redis"
    try:
        return request.state.datasource[key]
    except KeyError:
        raise ConnectorNotThereException(
            connector_name=key, message=f"{key} is not enable in env."
        )


async def get_me_postgres_session(request: Request):
    try:
        connector = await get_me_postgres(request)
        db = connector.session
        try:
            yield db
        finally:
            db.close()
    except KeyError:
        raise ConnectorNotThereException(
            connector_name="session", message=f"SQLAlchemy is not enable in env."
        )


async def get_postgres(request: Request) -> "PostgresConnector":
    return await get_me_postgres(request)


async def get_me_postgres(request) -> "PostgresConnector":
    """
    Return postgres connection wrapper class from request context
    :param request: request context
    :return: :class:`PostgresConnector`.
    """
    key = "postgres"
    try:
        if isinstance(request, Request):
            return request.state.datasource[key]
        return request.state["datasource"][key]
    except KeyError:
        raise ConnectorNotThereException(
            connector_name=key, message=f"{key} is not enable in env."
        )


def get_me(connection_name: str) -> Callable[[Request], Connector]:
    """
    get connection from configurable connector
    code `Depends(get_me("elastic_search"))`
    :param connection_name: connection_name
    """

    def connector_dependency(request: Request):
        try:
            connection: Connector = request.state.datasource[connection_name]
            return connection
        except KeyError:

            raise ConnectorIllegalNameException(
                connector_name=connection_name,
                message=f"{connection_name} not found in context. choose possible keys {request.state.datasource}",
            )

    return connector_dependency


def get_all_connectors(request: Request) -> Dict:
    """
    get all the connectors from request context
    code `Depends(get_all_connectors)`
    :param request: request context
    """
    return request.state.datasource


def get_aws_s3_connector(config: AWSAuth) -> S3Connector:
    """
    get aws connector
    service `aws_s3`
    """
    return S3Connector(config)
