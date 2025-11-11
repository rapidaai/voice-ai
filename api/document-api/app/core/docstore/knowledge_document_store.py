from abc import ABC, abstractmethod
from typing import Sequence

from app.core.rag.models.document import Document
from app.models.knowledge_model import (
    Knowledge,
    KnowledgeDocument,
    KnowledgeDocumentSegment,
)


class KnowledgeDocumentStore(ABC):

    # In the code snippet provided, the lines `knowledge: Knowledge` and `knowledge_document:
    # KnowledgeDocument` are class attributes that define the types of the `knowledge` and
    # `knowledge_document` variables in the `KnowledgeDocumentStore` class.
    knowledge: Knowledge
    knowledge_document: KnowledgeDocument

    def __init__(
        self, knowledge: Knowledge, knowledge_document: KnowledgeDocument
    ) -> None:
        super().__init__()
        self.knowledge = knowledge
        self.knowledge_document = knowledge_document

    @property
    def knowledge_id(self) -> int:
        return self.knowledge.id

    @abstractmethod
    def add_documents(
        self, docs: Sequence[Document], allow_update: bool = True
    ) -> None:
        raise NotImplementedError("Subclasses must implement this method")

    @property
    def docs(self) -> dict[str, Document]:
        raise NotImplementedError("Subclasses must implement this method")

    @abstractmethod
    def get_document_segment(self, doc_id: str) -> KnowledgeDocumentSegment:
        raise NotImplementedError("Subclasses must implement this method")
