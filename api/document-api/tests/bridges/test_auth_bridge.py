"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

from unittest.mock import AsyncMock

import pytest

from app.authentication import Authentication
from app.bridges.artifacts.protos import web_api_pb2
from app.bridges.internals.auth_bridge import AuthBridge
from app.exceptions.bridges_exceptions import BridgeException


def make_bridge() -> AuthBridge:
    bridge = AuthBridge.__new__(AuthBridge)
    bridge.service_url = "localhost:9001"
    return bridge


def authentication_response():
    return {
        "success": True,
        "code": 200,
        "authentication": {
            "principal": {
                "subject": {"type": "user", "id": "7"},
                "selectedTenant": {"organizationId": "0", "projectId": "0"},
                "proof": {
                    "method": "user_session",
                    "reference": {"type": "user_session", "value": "8"},
                },
            },
            "user": {
                "id": "7",
                "name": "Ada",
                "email": "ada@example.com",
                "status": "ACTIVE",
            },
        },
        "token": {"id": "8", "token": "token", "tokenType": "Bearer"},
    }


@pytest.mark.asyncio
async def test_authorize_returns_authentication_without_token():
    bridge = make_bridge()
    bridge.fetch = AsyncMock(return_value=authentication_response())

    result = await bridge.authorize("Bearer token", "7")

    assert result["principal"]["subject"] == {"type": "user", "id": "7"}
    assert result["user"]["email"] == "ada@example.com"
    assert "token" not in result
    tenant = Authentication.model_validate(result).principal.selected_tenant
    assert tenant.organization_id == 0
    assert tenant.project_id == 0
    assert bridge.fetch.call_args.kwargs["attr"] == "Authorize"
    assert isinstance(
        bridge.fetch.call_args.kwargs["message_type"], web_api_pb2.AuthorizeRequest
    )


@pytest.mark.asyncio
async def test_authorize_forwards_selected_project():
    bridge = make_bridge()
    bridge.fetch = AsyncMock(return_value=authentication_response())

    await bridge.authorize(
        "Bearer token", "7", project_id="11", organization_id="9"
    )

    metadata = {key: value for key, value in bridge.fetch.call_args.kwargs["metadata"]}
    assert metadata == {
        "authorization": "Bearer token",
        "x-auth-id": "7",
        "x-organization-id": "9",
        "x-project-id": "11",
    }


@pytest.mark.asyncio
async def test_authorize_rejects_missing_authentication():
    bridge = make_bridge()
    response = authentication_response()
    del response["authentication"]
    bridge.fetch = AsyncMock(return_value=response)

    with pytest.raises(BridgeException):
        await bridge.authorize("Bearer token", "7")


@pytest.mark.asyncio
async def test_authorize_rejects_missing_token():
    bridge = make_bridge()
    response = authentication_response()
    del response["token"]
    bridge.fetch = AsyncMock(return_value=response)

    with pytest.raises(BridgeException):
        await bridge.authorize("Bearer token", "7")


@pytest.mark.asyncio
async def test_authorize_rejects_failed_response():
    bridge = make_bridge()
    response = authentication_response()
    response["success"] = False
    bridge.fetch = AsyncMock(return_value=response)

    with pytest.raises(BridgeException):
        await bridge.authorize("Bearer token", "7")


@pytest.mark.asyncio
async def test_authorize_rejects_legacy_session_wrapper():
    bridge = make_bridge()
    response = authentication_response()
    response["session"] = {
        "authentication": response.pop("authentication"),
        "token": response.pop("token"),
    }
    bridge.fetch = AsyncMock(return_value=response)

    with pytest.raises(BridgeException):
        await bridge.authorize("Bearer token", "7")


def test_scope_authorize_contract_is_absent():
    assert not hasattr(AuthBridge, "scope_authorize")
