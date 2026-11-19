"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import logging
from contextlib import asynccontextmanager
from logging.config import dictConfig

# import uvicorn
from fastapi import FastAPI
from starlette.requests import Request
import uvicorn

from app.authentication import ServiceAssertionIssuer
from app.config import get_settings
from app.configs.log_config import LogConfig
from app.connectors import Connector
from app.connectors.connector_factory import attach_connectors
from app.exceptions import add_all_exception_handler
from app.middlewares import add_all_enabled_middleware
from app.routers import add_all_routers
from app.storage.connector_factory import attach_storage
from app.vars import service_name

settings = get_settings()
dictConfig(LogConfig(level=settings.log_level).model_dump())
_log = logging.getLogger("app")
# setting up service name which will be available throughout the application context
service_name.set(settings.service_name)

# Load destination-bound issuers before the request listener opens.
web_service_assertion_issuer = ServiceAssertionIssuer(
    destination="web-api",
    secret=settings.secret,
)
integration_service_assertion_issuer = ServiceAssertionIssuer(
    destination="integration-api",
    secret=settings.secret,
)

APP_STORAGE = {}


@asynccontextmanager
async def lifespan(app: FastAPI):
    cntr: Connector
    # load all enabled connector to app storage
    for cntr in attach_connectors(get_settings()):
        await cntr.connect()
        APP_STORAGE[cntr.name] = cntr
    # storage
    APP_STORAGE["storage"] = attach_storage(get_settings())

    yield
    if APP_STORAGE and type(APP_STORAGE) is dict:
        cntr: Connector
        # disconnecting all storage datasource
        for cntr in APP_STORAGE.values():
            await cntr.disconnect()


app = FastAPI(
    lifespan=lifespan,
    title=f"{service_name.get()}",
    description=f"{service_name.get()}",
    openapi_url=get_settings().openapi_url,
    version="0.1.0",
)
app.state.web_service_assertion_issuer = web_service_assertion_issuer
app.state.integration_service_assertion_issuer = integration_service_assertion_issuer

# Add all enabled middleware
add_all_enabled_middleware(app=app, setting=get_settings())

# Add exception handler
add_all_exception_handler(app=app)


# add datasource
@app.middleware("http")
async def add_datasource(request: Request, call_next):
    """
    Storage object injection in every request depends on enabled connector.
    """
    request.state.datasource = APP_STORAGE
    response = await call_next(request)
    return response


# Include routers.
add_all_routers(app=app)


# Running of app.
if __name__ == "__main__":
    _log.info(f"starting the app with {get_settings().host}:{get_settings().port}")
    uvicorn.run(
        "app.main:app",
        host=get_settings().host,
        port=get_settings().port,
        access_log=False,
    )
