from typing import List

from app.configs.extractor_config import ExtractorConfig
from app.core.rag.extractor.processors.confluence import ConfluenceExtractProcessor
from app.core.rag.extractor.processors.google_drive import GoogleDriveExtractProcessor
from app.core.rag.extractor.processors.manual_file import ManualFileExtractProcessor
from app.core.rag.extractor.processors.manual_url import ManualUrlExtractProcessor
from app.core.rag.extractor.processors.notion import NotionExtractProcessor
from app.core.rag.extractor.processors.one_drive import OneDriveExtractProcessor
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument
from app.storage.storage import Storage


class ExtractProcessor:
    # The line `storage: Storage` is defining a type hint for the `storage` attribute in the
    # `ExtractProcessor` class. It indicates that the `storage` attribute should be of type `Storage`,
    # which is a class or type that is expected to be passed as an argument when creating an instance
    # of the `ExtractProcessor` class. This type hint helps in documenting and enforcing the expected
    # type of the attribute, making the code more readable and potentially catching type-related
    # errors during development.
    storage: Storage

    # The line `extractor_config: ExtractorConfig` is defining a class attribute `extractor_config` of
    # type `ExtractorConfig` in the `ExtractProcessor` class. This attribute declaration indicates that
    # instances of the `ExtractProcessor` class will have a property named `extractor_config` that is
    # expected to be of type `ExtractorConfig`.
    extractor_config: ExtractorConfig

    def __init__(self, storage: Storage, extractor_config: ExtractorConfig):
        """
        The `__init__` function initializes an object with a `storage` attribute.

        :param storage: The `__init__` method in your code snippet is a constructor method for a class. It
        takes a parameter named `storage` of type `Storage`. This parameter is used to initialize an
        instance variable `self.storage` within the class
        :type storage: Storage
        """
        self.storage = storage
        self.extractor_config = extractor_config

    def extract(self, knowledge_document: KnowledgeDocument) -> List[Document]:
        # This code snippet is implementing a data extraction process based on the
        # `knowledge_document` object's `source` and `type` attributes. Here's a breakdown of what it
        # does:
        match knowledge_document.source:
            case "manual":
                match knowledge_document.type:
                    case "manual-file":
                        return ManualFileExtractProcessor(
                            self.storage, self.extractor_config
                        ).extract(knowledge_document)
                    case "manual-url":
                        return ManualUrlExtractProcessor().extract(
                            knowledge_document
                        )

            case "notion":
                return NotionExtractProcessor().extract(knowledge_document)
            case "google-drive":
                return GoogleDriveExtractProcessor().extract(knowledge_document)
            case "one-drive":
                return OneDriveExtractProcessor().extract(knowledge_document)
            case "confluence":
                return ConfluenceExtractProcessor().extract(knowledge_document)
            case _:
                raise ValueError(
                    f"Unsupported datasource type: {knowledge_document.document_source}"
                )
