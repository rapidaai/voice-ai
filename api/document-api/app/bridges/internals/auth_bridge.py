"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""
import logging

from google.protobuf.json_format import MessageToDict, ParseDict, ParseError
from grpc.aio import Metadata

from app.bridges import GRPCBridge
from app.bridges.artifacts.protos import (
    web_api_pb2,
    web_api_pb2_grpc,
)
from app.exceptions.bridges_exceptions import BridgeException

_log = logging.getLogger("bridges.auth_bridge")


class AuthBridge(GRPCBridge):
    async def authorize(
        self,
        auth_token: str,
        user_id: str,
        project_id: str = None,
        organization_id: str = None,
    ) -> dict:
        # metadata for request
        _metadata: Metadata = Metadata()
        _metadata.add("authorization", auth_token)
        _metadata.add("x-auth-id", user_id)
        if organization_id:
            _metadata.add("x-organization-id", organization_id)
        if project_id:
            _metadata.add("x-project-id", project_id)

        response = await self.fetch(
            stub=web_api_pb2_grpc.AuthenticationServiceStub,
            attr="Authorize",
            message_type=web_api_pb2.AuthorizeRequest(),
            preserving_proto_field_name=True,
            metadata=_metadata
        )

        try:
            result = ParseDict(response, web_api_pb2.AuthenticateResponse())
        except (ParseError, TypeError, ValueError) as error:
            raise BridgeException(
                message="Unable to receive provider credentials.",
                bridge_name="web",
            ) from error
        if (
            not result
            or not result.success
            or not result.HasField("authentication")
            or not result.HasField("token")
        ):
            raise BridgeException(
                message="Unable to receive provider credentials.",
                bridge_name="web",
            )
        return MessageToDict(
            result.authentication,
            preserving_proto_field_name=False,
        )
