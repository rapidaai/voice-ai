import datetime
import logging
import time
import traceback
from typing import Optional, cast, List

from app.bridges.internals.integration_bridge import IntegrationBridge
from app.bridges.internals.vault_bridge import VaultBridge
from app.connectors.elstic_search_connector import ElasticSearchConnector
from app.connectors.postgres_connector import PostgresConnector
from app.core.docstore.postgres_knowledge_document_store import (
    KnowledgeDocumentStore,
    PostgresKnowledgeDocumentStore,
)
from app.core.embedding.plain_embedding import PlainEmbedder
from app.core.model_runtime.model_manager import ModelManager
from app.core.rag.datasource.vdb.opensearch.opensearch_vector import OpenSearchVector
from app.core.rag.index_processor.index_processor_base import BaseIndexProcessor
from app.core.rag.index_processor.index_processor_factory import IndexProcessorFactory
from app.core.rag.models.document import Document
from app.exceptions.document_exception import (
    DocumentIsPausedException,
)
from app.models.knowledge_model import (
    Knowledge,
    KnowledgeDocument,
    KnowledgeDocumentSegment,
)
from app.services.knowledge_service import KnowledgeService
from app.storage.storage import Storage

_log = logging.getLogger("app.core.indexing_runner")


class IndexingRunner:
    # The lines `postgres: PostgresConnector` and `storage: Storage` are defining class attributes
    # `postgres` and `storage` respectively in the `IndexingRunner` class. These attributes are of
    # type `PostgresConnector` and `Storage` respectively. By defining these attributes in the class,
    # you are declaring that instances of the `IndexingRunner` class will have access to a
    # `PostgresConnector` instance stored in the `postgres` attribute and a `Storage` instance stored
    # in the `storage` attribute. This allows the `IndexingRunner` class to interact with the database
    # using the `PostgresConnector` and with the storage system using the `Storage` class.

    # postgres: PostgresConnector

    knowledge_service: KnowledgeService
    storage: Storage
    index_type: str
    model_manager: ModelManager

    #
    knowledge: Knowledge
    knowledge_document: KnowledgeDocument
    knowledge_document_store: KnowledgeDocumentStore

    postgres: PostgresConnector
    elastic_search: ElasticSearchConnector

    def __init__(
            self,
            storage: Storage,
            postgres: PostgresConnector,
            elastic_search: ElasticSearchConnector,
            knowledge: Knowledge,
            knowledge_document: KnowledgeDocument,
            integration_client: IntegrationBridge,
            vault_client: VaultBridge,
            index_type="paragraph-index",
    ):
        """
        The function initializes an object with references to a storage object and a PostgreSQL connector
        object.

        :param storage: The `storage` parameter is an object of the `Storage` class, which is likely used
        for managing and interacting with some form of storage or data repository. It could be responsible
        for reading, writing, and manipulating data in a specific storage system
        :type storage: Storage
        :param postgres: The `postgres` parameter in the `__init__` method is of type `PostgresConnector`.
        It seems like this parameter is used to pass an instance of the `PostgresConnector` class to the
        object being initialized. This allows the object to interact with a PostgreSQL database using the
        methods and
        :type postgres: PostgresConnector
        """
        self.storage = storage
        self.postgres = postgres
        self.elastic_search = elastic_search
        self.knowledge_service = KnowledgeService(postgres)
        self.index_type = index_type
        self.knowledge = knowledge
        self.knowledge_document = knowledge_document
        self.model_manager = ModelManager(
            vault_service_client=vault_client,
            integration_service_client=integration_client,
            project_id=knowledge.project_id,
            organization_id=knowledge.organization_id,
            model_provider_name=knowledge.embedding_model_provider_name,
            model_provider_id=knowledge.embedding_model_provider_id,
            model_parameters=self.knowledge_service.get_knowledge_model_options(knowledge.id),
            references={
                "knowledge_id": self.knowledge.id,
                "knowledge_document_id": self.knowledge_document.id,
            },
        )
        self.knowledge_document_store = PostgresKnowledgeDocumentStore(
            postgres=postgres,
            knowledge=knowledge,
            knowledge_document=knowledge_document,
        )

    async def run(self):
        """
        This function runs an indexing process for a list of knowledge documents, handling exceptions and
        logging information along the way.
        """

        try:
            index_processor = IndexProcessorFactory(
                self.index_type
            ).init_index_processor(self.storage)
            # extract
            text_docs = await self._extract(index_processor)
            documents = await self._transform(index_processor, text_docs)
            await self._load(index_processor=index_processor, documents=documents)

        except DocumentIsPausedException as e:
            _log.error(traceback.format_exc())
            await self._update_document_index_status(
                after_indexing_status="error",
                extra_update_params={
                    KnowledgeDocument.error: str(e),
                    KnowledgeDocument.completed_at: datetime.datetime.now(
                        datetime.timezone.utc
                    ).replace(tzinfo=None),
                },
            )
        except Exception as e:
            _log.error(traceback.format_exc())
            await self._update_document_index_status(
                after_indexing_status="error",
                extra_update_params={
                    KnowledgeDocument.error: str(e),
                    KnowledgeDocument.completed_at: datetime.datetime.now(
                        datetime.timezone.utc
                    ).replace(tzinfo=None),
                },
            )

    async def _extract(self, index_processor: BaseIndexProcessor) -> List[Document]:
        """
        This function extracts text documents from a knowledge document using the provided index processor,
        """
        text_docs = await index_processor.extract(
            knowledge_document=self.knowledge_document
        )
        await self._update_document_index_status(
            after_indexing_status="splitting",
            extra_update_params={
                KnowledgeDocument.word_count: sum(
                    [len(text_doc.page_content) for text_doc in text_docs]
                ),
                KnowledgeDocument.parsing_completed_at: datetime.datetime.now(
                    datetime.timezone.utc
                ).replace(tzinfo=None),
            },
        )

        # replace doc id to document model id
        text_docs = cast(list[Document], text_docs)
        for text_doc in text_docs:
            text_doc.metadata["knowledge_document_id"] = self.knowledge_document.id
            text_doc.metadata["knowledge_id"] = self.knowledge.id

        return text_docs

    # async def _model_manager(self) -> ModelManager:
    #     # The above code is creating an instance of `ModelManagerFactory` with three service clients
    #     # (`providerClient`, `vaultClient`, `integrationClient`) as arguments. It then calls the
    #     # `get_model_manager` method on this instance with various parameters such as `provider_id`,
    #     # `provider_model_id`, `project_id`, `organization_id`, and `docs`. This method is likely responsible
    #     # for retrieving a model manager object based on the provided parameters.
    #     return await self.model_manager.initialize(
    #         model_provider_id=self.knowledge.embedding_model_provider_id,
    #         model_provider_name=self.knowledge.embedding_model_provider_name,
    #         project_id=self.knowledge.project_id,
    #         organization_id=self.knowledge.organization_id,
    #     )

    async def _load(
            self,
            index_processor: BaseIndexProcessor,
            documents: List[Document],
    ) -> None:
        """
        insert index and update document/segment status to completed
        """
        indexing_start_at = time.perf_counter()
        await self._load_segments(documents=documents)

        # create the collection that will have the chunks
        collection_name = self.knowledge.storage_namespace
        _log.info(
            f"collection is being created for id={self.knowledge.id}  collection_name={collection_name}"
        )
        #

        tokens = 0
        chunk_size = 50
        #
        for i in range(0, len(documents), chunk_size):
            chunk_documents = documents[i: i + chunk_size]
            tokens += await self._process_chunk(
                collection_name=collection_name,
                index_processor=index_processor,
                chunk_documents=chunk_documents,
                model_manager=self.model_manager,
            )
        indexing_end_at = time.perf_counter()
        # update document status to completed
        _log.debug(f"Completing the processing of id={self.knowledge.id}")
        await self._update_document_index_status(
            after_indexing_status="completed",
            extra_update_params={
                KnowledgeDocument.token_count: tokens,
                KnowledgeDocument.completed_at: datetime.datetime.now(
                    datetime.timezone.utc
                ).replace(tzinfo=None),
                KnowledgeDocument.indexing_latency: indexing_end_at - indexing_start_at,
            },
        )

    async def _process_chunk(
            self,
            index_processor: BaseIndexProcessor,
            collection_name: str,
            chunk_documents: List[Document],
            model_manager: ModelManager,
    ) -> int:

        # load index

        tokens = await index_processor.load(
            knowledge_document=self.knowledge_document,
            documents=chunk_documents,
            embedder=PlainEmbedder(model_manager),
            vector_processor=OpenSearchVector(
                collection_name=collection_name, opensearch=self.elastic_search
            ),
        )

        document_ids = [document.document_id for document in chunk_documents]
        self.knowledge_service.complete_knowledge_document_segment(
            self.knowledge_document.id, document_ids=document_ids
        )
        return tokens

    async def _update_document_index_status(
            self, after_indexing_status: str, extra_update_params: Optional[dict] = None
    ) -> None:
        # The above code snippet is updating a KnowledgeDocument record in a PostgreSQL database. It
        # first retrieves the document with the specified `document_id`, checks if the document
        # exists, and then updates the `index_status` field of the document with the value specified
        # in `after_indexing_status`.
        extras = {KnowledgeDocument.index_status: after_indexing_status}
        if extra_update_params:
            extras.update(extra_update_params)

        self.knowledge_service.update_knowledge_document(
            knowledge_document_id=self.knowledge_document.id,
            extra_update_params=extras,
        )
        return

    async def _transform(
            self,
            index_processor: BaseIndexProcessor,
            text_docs: list[Document],
    ) -> list[Document]:
        """

        :param index_processor:
        :param text_docs:
        :return:
        """
        _log.info(
            "Transforming knowledge_id: {} and knowledge_document_id: {}".format(
                self.knowledge_document.knowledge_id,
                self.knowledge_document.knowledge_id,
            )
        )

        # transform the document
        documents = await index_processor.transform(
            self.knowledge_document,
            text_docs,
            doc_language=self.knowledge_document.language,
        )
        return documents

    async def _load_segments(self, documents: List[Document]):
        _log.info(
            "Loading knowledge_id: {} and knowledge_document_id: {}".format(
                self.knowledge_document.knowledge_id, self.knowledge_document.id
            )
        )

        #
        self.knowledge_document_store.add_documents(docs=documents)
        # add document segments
        await self._update_document_index_status(
            after_indexing_status="indexing",
            extra_update_params={
                KnowledgeDocument.cleaning_completed_at: datetime.datetime.now(
                    datetime.timezone.utc
                ).replace(tzinfo=None),
                KnowledgeDocument.splitting_completed_at: datetime.datetime.now(
                    datetime.timezone.utc
                ).replace(tzinfo=None),
            },
        )

        # update segment status to indexing
        self.knowledge_service.update_knowledge_document_segment(
            knowledge_document_id=self.knowledge_document.id,
            update_params={
                KnowledgeDocumentSegment.status: "indexing",
                KnowledgeDocumentSegment.indexing_at: datetime.datetime.now(
                    datetime.timezone.utc
                ).replace(tzinfo=None),
            },
        )
