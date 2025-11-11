import logging

from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document

logger = logging.getLogger(__name__)

class UnstructuredIODocumentExtractor(BaseExtractor):
    """Loader that uses unstructured to load word documents."""

    def __init__(
        self,
        file_path: str,
        api_url: str,
        api_key: str
    ):
        """Initialize with file path."""
        self._file_path = file_path
        self._api_url = api_url,
        self.api_key = api_key

    def extract(self) -> list[Document]:
        from unstructured.partition.api import (
        partition_via_api,
    )
        elements = partition_via_api(
            filename=self.file_path,
            api_url = "https://api.unstructuredapp.io/general/v0/general",
            api_key = self.api_key,
        )
    
        documents = []
        for elem in elements:
            text = elem.text.strip()
            documents.append(Document(page_content=text))
        return documents
