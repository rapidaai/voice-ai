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

import functools
import logging
from typing import List, Optional

import yaml
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

from app.configs.auth_config import AuthenticationConfig
from app.configs.celery_config import CeleryConfig
from app.configs.elastic_search_config import ElasticSearchConfig
from app.configs.extractor_config import ExtractorConfig
from app.configs.internal_service_config import InternalServiceConfig
from app.configs.postgres_config import PostgresConfig
from app.configs.redis_config import RedisConfig
from app.configs.storage_config import AssetStoreConfig

_log = logging.getLogger("app.config")


class ApplicationSettings(BaseSettings):
    """
    A wrapper class from config paths to config values.
    Paths are dot-separated expressions such as <code>foo.bar.baz</code>.
    Values are as in
    (booleans, strings, numbers, lists, or objects), represented by
    pydantic.
    """

    #
    # Service name as identifiers for service
    service_name: str = "knowledge-api"
    # defined host for service
    host: str = "0.0.0.0"
    port: int = "7474"

    # cors config for service
    # all origins are allowed by default
    cors_allow_origins: Optional[List[str]] = Field(
        ["*"],
        title="CORS allowed origins",
        description="List of origins allowed for CORS requests",
    )

    # Indicate that cookies should be supported for cross-origin requests.
    cors_allow_credentials: bool = True

    # A list of HTTP methods that should be allowed for cross-origin requests.
    cors_allow_methods: List[str] = Field(
        ["*"],
        title="CORS allowed method",
        description="List of method allowed for CORS requests",
    )

    # A list of HTTP request headers that should be supported for cross-origin requests
    cors_allow_headers: List[str] = Field(
        ["*"],
        title="CORS allowed headers",
        description="List of headers allowed for CORS requests",
    )

    # logging config
    log_level: Optional[str] = "INFO"
    # if url is empty then it will be disabled
    openapi_url: Optional[str] = None

    # Redis instance to be connected
    # current implementation support only singleTon redis
    # for cluster and sentinel it's not available.
    redis: Optional[RedisConfig] = Field(
        default=None,
        title="Redis connection configuration",
        description="the redis connection which is needed",
    )

    #
    # Postgres connection configuration
    postgres: Optional[PostgresConfig] = Field(title="Postgres connection config")

    # # elastic search connection configuration
    # # - Single node support enabled
    elastic_search: Optional[ElasticSearchConfig] = Field(
        default=None, title="Elastic search configs"
    )

    # Client authentication
    internal_service: Optional[InternalServiceConfig] = Field(
        default=None, title="All the internal service config"
    )

    # storage
    storage: Optional[AssetStoreConfig] = Field(
        default=None, title="storage config for the application"
    )
    #

    authentication_config: Optional[AuthenticationConfig] = Field(
        default=None, description="auth config for internal service " "communication"
    )

    knowledge_extractor_config: Optional[ExtractorConfig] = Field(
        description="config for extracting knowledge from files"
    )

    model_config = SettingsConfigDict(env_file_encoding="utf-8", extra="ignore")


@functools.lru_cache()
def get_settings() -> "ApplicationSettings":
    """Get current app settings."""
    with open("config.yaml", "r") as file:
        config_data = yaml.safe_load(file)
    config = ApplicationSettings(**config_data)
    return config
