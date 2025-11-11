from typing import Any, List

from app.core.rag.extractor.extractor_base import EntityExtractor
from app.core.rag.models.document import Document
from app.core.transformers.document_transformer import BaseDocumentTransformer


class EntityDocumentTransformer(BaseDocumentTransformer):
    # The `_entity_extractor: BaseExtractor` line in the `EntityDocumentTransformer` class is defining
    # a private attribute `_entity_extractor` of type `BaseExtractor`. This attribute is used to store
    # an instance of a class that implements the `BaseExtractor` interface. In this case, the
    # `SpacyEntityExtractor` class is assigned to `_entity_extractor` in the constructor `__init__()`
    # method. This design allows for flexibility in swapping out different implementations of
    # `BaseExtractor` without changing the rest of the code that depends on this attribute.

    def __init__(self, entity_extractor: EntityExtractor) -> None:
        super().__init__()
        self._entity_extractor = entity_extractor

    def transform_documents(
            self, documents: List[Document], **kwargs: Any
    ) -> List[Document]:
        dc: List[Document] = []
        for d in documents:
            dc.append(self._entity_extractor.extract(d))
        return dc
