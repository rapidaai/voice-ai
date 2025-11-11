from collections.abc import Sequence
import logging
from typing import Any
from sqlalchemy import func
from app.connectors.postgres_connector import PostgresConnector
from app.core.docstore.knowledge_document_store import KnowledgeDocumentStore
from app.core.rag.datasource import vdb
from app.core.rag.models.document import Document
from app.models.knowledge_model import (
    Knowledge,
    KnowledgeDocument,
    KnowledgeDocumentSegment,
)
from app.utils.general import count_string_token, count_string_word


# from extensions.ext_database import db
# from models.knowledge import KnowledgeDocument, KnowledgeDocumentSegment, Knowledge

logger = logging.getLogger(__name__)


class PostgresKnowledgeDocumentStore(KnowledgeDocumentStore):
    # The line `postgres: PostgresConnector` is defining a class attribute `postgres` of type
    # `PostgresConnector`. This attribute is being declared at the class level, outside of any method, and
    # is not initialized in the `__init__` method. This means that all instances of the
    # `KnowledgeDocumentStore` class will have access to this attribute, which can be used to interact
    # with a PostgreSQL database using the `PostgresConnector` class.

    postgres: PostgresConnector

    def __init__(
        self,
        postgres: PostgresConnector,
        knowledge: Knowledge,
        knowledge_document: KnowledgeDocument,
    ):
        # The lines `self._knowledge = knowledge`, `self.knowledge_document = knowledge_document`,
        # and `self._knowledge_id = knowledge_id` in the `__init__` method of the
        # `KnowledgeDocumentStore` class are initializing instance variables with the values passed to
        # the constructor when creating an instance of the class.
        super().__init__(knowledge=knowledge, knowledge_document=knowledge_document)
        self.postgres = postgres

    @classmethod
    def from_dict(cls, config_dict: dict[str, Any]) -> "KnowledgeDocumentStore":
        return cls(**config_dict)

    def to_dict(self) -> dict[str, Any]:
        """Serialize to dict."""
        return {
            vdb.constants.Field.KNOWLEDGE_DOCUMENT_ID_KEY.value: self.knowledge_document.id,
        }

    @property
    def docs(self) -> dict[str, Document]:
        with self.postgres.session as session:
            document_segments = (
                session.query(KnowledgeDocumentSegment)
                .filter(
                    KnowledgeDocumentSegment.knowledge_document_id
                    == self.knowledge_document.id
                )
                .all()
            )
            output = {}
            for document_segment in document_segments:
                doc_id = document_segment.index_node_id
                output[doc_id] = Document(
                    page_content=document_segment.content,
                    metadata={
                        vdb.constants.Field.DOCUMENT_ID_KEY.value: document_segment.index_node_id,
                        vdb.constants.Field.DOCUMENT_HASH_KEY.value: document_segment.index_node_hash,
                        vdb.constants.Field.KNOWLEDGE_ID_KEY.value: document_segment.knowledge_id,
                        vdb.constants.Field.KNOWLEDGE_DOCUMENT_ID_KEY.value: document_segment.knowledge_document_id,
                    },
                )

            return output

    def _clean_text(self, text: str) -> str:
        """Remove NULL characters from text."""
        return text.replace('\x00', '')


    def add_documents(
        self, docs: Sequence[Document], allow_update: bool = True
    ) -> None:
        with self.postgres.session as session:
            max_position = (
                session.query(func.max(KnowledgeDocumentSegment.position))
                .filter(
                    KnowledgeDocumentSegment.knowledge_document_id
                    == self.knowledge_document.id
                )
                .scalar()
            )

            # tokens = 0
            if max_position is None:
                max_position = 0

            for doc in docs:
                if not isinstance(doc, Document):
                    raise ValueError("doc must be a Document")

                 # Clean the text content
                doc.page_content = self._clean_text(doc.page_content)

                segment_document = self.get_document_segment(doc_id=doc.document_id)
                # NOTE: doc could already exist in the store, but we overwrite it
                if not allow_update and segment_document:
                    logger.warning(
                        f"doc_id {doc.document_id} already exists. "
                        "Set allow_update to True to overwrite."
                    )

                if not segment_document:
                    max_position += 1
                    segment_document = KnowledgeDocumentSegment()
                    segment_document.knowledge_id = self.knowledge_id
                    segment_document.knowledge_document_id = self.knowledge_document.id
                    segment_document.word_count = count_string_word(doc.page_content)
                    segment_document.index_node_id = doc.document_id
                    segment_document.index_node_hash = doc.document_hash
                    segment_document.position = max_position
                    segment_document.content = doc.page_content
                    segment_document.token_count = count_string_token(doc.page_content)
                    segment_document.enabled = False
                    segment_document.created_by = self.knowledge_document.created_by
                    if doc.metadata.get("answer"):
                        segment_document.answer = doc.metadata.pop("answer", "")

                    session.add(segment_document)
                else:
                    segment_document.content = doc.page_content
                    if doc.metadata.get("answer"):
                        segment_document.answer = doc.metadata.pop("answer", "")
                    segment_document.index_node_hash = doc.document_hash
                    segment_document.word_count = count_string_word(doc.page_content)
                    segment_document.tokens = count_string_token(doc.page_content)

                session.commit()

    def get_document_segment(self, doc_id: str) -> KnowledgeDocumentSegment:
        with self.postgres.session as session:
            document_segment = (
                session.query(KnowledgeDocumentSegment)
                .filter(
                    KnowledgeDocumentSegment.knowledge_id
                    == self.knowledge_document.knowledge_id,
                    KnowledgeDocumentSegment.index_node_id == doc_id,
                )
                .first()
            )

            return document_segment
