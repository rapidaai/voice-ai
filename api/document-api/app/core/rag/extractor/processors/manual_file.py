import tempfile
from pathlib import Path
from typing import List

from app.configs.extractor_config import ExtractorConfig
from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document
from app.exceptions.pipeline_exception import IllegalDocumentExtractorConfigException
from app.models.knowledge_model import KnowledgeDocument
from app.storage.storage import Storage
from app.utils.general import dynamic_class_import


class ManualFileExtractProcessor:
    # The line `storage: Storage` is defining a type hint for the `storage` attribute in the
    # `ExtractProcessor` class. It indicates that the `storage` attribute should be of type `Storage`,
    # which is a class or type that is expected to be passed as an argument when creating an instance
    # of the `ExtractProcessor` class. This type hint helps in documenting and enforcing the expected
    # type of the attribute, making the code more readable and potentially catching type-related
    # errors during development.
    storage: Storage

    # The line `config: ExtractorConfig` in the code snippet is defining a type hint for the `config`
    # attribute in the `ManualFileExtractProcessor` class. It indicates that the `config` attribute
    # should be of type `ExtractorConfig`, which is a class or type that is expected to be passed as
    # an argument when creating an instance of the `ManualFileExtractProcessor` class.
    config: ExtractorConfig

    """
    The `__init__` function initializes an object with a `storage` attribute.
    :param storage: The `__init__` method in your code snippet is a constructor method for a class. It
    takes a parameter named `storage` of type `Storage`. This parameter is used to initialize an
    instance variable `self.storage` within the class
    :type storage: Storage
    """

    def __init__(self, storage: Storage, config: ExtractorConfig):
        self.storage = storage
        self.config = config

    def extract(self, knowledge_document: KnowledgeDocument) -> List[Document]:
        temp_dir = tempfile.TemporaryDirectory()
        try:
            file_extension = Path(knowledge_document.document_url).suffix.lower()
            file_path = f"{temp_dir.name}/{next(tempfile._get_candidate_names())}{file_extension}"

            self.storage.download(knowledge_document.document_url, file_path)

            extractor = next(
                (entry for entry in self.config.file_extensions if entry.extension == file_extension),
                next((entry for entry in self.config.file_extensions if entry.extension == "*"), None),
            )
            if not extractor:
                raise IllegalDocumentExtractorConfigException()

            extractor_class: BaseExtractor = dynamic_class_import(extractor.extractor)
            options = extractor.options.model_dump() if extractor.options else {}

            doc: List[Document] = extractor_class(file_path, **options).extract()
            return doc
        finally:
            temp_dir.cleanup()
