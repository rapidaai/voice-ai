# app/tasks.py
import asyncio
import logging
import time
from typing import Dict

from app.bridges.bridge_factory import (
    get_me_integration_client,
    get_me_provider_client,
    get_me_vault_service_client,
)
from app.celery_worker import celery_app
from app.config import get_settings
from app.connectors.connector_factory import get_me_elastic_search, get_me_postgres
from app.core.indexing_runner import IndexingRunner
from app.exceptions.document_exception import DocumentIsPausedException
from app.services.knowledge_service import KnowledgeService
from app.storage.connector_factory import get_me_storage

_log = logging.getLogger("app.tasks.index_knowledge_document")


@celery_app.task
def index_document(data):
    """
    Example Celery task that uses the APP_STORAGE connectors from Celery state.
    """

    return asyncio.run(__index_document(index_document.request, data))


async def __index_document(request, data: Dict):
    try:
        postgres = await get_me_postgres(request)
        elastic_search = await get_me_elastic_search(request)
        storage = await get_me_storage(request)
        integration_client = get_me_integration_client(get_settings())
        vault_client = get_me_vault_service_client(get_settings())
        provider_client = get_me_provider_client(get_settings())

        org_id = data.get("organization_id")
        project_id = data.get("project_id")
        knowledge_id = data.get("knowledge_id")
        knowledge_document_id = data.get("knowledge_document_id")

        if not org_id:
            _log.error(
                f"Failed to start the knowledge dataset runner. org id is not present {org_id}"
            )
            return {
                "status": "error",
                "msg": f"Failed to start the knowledge dataset runner. org id is not present {org_id}",
            }
        if not project_id:
            _log.error(
                f"Failed to start the knowledge dataset runner. project id is not present {project_id}"
            )
            return {
                "status": "error",
                "msg": f"Failed to start the knowledge dataset runner. project id is not present {project_id}",
            }
        if not knowledge_id:
            _log.error(
                f"Failed to start the knowledge dataset runner. knowledge id is not present {knowledge_id}"
            )
            return {
                "status": "error",
                "msg": f"Failed to start the knowledge dataset runner. knowledge id is not present {knowledge_id}",
            }
        if not knowledge_document_id:
            _log.error(
                f"Failed to start the knowledge dataset runner. knowledge_document_id id is not present {knowledge_id}"
            )
            return {
                "status": "error",
                "msg": f"Failed to start the knowledge dataset runner.  {knowledge_document_id}",
            }
        start_at = time.perf_counter()
        knowledge_service = KnowledgeService(postgres)
        document = knowledge_service.get_knowledge_document(
            knowledge_id=knowledge_id,
            knowledge_document_id=knowledge_document_id,
            project_id=project_id,
            organization_id=org_id,
        )
        if not document:
            return {
                "status": "error",
                "msg": "Failed to start the knowledge dataset runner. document not found",
            }
        try:

            await IndexingRunner(
                postgres=postgres,
                elastic_search=elastic_search,
                storage=storage,
                knowledge=knowledge_service.get_knowledge(document.knowledge_id),
                knowledge_document=document,
                integration_client=integration_client,
                provider_client=provider_client,
                vault_client=vault_client,
                index_type="paragraph-index",
            ).run()
            end_at = time.perf_counter()
            _log.info(
                "Processed dataset: {} latency: {}".format(
                    knowledge_id, end_at - start_at
                )
            )
        except DocumentIsPausedException as ex:
            _log.info(str(ex))
            return {"status": "error", "msg": str(ex)}
        except Exception as ex:
            return {"status": "error", "msg": str(ex)}
        return {"processed_data": "done"}
    except Exception as ex:
        return {"status": "error", "msg": str(ex)}
