from abc import ABC, abstractmethod
from typing import List, Tuple


class Embedder(ABC):
    """Interface for embedding models."""

    @abstractmethod
    async def embed_documents(self, texts: List[str]) -> Tuple[List[List[float]], int]:
        """Embed search docs."""
        raise NotImplementedError
