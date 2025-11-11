"""
author: prashant.srivastav
"""
import logging
from typing import List, Tuple
from google.protobuf.json_format import ParseDict
from grpc.aio import Metadata
from app.bridges import GRPCBridge
from app.bridges.artifacts.protos import (
    integration_api_pb2_grpc,
)
from app.bridges.artifacts.protos.common_pb2 import Metric
from app.bridges.artifacts.protos.integration_api_pb2 import Embedding, EmbeddingRequest, EmbeddingResponse
from app.exceptions.bridges_exceptions import BridgeException

_log = logging.getLogger("bridges.integration_bridge")


class IntegrationBridge(GRPCBridge):
    
    async def get_embedding(self,
                            auth_token: str,
                            provider_name: str,
                            request: EmbeddingRequest) -> Tuple[List[Embedding], List[Metric]]:
        """
        This function retrieves embedding data from a specified provider using gRPC communication and
        returns the embedding data and associated metrics.
        :param auth_token: The `auth_token` parameter in the `get_embedding` method is a string that
        represents the authentication token used to access the embedding service. It is typically a
        secure token or key that grants permission to make requests to the service on behalf of the user
        or application
        :type auth_token: str
        :param provider_name: The `provider_name` parameter in the `get_embedding` method is used to
        specify the name of the service provider for which you want to fetch embeddings. It is used to
        determine which gRPC stub to use based on the mapping defined in the `provider_stub_map`
        dictionary. The available provider names
        :type provider_name: str
        :param request: The `request` parameter in the `get_embedding` method is of type
        `EmbeddingRequest`. It is used to pass the request data needed to fetch embeddings from a
        specific provider. The `EmbeddingRequest` likely contains information such as the text or data
        for which embeddings are being requested, any
        :type request: EmbeddingRequest
        :return: The `get_embedding` method returns a tuple containing two lists: one list of
        `Embedding` objects and one list of `Metric` objects.
        """
        # metadata for request
        _metadata: Metadata = Metadata()
        _metadata.add("x-internal-service-key", auth_token)

        # Choose the appropriate stub based on provider_name

        provider_stub_map = {
            "cohere": integration_api_pb2_grpc.CohereServiceStub,
            "openai": integration_api_pb2_grpc.OpenAiServiceStub,
            "voyageai": integration_api_pb2_grpc.VoyageAiServiceStub,
            "bedrock": integration_api_pb2_grpc.BedrockServiceStub,
            "azure-openai": integration_api_pb2_grpc.AzureServiceStub,
            "google": integration_api_pb2_grpc.GoogleServiceStub
        }

        # Fetch the correct service stub for the provider
        if provider_name in provider_stub_map:
            stub_class = provider_stub_map[provider_name]
            response = await self.fetch(
                stub=stub_class,
                attr="Embedding",
                message_type=request,
                preserving_proto_field_name=True,
                metadata=_metadata
            )
        else:
            raise BridgeException(message="Unsupported provider name.", bridge_name="integration")

        # Parse the response to an EmbeddingResponse
        result = ParseDict(response, EmbeddingResponse())

        # Handle any errors in the result
        if not result or not result.success:
            raise BridgeException(message="Failed to retrieve embedding response.", bridge_name="integration")

        # Convert response data to a dictionary
        return result.data, result.metrics
