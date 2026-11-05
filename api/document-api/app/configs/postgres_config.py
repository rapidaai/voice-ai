"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""
from typing import Optional

from pydantic import model_validator

from app.configs import ExternalDatasourceModel
from app.configs.auth.basic_auth import BasicAuth


class PostgresConfig(ExternalDatasourceModel):
    """
    Postgres Config Template
    """

    # Host of postgres without protocol and port
    host: str

    # Postgres running port
    port: int

    # Database name
    db: str

    # Support of user and password authentication to postgres
    auth: Optional[BasicAuth]

    # Minimum number of connection to be active at any given point of time
    ideal_connection: int = 1

    # Maximum number of connection to be active at any given point of time
    max_connection: int = 5

    @model_validator(mode="after")
    def validate_connection_limits(self):
        """Validate the retained and maximum connection limits."""
        if self.ideal_connection < 1:
            raise ValueError("ideal_connection must be at least 1")
        if self.max_connection < self.ideal_connection:
            raise ValueError("max_connection must be greater than or equal to ideal_connection")
        return self

    # do we need to place in docker container
    # or place as external service
    dockerize: Optional[bool] = True

    class Config:
        # need to be case-sensitive db-name (ato generated db name problem)
        env_nested_delimiter = "__"
        case_sensitive = True
        env_file_encoding = "utf-8"
