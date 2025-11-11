import logging

from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document

logger = logging.getLogger(__name__)


# pip install pi-heif
# pip install unstructured-inference
# pip install unstructured.pytesseract

class UnstructuredPDFExtractor(BaseExtractor):
    """Load md files.


    Args:
        file_path: Path to the file to load.

        remove_hyperlinks: Whether to remove hyperlinks from the text.

        remove_images: Whether to remove images from the text.

        encoding: File encoding to use. If `None`, the file will be loaded
        with the default system encoding.

        autodetect_encoding: Whether to try to autodetect the file encoding
            if the specified encoding fails.
    """

    def __init__(
            self,
            file_path: str,
    ):
        """Initialize with file path."""
        self._file_path = file_path

    def extract(self) -> list[Document]:
        from unstructured.partition.pdf import partition_pdf
        elements = partition_pdf(filename=self._file_path,
                                 chunking_strategy="by_title",  # mandatory to use ``lattice`` strategy
                                 strategy="hi_res",  # mandatory to use ``hi_res`` strategy)
                                 )

        documents = []
        for element in elements:
            documents.append(Document(page_content=element.text))

        return documents
        # metadata = {"source": blob.source, "page": page_number}
        # yield Document(page_content=content, metadata=metadata)
