from typing import List
from app.core.rag.extractor.documents.notion_extractor import NotionExtractor
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument


class NotionExtractProcessor:

    def __init__(self):
        pass

    def extract(self, knowledge_document: KnowledgeDocument) -> List[Document]:
        extractor = NotionExtractor(
            notion_workspace_id=knowledge_document.notion_info.notion_workspace_id,
            notion_obj_id=knowledge_document.notion_info.notion_obj_id,
            notion_page_type=knowledge_document.notion_info.notion_page_type,
            document_model=knowledge_document.notion_info.document,
            tenant_id=knowledge_document.notion_info.tenant_id,
        )
        return extractor.extract()
