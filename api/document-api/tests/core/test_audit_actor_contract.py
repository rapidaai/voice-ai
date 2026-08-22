from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock

import pytest

from app.core.indexing_runner import IndexingRunner
from app.core.rag.datasource.vdb.opensearch.opensearch_vector import OpenSearchVector
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument
from app.services.knowledge_service import KnowledgeService


@pytest.mark.parametrize("actor_type", ["user", "project", "service"])
def test_relational_updates_persist_authenticated_actor(actor_type):
    session = MagicMock()
    session.query.return_value.filter.return_value.first.return_value = object()
    session_context = MagicMock()
    session_context.__enter__.return_value = session
    postgres = SimpleNamespace(session=session_context)
    actor = {"type": actor_type, "id": 41}

    KnowledgeService(postgres, actor=actor).update_knowledge_document(
        7, {KnowledgeDocument.index_status: "indexing"}
    )

    updates = session.query.return_value.filter.return_value.update.call_args.args[0]
    assert updates[KnowledgeDocument.updated_actor_type] == actor_type
    assert updates[KnowledgeDocument.updated_actor_id] == 41


@pytest.mark.asyncio
@pytest.mark.parametrize("actor_type", ["user", "project", "service"])
async def test_opensearch_writes_preserve_authenticated_actor(actor_type):
    actor = {"type": actor_type, "id": 41}
    document = Document(
        page_content="audited content",
        metadata={"document_id": "doc-1", "document_hash": "hash-1"},
    )
    runner = object.__new__(IndexingRunner)
    runner.actor = actor
    runner.knowledge = SimpleNamespace(id=11)
    runner.knowledge_document = SimpleNamespace(id=12)
    runner.knowledge_service = MagicMock()
    processor = MagicMock()
    processor.extract = AsyncMock(return_value=[document])

    extracted = await runner._extract(processor)
    assert extracted[0].metadata["created_actor"] == actor
    assert extracted[0].metadata["updated_actor"] == actor

    connection = MagicMock()
    connection.bulk = AsyncMock(return_value={"errors": False, "items": []})
    vector = OpenSearchVector("knowledge", SimpleNamespace(connection=connection))
    await vector.add_texts(extracted, [[0.1, 0.2]])

    actions = connection.bulk.await_args.kwargs["body"]
    assert actions[1]["doc"]["metadata"]["created_actor"] == actor
    assert actions[1]["doc"]["metadata"]["updated_actor"] == actor


def test_relational_update_rejects_missing_actor():
    session = MagicMock()
    session.query.return_value.filter.return_value.first.return_value = object()
    session_context = MagicMock()
    session_context.__enter__.return_value = session
    postgres = SimpleNamespace(session=session_context)

    with pytest.raises(ValueError, match="audit actor is required"):
        KnowledgeService(postgres).update_knowledge_document(
            7, {KnowledgeDocument.index_status: "indexing"}
        )
