# The `BaseIndexProcessor` class defines an abstract interface for extracting, transforming, loading,
# and cleaning documents in a knowledge management system.
"""Abstract interface for document loader implementations."""
from abc import ABC, abstractmethod
from typing import Optional
from app.core.embedding.embedding import Embedder
from app.core.rag.datasource.vdb.vector_base import BaseVector
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument


class BaseIndexProcessor(ABC):

    @abstractmethod
    async def extract(
        self, knowledge_document: KnowledgeDocument, **kwargs
    ) -> list[Document]:
        raise NotImplementedError

    @abstractmethod
    async def transform(
        self, knowledge_document: KnowledgeDocument, documents: list[Document], **kwargs
    ) -> list[Document]:
        raise NotImplementedError

    @abstractmethod
    async def load(
        self,
        knowledge_document: KnowledgeDocument,
        documents: list[Document],
        embedder: Embedder,
        vector_processor: BaseVector,
    ) -> int:
        raise NotImplementedError

    async def clean(
        self, knowledge_document: KnowledgeDocument, node_ids: Optional[list[str]]
    ):
        raise NotImplementedError
