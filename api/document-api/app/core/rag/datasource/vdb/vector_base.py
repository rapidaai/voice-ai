from __future__ import annotations

from abc import ABC, abstractmethod

from app.core.rag.models.document import Document


class BaseVector(ABC):

    def __init__(self, collection_name: str):
        self._collection_name = collection_name

    @abstractmethod
    async def create(
        self, texts: list[Document], embeddings: list[list[float]], **kwargs
    ):
        raise NotImplementedError

    @abstractmethod
    async def add_texts(
        self, documents: list[Document], embeddings: list[list[float]], **kwargs
    ):
        """
        The `add_texts` function is a placeholder method that raises a `NotImplementedError`.
        :param documents: A list of `Document` objects that contain text data to be processed
        :type documents: list[Document]
        :param embeddings: The `embeddings` parameter in the `add_texts` method is expected to be a list of
        lists of floats. Each inner list represents the embedding for a document in the `documents`
        parameter. The embeddings are used to associate numerical representations with the textual content
        of the documents for various natural language processing
        :type embeddings: list[list[float]]
        """
        raise NotImplementedError

    @abstractmethod
    async def text_exists(self, id: str) -> bool:
        """
        The function `text_exists` is an asynchronous method that checks if a text with a given ID exists
        and returns a boolean value.
        :param id: The `id` parameter in the `text_exists` method is a string type
        :type id: str
        """
        raise NotImplementedError
