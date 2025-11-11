import logging
from typing import Optional

from app.connectors.elstic_search_connector import ElasticSearchConnector
from app.core.rag.datasource.vdb import constants
from app.core.rag.datasource.vdb.constants import VectorType
from app.core.rag.datasource.vdb.vector_base import BaseVector
from app.core.rag.models.document import Document
from app.exceptions.pipeline_exception import VectorDatabaseIndexingException
from app.utils.general import generate_text_hash

logger = logging.getLogger(__name__)


class OpenSearchVector(BaseVector):
    # The line `opensearch: ElasticSearchConnector` is defining a class attribute `opensearch` of type
    # `ElasticSearchConnector`. This attribute is used to store an instance of the
    # `ElasticSearchConnector` class that will be passed to the `OpenSearchVector` class during
    # initialization. This allows the `OpenSearchVector` class to interact with an instance of
    # `ElasticSearchConnector` for performing operations related to OpenSearch (Elasticsearch) data
    # storage and retrieval.
    # The `opensearch` attribute in the `OpenSearchVector` class is used to store an instance of the
    # `ElasticSearchConnector` class. This attribute allows the `OpenSearchVector` class to interact
    # with an instance of `ElasticSearchConnector` for performing operations related to OpenSearch
    # (Elasticsearch) data storage and retrieval. The `opensearch` attribute is initialized in the
    # constructor of the `OpenSearchVector` class and is used throughout the class methods to execute
    # operations such as creating collections, adding texts, searching by vector, searching by full
    # text, deleting documents, and more using the Elasticsearch connection provided by the
    # `opensearch` attribute.
    opensearch: ElasticSearchConnector

    def __init__(self, collection_name: str, opensearch: ElasticSearchConnector):
        super().__init__(collection_name)
        self.opensearch = opensearch

    async def get_type(self) -> str:
        return VectorType.OPENSEARCH

    async def create(
            self, texts: list[Document], embeddings: list[list[float]], **kwargs
    ):
        metadatas = [d.metadata for d in texts]
        await self.create_collection(embeddings, metadatas)
        await self.add_texts(texts, embeddings)

    async def add_texts(
            self, documents: list[Document], embeddings: list[list[float]], **kwargs
    ):
        actions = []
        for i in range(len(documents)):
            document_id = generate_text_hash(
                documents[i].page_content
            )  # Use hash as the document ID
            actions.append(
                {
                    "update": {
                        "_index": self._collection_name.lower(),
                        "_id": document_id,  # Use the hash as the ID
                    }
                }
            )

            actions.append(
                {
                    "doc": {
                        constants.Field.DOCUMENT_HASH_KEY.value: document_id,
                        constants.Field.DOCUMENT_ID_KEY.value: document_id,
                        constants.Field.VECTOR_KEY.value: embeddings[i],
                        constants.Field.TEXT_KEY.value: documents[i].page_content,
                        constants.Field.METADATA_KEY.value: documents[i].metadata,
                        constants.Field.ENTITIES_KEY.value: documents[i].entities,
                    },
                    "doc_as_upsert": True,
                }
            )

        # Perform the bulk operation

        try:
            response = await self.opensearch.connection.bulk(body=actions)
            if response["errors"]:
                logger.error("Bulk insert encountered errors: %s", response["items"])
                raise VectorDatabaseIndexingException("unable to index the document")

            return response

        except Exception as ex:
            logger.debug(f"got exception with {ex}")

    async def create_collection(
            self,
            embeddings: list,
            metadatas: Optional[list[dict]] = None,
            index_params: Optional[dict] = None,
    ):
        if not await self.opensearch.connection.indices.exists(
                index=self._collection_name.lower()
        ):
            index_body = {
                "settings": {
                    "number_of_shards": 1,
                    "number_of_replicas": 1,
                    "index": {"knn": True}
                },
                "mappings": {
                    "properties": {
                        constants.Field.TEXT_KEY.value: {"type": "text"},
                        constants.Field.DOCUMENT_ID_KEY.value: {"type": "keyword"},
                        constants.Field.DOCUMENT_HASH_KEY.value: {"type": "keyword"},
                        constants.Field.VECTOR_KEY.value: {
                            "type": "knn_vector",
                            "dimension": len(
                                embeddings[0]
                            ),  # Make sure the dimension is correct here
                            "method": {
                                "name": "hnsw",
                                "space_type": "l2",
                                "engine": "faiss",
                                "parameters": {"ef_construction": 64, "m": 8},
                            },
                        },
                        constants.Field.ENTITIES_KEY.value: {
                            "type": "object",
                            "dynamic": True,
                        },
                        constants.Field.METADATA_KEY.value: {
                            "type": "object",
                            "properties": {
                                constants.Field.KNOWLEDGE_DOCUMENT_ID_KEY.value: {
                                    "type": "keyword"
                                },
                                constants.Field.KNOWLEDGE_ID_KEY.value: {
                                    "type": "keyword"
                                },
                                constants.Field.PROJECT_ID_KEY.value: {
                                    "type": "keyword"
                                },
                                constants.Field.ORGANIZATION_ID_KEY.value: {
                                    "type": "keyword"
                                },
                            },
                        },
                    }
                },
            }

            await self.opensearch.connection.indices.create(
                index=self._collection_name.lower(), body=index_body
            )

    async def text_exists(self, id: str) -> bool:
        try:
            response = await self.opensearch.connection.get(
                index=self._collection_name.lower(), id=id
            )
            return response["found"]  # Returns True if found, False otherwise
        except Exception as e:
            if e.status == 404:
                return False  # Document does not exist
            elif e.status == 400:  # Bad Request or index not found
                return False  # Index might not exist
            # Optionally handle other exceptions as needed
            return False  # Return False for any other error
