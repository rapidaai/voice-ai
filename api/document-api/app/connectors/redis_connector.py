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
from typing import Optional

import redis.asyncio as aioredis
from redis.exceptions import RedisError

from app.configs.redis_config import RedisConfig
from app.connectors import Connector
from app.exceptions.connector_exception import ConnectorClientFailureException
from app.observabilities import SpanOutcome, within_span

_log = logging.getLogger("app.connector.redis")


class RedisConnector(Connector):
    """
    Redis connection wrapper class
    Hold connection and configuration
    """

    # version of redis server {future}
    version: Optional[str]

    # connection instance
    connection: Optional[aioredis.Redis] = None

    # configurations or env
    _config: RedisConfig

    # name of connection
    _name: str

    __connection_pool: Optional[aioredis.BlockingConnectionPool] = None

    def __init__(self, config: RedisConfig, name: str = "redis"):
        self._config = config
        self._name = name

    async def connect(self):
        """Create connection pool from the Redis server"""

        # checking if connection and connection pool
        # is already available don't create new connection
        if self.connection and self.__connection_pool:
            return

        try:
            self.__connection_pool = aioredis.BlockingConnectionPool(
                host=self._config.host,
                port=self._config.port,
                max_connections=self._config.max_connection,
                db=self._config.db,
                decode_responses=self._config.decode_responses,
            )
            _log.info("Trying to connect to redis.")
            if self._config.auth is not None:
                self.connection = await aioredis.Redis(
                    connection_pool=self.__connection_pool,
                    username=self._config.auth.user,
                    password=self._config.auth.password.get_secret_value(),
                    encoding=self._config.charset,
                )
                _log.info("Connecting with auth.")
            else:
                self.connection = aioredis.StrictRedis(
                    connection_pool=self.__connection_pool,
                    encoding=self._config.charset,
                )

            # ping and fail if connection is not established
            if not await self.is_connected():
                raise ConnectorClientFailureException(
                    connector_name=self.name, message="Failed to connect to redis"
                )
            _log.info("Connected to redis.")

        except Exception as e:
            self.connection = None
            self.__connection_pool = None
            _log.error(f"Failed to connect to redis. {e}")
            raise ConnectorClientFailureException(
                connector_name=self.name, message=str(e)
            )

    async def disconnect(self):
        """Disconnects from the Redis server"""
        _log.info("Disconnect with redis.")
        if not await self.is_connected():
            _log.info("Disconnect called when it is not connected.")
            return
        try:
            # close the connection
            # close the connection pool. This will close all the connections in the pool.
            await self.connection.close()
            await self.__connection_pool.disconnect()
            if hasattr(self.connection, "wait_closed"):
                await self.connection.wait_closed()  # type: ignore[union-attr]
        except OSError:
            pass
        self.connection = None
        self.__connection_pool = None

    # check if connection is active and available to use
    async def is_connected(self) -> bool:
        return bool(self.connection and await self.ping is True)

    # ping redis server
    @property
    async def ping(self) -> bool:
        return await self.operate("ping")

    # return name of connection
    @property
    def name(self) -> str:
        return self._name

    async def pipeline(self, transaction: bool = True):
        # command: str, *arg, **kwargs
        with within_span(
            f"REDIS {self._config.host.lower()}:{self._config.port}",
            span_type="external",
            span_subtype="redis",
            span_action="pipeline",
        ) as span:
            try:
                await self.connect()
                return self.connection.pipeline(transaction=transaction)
            except RedisError as redis_error:
                _log.error(f"Failed to connect for {self.name} . {str(redis_error)}")
                span.set_status(SpanOutcome.FAILURE, description=str(redis_error))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(redis_error)
                )
            except Exception as err:
                _log.error(
                    f"Failed to do the operation pipeline from {self.name}. {str(err)}"
                )
                span.set_status(SpanOutcome.FAILURE, description=str(err))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(err)
                )

    async def operate(self, command: str, *arg, **kwargs):
        """
        Execute redis command
        Definition of all the command and their arguments are declarative in Redis Connection.
        Connection class is used to execute the command
        :param command: command to execute
        :param arg: arguments for command
        :param kwargs: keyword arguments for command
        """
        try:
            with within_span(
                f"REDIS {self._config.host.lower()}:{self._config.port}",
                span_type="external",
                span_subtype="redis",
                span_action=command,
            ):
                await self.connect()
                return await getattr(self.connection, command)(*arg, **kwargs)
        except RedisError as redis_error:
            _log.error(f"Failed to connect for {self.name} . {str(redis_error)}")
            raise ConnectorClientFailureException(
                connector_name=self.name, message=str(redis_error)
            )
        except Exception as err:
            _log.error(
                f"Failed to do the operation {command} from {self.name}. {str(err)}"
            )
            raise ConnectorClientFailureException(
                connector_name=self.name, message=str(err)
            )
