from typing import List
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument


class ManualUrlExtractProcessor:

    def __init__(self):
        pass

    def extract(self, knowledge_document: KnowledgeDocument) -> List[Document]:
        raise ValueError(
            f"Unsupported datasource type: {knowledge_document.document_source}"
        )
