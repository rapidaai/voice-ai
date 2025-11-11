from typing import Optional
from pydantic import BaseModel, Field
import app.core.rag.datasource.vdb.constants as VDBConstant


class Document(BaseModel):
    """Class for storing a piece of text and associated metadata."""

    page_content: str

    """Arbitrary metadata about the page content (e.g., source, relationships to other
        documents, etc.).
    """
    metadata: Optional[dict] = Field(default_factory=dict)

    # The line `entity: Optional[dict] = Field(default_factory=dict)` in the `Document` class is
    # defining a field named `entity` that can hold a dictionary as its value. Here's what each part
    # of this line does:
    entities: Optional[dict] = Field(default_factory=dict)

    @property
    def document_id(self):
        return self.metadata[VDBConstant.Field.DOCUMENT_ID_KEY.value]

    @property
    def document_hash(self):
        return self.metadata[VDBConstant.Field.DOCUMENT_HASH_KEY.value]

    @property
    def knowledge_id(self):
        return self.metadata[VDBConstant.Field.KNOWLEDGE_ID_KEY.value]
