"""app.routers"""
from fastapi import FastAPI

from app.commons.j_response import JResponse
from app.routers.health_check import H1
from app.routers.v1 import V1


def add_all_routers(app: FastAPI) -> None:
    app.include_router(
        H1, prefix="", tags=["health-check"], default_response_class=JResponse
    )
    app.include_router(V1, prefix="/v1", tags=["v1"])
