import logging

from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document

logger = logging.getLogger(__name__)


class UnstructuredHTMLExtractor(BaseExtractor):
    """Load md files.
    """

    def __init__(
        self,
        file_path: str,
    ):
        """Initialize with file path."""
        self._file_path = file_path

    def extract(self) -> list[Document]:
        from unstructured.partition.html import partition_html

        elements = partition_html(filename=self._file_path)
        from unstructured.chunking.title import chunk_by_title
        chunks = chunk_by_title(elements, max_characters=2000, combine_text_under_n_chars=2000)
        documents = []
        for chunk in chunks:
            text = chunk.text.strip()
            documents.append(Document(page_content=text))

        return documents
