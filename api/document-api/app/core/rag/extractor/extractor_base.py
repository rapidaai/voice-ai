"""Abstract interface for document loader implementations."""

from abc import ABC, abstractmethod
from typing import List

from app.core.rag.models.document import Document


class BaseExtractor(ABC):
    """Interface for extract files."""

    @abstractmethod
    def extract(self) -> List[Document]:
        raise NotImplementedError


class EntityExtractor(ABC):
    """Interface for extract files."""

    @abstractmethod
    def extract(self, document: Document) -> Document:
        raise NotImplementedError
