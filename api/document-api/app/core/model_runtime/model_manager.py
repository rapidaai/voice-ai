"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""
import logging
from typing import Dict, List, Optional, OrderedDict, Tuple

from app.authentication import ServiceAssertionIssuer, Tenant
from app.bridges.artifacts.protos.common_pb2 import Metric
from app.bridges.artifacts.protos.integration_api_pb2 import Credential, Embedding
from app.bridges.artifacts.protos.vault_api_pb2 import VaultCredential
from app.bridges.internals.integration_bridge import IntegrationBridge
from app.bridges.internals.vault_bridge import VaultBridge
from app.core.callers.input_builder import build_embedding_input
from app.core.errors.error import (
    ProviderTokenNotInitError,
    ModelCurrentlyNotSupportError,
)
from app.exceptions import RapidaException
from app.exceptions.bridges_exceptions import BridgeException

logger = logging.getLogger(__name__)


class ModelManager:
    # The lines `provider_service_client: ProviderBridge`, `vault_service_client: VaultBridge`, and
    # `integration_service_client: IntegrationBridge` are defining class attributes in the
    # `ModelManagerFactory` class. These attributes are instances of classes `ProviderBridge`,
    # `VaultBridge`, and `IntegrationBridge` respectively.
    __vault_service_client: VaultBridge
    __integration_service_client: IntegrationBridge
    __references: Dict

    # The `credential: Optional[VaultCredential]` line in the `ModelManager` class is defining a class
    # attribute named `credential` that can hold an optional value of type `VaultCredential`.
    credential: Optional[VaultCredential] = None

    # The `project_id: int` parameter in the `ModelManager` class is used to store the project ID
    # associated with the model manager instance. This parameter is passed to various methods within
    # the class to perform operations specific to that project. It is used in methods like
    # `_fetch_credential` and `_get_model` to retrieve credentials and provider models specific to the
    # project identified by `project_id`. This helps in ensuring that the operations performed by the
    # `ModelManager` are scoped to the correct project context.
    project_id: int
    # The `organization_id: int` parameter in the `ModelManager` class is used to store the
    # organization ID associated with the model manager instance. This parameter is passed to various
    # methods within the class to perform operations specific to that organization.
    organization_id: int

    # model specific parameters
    model_provider_name: str
    model_parameters: Optional[Dict] = {}


    def __init__(
        self,
        vault_service_client: VaultBridge,
        integration_service_client: IntegrationBridge,
        project_id: int,
        organization_id: int,
        #
        model_provider_name: str,
        model_parameters: Optional[Dict] = {},
        references: Optional[Dict] = {},
        web_service_assertion_issuer: ServiceAssertionIssuer = None,
        integration_service_assertion_issuer: ServiceAssertionIssuer = None,
    ):
        """
        initialize the model manager with required parameters that will help do things
        """
        self.__vault_service_client = vault_service_client
        self.__integration_service_client = integration_service_client
        self.__references = references
        self.project_id = project_id
        self.organization_id = organization_id
        self.model_provider_name = model_provider_name
        self.model_parameters = model_parameters
        if web_service_assertion_issuer is None or integration_service_assertion_issuer is None:
            raise ValueError("service assertion issuers are required")
        self.web_service_assertion_issuer = web_service_assertion_issuer
        self.integration_service_assertion_issuer = integration_service_assertion_issuer

    async def _fetch_credential(self, credential_id: int) -> VaultCredential:
        """

        Args:
            credential_id:

        Returns:

        """
        try:
            return await self.__vault_service_client.get_credential(
                auth_token=self.web_service_assertion_issuer.issue(
                    tenant=Tenant(
                        organizationId=self.organization_id,
                        projectId=self.project_id,
                    ),
                ),
                crendential_id=int(credential_id),
            )
        except RapidaException:
            raise ProviderTokenNotInitError("Unable to get credentials for provider.")

    async def invoke_text_embedding(
        self, texts: List[str]
    ) -> Tuple[List[Embedding], List[Metric]]:
        """
        Invoke large language model

        :param texts: texts to embed
        :return: embeddings result
        """
        try:
            if not self.credential:

                credential_id = self.model_parameters.get("rapida.credential_id")
                if not credential_id:
                    raise Exception("credential_id not found in options")

                self.credential = await self._fetch_credential(credential_id)
                if not self.credential.value:
                    raise Exception("illegal state of credentials")

                self.__references.update(
                    {
                        "vault_name": self.credential.name,
                        "vault_id": self.credential.id,
                    }
                )

            credential = self.credential.value
            request = build_embedding_input(
                credential=Credential(id=self.credential.id, value=credential),
                parameters=self.model_parameters,
                contents=OrderedDict((i, s) for i, s in enumerate(texts)),
                additional_data=self.__references,
            )

            return await self.__integration_service_client.get_embedding(
                auth_token=self.integration_service_assertion_issuer.issue(
                    tenant=Tenant(
                        organizationId=self.organization_id,
                        projectId=self.project_id,
                    ),
                ),
                provider_name=self.model_provider_name,
                request=request,
            )
        except BridgeException as ex:
            raise ex
