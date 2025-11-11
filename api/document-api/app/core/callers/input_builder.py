"""
author: prashant.srivastav
"""

from typing import MutableMapping, Union, Mapping

from google.protobuf import any_pb2
from google.protobuf.wrappers_pb2 import StringValue

from app.bridges.artifacts.protos.integration_api_pb2 import (
    Credential,
    EmbeddingRequest,
)


def build_embedding_input(
    credential: Credential,
    contents: MutableMapping[int, str],
    parameters: Mapping[str, str],
    additional_data: Union[MutableMapping[str, str], None] = None,
) -> EmbeddingRequest:
    parameters_dict = {}
    for k, v in parameters.items():
        if k and v:
            any_msg = any_pb2.Any()
            string_value = StringValue(value=v)
            any_msg.Pack(string_value)
            parameters_dict[k] = any_msg
    return EmbeddingRequest(
        credential=credential,
        content=contents,
        modelParameters=parameters_dict,
        additionalData={
            key: str(value) for key, value in (additional_data or {}).items()
        },
    )
