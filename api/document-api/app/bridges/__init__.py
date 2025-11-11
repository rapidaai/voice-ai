"""
author: prashant.srivastav

Base Bridge class for all RPC and gRPC clients.
- Changing this class will impact all RPC and gRPC clients and consumer service which implement and inherit
    this class.
- Considered as core computability layer for all RPC and gRPC clients the CHANGE REQUEST is not allowed.
- Mostly response being handled with JSON format.

"""
import logging
import time
from abc import ABC
from typing import Any, Dict
import grpc
from google.protobuf.json_format import MessageToDict
from google.protobuf.message import Message
from grpc import aio as grpc_aio
from grpc.aio import Metadata

from grpc.aio._channel import UnaryUnaryMultiCallable
from app.exceptions import RapidaException

_log = logging.getLogger("bridges.grpc_bridge")


class GRPCBridge(ABC):
    """
    Base class for all GRPC bridges.
    """

    # base url as hosts
    service_url: str

    @classmethod
    def __init__(cls, service_url: str):
        """
        Initialize the GRPC bridge
        :param service_url:
        """
        cls.service_url = service_url

    @classmethod
    def channel(cls):
        # if not cls.rapida_is_secure:
        return grpc_aio.insecure_channel(cls.service_url)
        # return grpc_aio.secure_channel(cls.service_url, grpc.ssl_channel_credentials())

    @classmethod
    async def fetch(
            cls,
            stub: Any,
            attr: str,
            message_type: Message,
            metadata: Metadata,
            **unmarshal_options,
    ) -> Dict[str, Any]:
        """
        Generic requestor
        :param stub:
        :param attr:
        :param message_type:
        :param metadata:
        :param unmarshal_options
        Args:
            including_default_value_fields: If True, singular primitive fields,
                repeated fields, and map fields will always be serialized.  If
                False, only serialize non-empty fields.  Singular message fields
                and one of fields are not affected by this option.
            preserving_proto_field_name: If True, use the original proto field
                names as defined in the .proto file. If False, convert the field
                names to lowerCamelCase.
            use_integers_for_enums: If true, print integers instead of enum names.
            descriptor_pool: A Descriptor Pool for resolving types. If None uses the
                default.
            float_precision: If set, use this to specify float field valid digits.
        :return:
        """
        started_request = time.time()
        try:
            _log.debug(f"grpc_bridge: in span of {attr}")
            async with cls.channel() as channel:
                _log.debug(f"grpc_bridge: created secure channel for {cls.service_url}")

                # get attribute from stub
                unary_unary: UnaryUnaryMultiCallable = getattr(stub(channel), attr)

                # request executed with asynchronous invocation of a unary-call RPC.
                results = await unary_unary(request=message_type, metadata=metadata)
                # parsing result and unmarshalling the result with given options
                json_result = MessageToDict(results, **unmarshal_options)
                _log.info(
                    f"{cls.__qualname__} {attr} [status:OK request:{time.time() - started_request}s]"
                )
                return json_result
        except grpc_aio.AioRpcError as ex:
            # if there is error from inter service client communication then log and throw scoped exception
            _log.error(
                f"{cls.__qualname__} {attr} [status:NOT_OK request:{time.time() - started_request}s]"
            )
            raise RapidaException(message=str(ex), error_code=500, status_code=400)
