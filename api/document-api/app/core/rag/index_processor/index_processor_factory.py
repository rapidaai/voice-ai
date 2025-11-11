"""Abstract interface for document loader implementations."""
from app.core.rag.index_processor.constant.index_type import IndexType
from app.core.rag.index_processor.index_processor_base import BaseIndexProcessor
from app.core.rag.index_processor.processor.paragraph_index_processor import ParagraphIndexProcessor
from app.core.rag.index_processor.processor.qa_index_processor import QAIndexProcessor
from app.storage.storage import Storage


# from app.core.rag.index_processor.processor.qa_index_processor import QAIndexProcessor


class IndexProcessorFactory:
    """IndexProcessorInit.
    """

    def __init__(self, index_type: str):
        self._index_type = index_type

    def init_index_processor(self, storage: Storage) -> BaseIndexProcessor:
        """
        This block of code is part of a factory class called `IndexProcessorFactory` in Python. It defines a
        method `init_index_processor` that is responsible for creating and returning an instance of a
        specific type of `BaseIndexProcessor` based on the value of the `_index_type` attribute.
        """

        if not self._index_type:
            raise ValueError("Index type must be specified.")
        if self._index_type == IndexType.PARAGRAPH_INDEX.value:
            return ParagraphIndexProcessor(storage)
        elif self._index_type == IndexType.QA_INDEX.value:
            return QAIndexProcessor()
        else:
            raise ValueError(f"Index type {self._index_type} is not supported.")
