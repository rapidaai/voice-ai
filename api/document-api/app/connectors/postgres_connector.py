import logging
from typing import Optional

from sqlalchemy import create_engine
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session

from app.configs.postgres_config import PostgresConfig
from app.connectors import Connector
from app.exceptions.connector_exception import ConnectorClientFailureException
from app.observabilities import SpanOutcome, within_span

_log = logging.getLogger("app.connector.postgres")


class PostgresConnector(Connector):
    """
    PostgresConnector to manage the connection with
    Postgres and provide wrapped feature to easily maintain consistency of connection.
    """

    # version of postgres server {future}
    version: Optional[str]

    # Hold configuration if you need reconnect
    _config: PostgresConfig

    # name of connection
    _name: str

    # Connection Pool
    _connection_pool: Optional[Engine]

    def __init__(self, config: PostgresConfig, name: str = "postgres"):
        self._config = config
        self._name = name

    # return the name of connection
    @property
    def name(self) -> str:
        return self._name

    async def connect(self):
        with within_span(
            (
                f"PSQL Connection {self._config.host.lower()}:{self._config.port}"
                if self._config.port is not None
                else f"PSQL Connection {self._config.host.lower()}"
            ),
            span_type="external",
            span_subtype="postgres",
            span_action="connect",
        ) as span:
            try:
                _log.info(f"Connecting to Postgres. {self._config.host.lower()}")

                # Create the asynchronous engine
                self._connection_pool = create_engine(
                    f"postgresql://{self._config.auth.user}:{self._config.auth.password.get_secret_value()}@{self._config.host}:{self._config.port}/{self._config.db}"
                )
                # acquire connection from pool
                if not await self.is_connected():
                    _log.error("Failed to connect to Postgres.")
                    raise ConnectorClientFailureException(
                        connector_name=self.name,
                        message="Failed to connect to Postgres.",
                    )
                _log.info("Connected to Postgres.")
            except Exception as e:
                _log.error(f"Failed to connect to postgres. {e}")
                span.set_status(SpanOutcome.FAILURE, description=str(e))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(e)
                )

    async def is_connected(self) -> bool:
        return self._connection_pool is not None

    # Close of connection or releasing from acquired connection
    async def disconnect(self):
        if self._connection_pool is not None:
            try:
                self._connection_pool.dispose()
            except Exception as e:
                _log.error(f"Failed to close connection. {e}")
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(e)
                )
        self._connection_pool = None

    @property
    def session(self) -> Session:
        return Session(
            bind=self._connection_pool,
            autocommit=False,
            autoflush=False,
        )
        # expire_on_commit=False)
