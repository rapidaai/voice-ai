# The `KnowledgeService` class in the provided Python code contains methods for updating and
# retrieving knowledge documents from a database using SQLAlchemy.
import datetime
import logging
from typing import List, Optional, Union

from sqlalchemy import desc

from app.connectors.postgres_connector import PostgresConnector
from app.exceptions.document_exception import (
    DocumentNotFoundException,
)
from app.models.knowledge_model import (
    KnowledgeDocument,
    Knowledge,
    KnowledgeDocumentSegment, KnowledgeEmbeddingModelOption,
)

_log = logging.getLogger("app.services.knowledge_service")


class KnowledgeService:
    # The `db: Session` in the `KnowledgeService` class is a type hint that specifies the type of the
    # `db` attribute. In this case, it indicates that the `db` attribute is expected to be an instance
    # of the `Session` class from SQLAlchemy's ORM (Object-Relational Mapping) module. This type hint
    # helps improve code readability and provides information to developers about the expected type of
    # the attribute.
    postgres: PostgresConnector

    def __init__(self, postgres: PostgresConnector):
        # The line `session = db` in the `KnowledgeService` class constructor is initializing the `db`
        # attribute of the class with the value passed to the constructor.
        self.postgres = postgres

    def update_knowledge_document(
            self, knowledge_document_id: int, extra_update_params: Optional[dict] = None
    ) -> None:
        # This code snippet is from the `update_knowledge_document` method in the `KnowledgeService`
        # class. Here's a breakdown of what it does:
        with self.postgres.session as session:
            _log.debug(
                f"updating knowledge document with {knowledge_document_id} and params {extra_update_params}"
            )
            document = (
                session.query(KnowledgeDocument)
                .filter(KnowledgeDocument.id == knowledge_document_id)
                .first()
            )

            if not document:
                raise DocumentNotFoundException(status_code=500)
            session.query(KnowledgeDocument).filter(
                KnowledgeDocument.id == knowledge_document_id
            ).update(extra_update_params)
            session.commit()

    def get_knowledge_document(
            self,
            knowledge_id: Union[int, None],
            knowledge_document_id: int,
            project_id: int,
            organization_id: int,
    ) -> Union[KnowledgeDocument, None]:
        # This code snippet is from the `get_knowledge_document` method in the `KnowledgeService`
        # class. Here's a breakdown of what it does:
        with self.postgres.session as session:
            qry = session.query(KnowledgeDocument).filter(
                KnowledgeDocument.id == knowledge_document_id,
                KnowledgeDocument.project_id == project_id,
                KnowledgeDocument.organization_id == organization_id,
            )

            if knowledge_id is not None:
                qry = qry.filter(KnowledgeDocument.knowledge_id == knowledge_id)

            return (
                qry.outerjoin(Knowledge)
                .order_by(desc(KnowledgeDocument.created_date))
                .first()
            )

    def complete_knowledge_document_segment(
            self, knowledge_document_id: int, document_ids: List[str]
    ):
        with self.postgres.session as session:
            session.query(KnowledgeDocumentSegment).filter(
                KnowledgeDocumentSegment.knowledge_document_id == knowledge_document_id,
                KnowledgeDocumentSegment.index_node_id.in_(document_ids),
                KnowledgeDocumentSegment.status == "indexing",
            ).update(
                {
                    KnowledgeDocumentSegment.status: "completed",
                    KnowledgeDocumentSegment.enabled: True,
                    KnowledgeDocumentSegment.completed_at: datetime.datetime.now(
                        datetime.timezone.utc
                    ).replace(tzinfo=None),
                }
            )
            session.commit()

    def update_knowledge_document_segment(
            self, knowledge_document_id: int, update_params: dict
    ) -> None:
        """
        The function `update_knowledge_document_segment` updates segments in a knowledge document based on the
        provided parameters.

        :param knowledge_document_id: The `knowledge_document_id` parameter is an integer that represents
        the unique identifier of a knowledge document. This identifier is used to filter the
        KnowledgeDocumentSegment records that are associated with the specified knowledge document for
        updating
        :type knowledge_document_id: int
        :param update_params: The `update_params` parameter in the `_update_segments_by_document` method is
        a dictionary that contains the key-value pairs of the fields and their updated values that need to
        be updated in the `KnowledgeDocumentSegment` table for a specific `knowledge_document_id`
        :type update_params: dict
        """

        with self.postgres.session as session:
            session.query(KnowledgeDocumentSegment).filter(
                KnowledgeDocumentSegment.knowledge_document_id == knowledge_document_id
            ).update(update_params)
            session.commit()

    def get_knowledge(self, knowledge_id: str) -> Knowledge:
        with self.postgres.session as session:
            return session.query(Knowledge).filter(Knowledge.id == knowledge_id).first()

    def get_knowledge_model_options(self, knowledge_id) -> dict[str, str]:
        with self.postgres.session as session:
            options = session.query(KnowledgeEmbeddingModelOption).filter_by(knowledge_id=knowledge_id).all()
            return {opt.key: opt.value for opt in options}
