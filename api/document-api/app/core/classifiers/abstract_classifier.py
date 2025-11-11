from abc import ABC, abstractmethod
from enum import Enum
from typing import List

from app.core.rag.models.document import Document


class Domain(str, Enum):
    FINANCIAL = "financial"
    TECHNICAL = "technical"
    MEDICAL = "medical"
    UNKNOWN = "unknown"


class AbstractClassifier(ABC):

    @abstractmethod
    def classify(self, docs: List[Document]) -> Domain:
        raise NotImplementedError
