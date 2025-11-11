import logging

from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document

logger = logging.getLogger(__name__)


# pip install pi-heif
# pip install openparse
# pip install "openparse[ml]"

class OpenparsePDFExtractor(BaseExtractor):
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
        logger.info(f"Extracting documents from {self._file_path} using OpenparsePDFExtractor")
        import openparse
        parser = openparse.DocumentParser()
        parsed_basic_doc = parser.parse(file=self._file_path)
        documents = []
        for node in parsed_basic_doc.nodes:
            text = node.text.strip()
            documents.append(Document(page_content=text))
        return documents
