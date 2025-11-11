from abc import ABC, abstractmethod
import logging
from app.bridges.artifacts.protos.knowledge_api_pb2 import KnowledgeDocument
from app.core.embedding.embedding import Embedder
from app.core.rag.datasource.vdb.vector_base import BaseVector
from app.core.rag.models.document import Document


class AbstractVectorFactory(ABC):
    @abstractmethod
    def init_vector(self, collection_name: str) -> BaseVector:
        raise NotImplementedError


_log = logging.getLogger(__name__)


class Vector:
    _knowledge_document: KnowledgeDocument
    _embeddings: Embedder
    _vector_processor: BaseVector

    def __init__(
        self,
        knowledge_document: KnowledgeDocument,
        embedder: Embedder,
        vector_processor: BaseVector,
    ):
        self._knowledge_document = knowledge_document
        self._embeddings = embedder
        self._vector_processor = vector_processor

    async def create(self, texts: list = None, **kwargs) -> int:
        if texts:
            embeddings, token = await self._embeddings.embed_documents(
                [document.page_content for document in texts]
            )

            await self._vector_processor.create(
                texts=texts, embeddings=embeddings, **kwargs
            )
            return token
        return 0

    async def add_texts(self, documents: list[Document], **kwargs) -> int:
        # if kwargs.get("duplicate_check", True):
        #     documents = self._filter_duplicate_texts(documents)
        embeddings, token = await self._embeddings.embed_documents(
            [document.page_content for document in documents]
        )
        await self._vector_processor.create(
            texts=documents, embeddings=embeddings, **kwargs
        )
        return token

    async def text_exists(self, id: str) -> bool:
        return await self._vector_processor.text_exists(id)

    async def _filter_duplicate_texts(self, texts: list[Document]) -> list[Document]:
        for text in texts:
            doc_id = text.document_hash
            exists_duplicate_node = self.text_exists(doc_id)
            if exists_duplicate_node:
                texts.remove(text)
        return texts
