"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import logging

from fastapi import FastAPI

from app.authentication import ServiceAssertionVerifier
from app.bridges.bridge_factory import service_grpc_client
from app.bridges.internals.auth_bridge import AuthBridge
from app.config import ApplicationSettings
from app.middlewares.contextual_logging_middleware import ContextualLoggingMiddleware
from app.middlewares.cors_middleware import CORSMiddleware
from app.middlewares.principal_authentication_middleware import PrincipalAuthenticationMiddleware
from app.middlewares.request_logger_middleware import RequestLoggerMiddleware

_log = logging.getLogger("app.middlewares")


def add_all_enabled_middleware(app: FastAPI, setting: ApplicationSettings):
    """
    Adding enabled apm middleware to service
    :param app: fastApi app
    :param setting: _Setting, settings of app
    """
    app.add_middleware(
        PrincipalAuthenticationMiddleware,
        authentication_config=setting.authentication_config,
        service_assertion_verifier=ServiceAssertionVerifier(
            setting.service_name, setting.secret
        ),
        user_authentication_resolver=service_grpc_client(
            bridge=AuthBridge,
            service_url=setting.authentication_config.user.auth_host,
        ).authorize,
    )

    # if there are any allowed origins, add middleware for cors with all configured cors settings
    app.add_middleware(CORSMiddleware, settings=setting)

    # add all default middleware
    app.add_middleware(RequestLoggerMiddleware)

    # adding contextual middleware
    app.add_middleware(ContextualLoggingMiddleware, settings=setting)
