"""
Copyright (c) 2024 Prashant Srivastav <prashant@rapida.ai>
All rights reserved.

This code is licensed under the MIT License. You may obtain a copy of the License at
https://opensource.org/licenses/MIT.

Unless required by applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions
and limitations under the License.

"""
import logging

from fastapi import APIRouter, Depends, BackgroundTasks
from starlette.requests import Request

from app.bridges.bridge_factory import (
    get_integration_client,
    get_vault_service_client,
)
from app.bridges.internals.integration_bridge import IntegrationBridge
from app.bridges.internals.vault_bridge import VaultBridge
from app.commons.j_response import JResponse
from app.connectors.connector_factory import (
    get_elastic_search,
    get_postgres,
)
from app.connectors.elstic_search_connector import ElasticSearchConnector
from app.connectors.postgres_connector import PostgresConnector
from app.core.indexing_runner import IndexingRunner
from app.exceptions import RapidaException
from app.middlewares.auth.user import User
from app.models.input_model import IndexDocumentRequest
from app.services.knowledge_service import KnowledgeService
from app.storage.connector_factory import get_storage
from app.storage.storage import Storage

# from app.tasks import index_knowledge_document

V1 = APIRouter()
_log = logging.getLogger("app.routers.v1")


@V1.get("/ping/", response_class=JResponse)
async def ping(
        request: Request,
        psql: PostgresConnector = Depends(get_postgres),
):
    """Ping enabled"""
    return JResponse.default_ok(data={"ping": "pong"})


@V1.post("/knowledge/index/document/", response_class=JResponse)
async def index_document(
        request: Request,
        input_request: IndexDocumentRequest,
        background_tasks: BackgroundTasks,
        postgres: PostgresConnector = Depends(get_postgres),
        elastic_search: ElasticSearchConnector = Depends(get_elastic_search),
        storage: Storage = Depends(get_storage),
        integration_client: IntegrationBridge = Depends(get_integration_client),
        vault_client: VaultBridge = Depends(get_vault_service_client),
):
    if not request.auth:
        raise RapidaException(
            status_code=401,
            message="Un-authenticated request",
            error_code=401,
        )

    # _log.debug("storage ==> ", storage)
    authenticated_user: User = request.user
    project_id = authenticated_user.project_id
    organization_id = authenticated_user.organization_id
    knowledge_service = KnowledgeService(postgres)

    #

    for knowledge_document_id in input_request.knowledgeDocumentId:
        document = knowledge_service.get_knowledge_document(
            knowledge_id=input_request.knowledgeId,
            knowledge_document_id=knowledge_document_id,
            project_id=project_id,
            organization_id=organization_id,
        )
        if document is not None:
            runner = IndexingRunner(
                postgres=postgres,
                elastic_search=elastic_search,
                storage=storage,
                knowledge=knowledge_service.get_knowledge(document.knowledge_id),
                knowledge_document=document,
                integration_client=integration_client,
                vault_client=vault_client,
                index_type="paragraph-index",
            )

            background_tasks.add_task(runner.run)
            # await asyncio.create_task(runner.run())
    return JResponse.default_ok(data={"details": "indexing the doucment"}, code=200)


@V1.get("/knowledge/get/{knowledge_document_id}")
async def get_all_knowledge_base(
        knowledge_document_id: int,
        request: Request,
        postgres: PostgresConnector = Depends(get_postgres),
):
    if not request.auth:
        raise RapidaException(
            status_code=401,
            message="Un-authenticated request",
            error_code=401,
        )
    authenticated_user: User = request.user
    user_id = authenticated_user.user_id
    project_id = authenticated_user.project_id
    organization_id = authenticated_user.organization_id

    document = KnowledgeService(postgres).get_knowledge_document(
        knowledge_id=None,
        knowledge_document_id=knowledge_document_id,
        project_id=project_id,
        organization_id=organization_id,
    )

    if document is not None:
        return JResponse.default_ok(
            data={
                "document": document.to_dict(),
                "knowledge_document_id": knowledge_document_id,
                "user_id": user_id,
                "project_id": project_id,
                "organization_id": organization_id,
            }
        )

    return JResponse.default_ok(
        data={"details": "Unable to find knowledge document"}, code=400
    )
