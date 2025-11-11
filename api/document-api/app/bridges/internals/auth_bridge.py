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

from google.protobuf.json_format import ParseDict, MessageToDict
from grpc.aio import Metadata

from app.bridges import GRPCBridge
from app.bridges.artifacts.protos import (
    web_api_pb2,
    web_api_pb2_grpc,
)
from app.exceptions.bridges_exceptions import BridgeException

_log = logging.getLogger("bridges.auth_bridge")


class AuthBridge(GRPCBridge):
    async def authorize(self, auth_token: str, user_id: str) -> dict:
        # metadata for request
        _metadata: Metadata = Metadata()
        _metadata.add("authorization", auth_token)
        _metadata.add("x-auth-id", user_id)

        response = await self.fetch(
            stub=web_api_pb2_grpc.AuthenticationServiceStub,
            attr="Authorize",
            message_type=web_api_pb2.AuthorizeRequest(),
            preserving_proto_field_name=True,
            metadata=_metadata
        )

        result = ParseDict(response, web_api_pb2.AuthenticateResponse())
        if not result or not result.success:
            raise BridgeException(message="Unable to receive provider credentials.", bridge_name="web")
        return MessageToDict(result.data)

    async def scope_authorize(self, token: str):
        # metadata for request
        _metadata: Metadata = Metadata()
        _metadata.add("x-api-key", token)

        response = await self.fetch(
            stub=web_api_pb2_grpc.AuthenticationServiceStub,
            attr="Invoke",
            message_type=web_api_pb2.AuthorizeRequest(),
            preserving_proto_field_name=True,
            metadata=_metadata
        )
        return ParseDict(response, web_api_pb2.AuthenticateResponse())
