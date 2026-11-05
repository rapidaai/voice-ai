from unittest.mock import patch

import pytest
from pydantic import SecretStr, ValidationError

from app.configs.auth.basic_auth import BasicAuth
from app.configs.postgres_config import PostgresConfig
from app.connectors.postgres_connector import PostgresConnector
from app.exceptions.connector_exception import ConnectorClientFailureException


def postgres_config(ideal_connection: int = 1, max_connection: int = 5) -> PostgresConfig:
    return PostgresConfig(
        host="localhost",
        port=5432,
        db="assistant_db",
        auth=BasicAuth(user="rapida_user", password=SecretStr("test")),
        ideal_connection=ideal_connection,
        max_connection=max_connection,
    )


@pytest.mark.asyncio
async def test_connect_applies_configured_pool_limits():
    connector = PostgresConnector(postgres_config())

    with patch("app.connectors.postgres_connector.create_engine") as create_engine:
        await connector.connect()

    create_engine.assert_called_once_with(
        "postgresql://rapida_user:test@localhost:5432/assistant_db",
        pool_size=1,
        max_overflow=4,
    )


@pytest.mark.asyncio
async def test_connect_wraps_engine_creation_failure():
    connector = PostgresConnector(postgres_config())

    with patch(
        "app.connectors.postgres_connector.create_engine",
        side_effect=RuntimeError("engine unavailable"),
    ), pytest.raises(ConnectorClientFailureException):
        await connector.connect()


def test_postgres_config_rejects_idle_connections_above_maximum():
    with pytest.raises(ValidationError):
        postgres_config(ideal_connection=6, max_connection=5)
