"""
Tests for model_manager.py

Covers:
- contents is OrderedDict (not a 1-tuple) — trailing-comma bug fix
- text order preserved in contents
- credential fetched once and cached
- missing credential_id raises exception
- BridgeException propagated from integration layer
- correct provider_name forwarded to get_embedding
"""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from app.core.model_runtime.model_manager import ModelManager
from app.exceptions.bridges_exceptions import BridgeException


def _make_credential(cred_id: int = 42) -> MagicMock:
    """Return a MagicMock VaultCredential.

    `cred.value` is intentionally left as an auto-created MagicMock (truthy).
    Credential(value=MagicMock()) produces an empty proto Struct without error,
    which lets invoke_text_embedding proceed past the nil-value guard.
    Tests that need an explicit null value set cred.value = None directly.
    """
    cred = MagicMock()
    cred.id = cred_id
    cred.name = "test-cred"
    return cred


def _make_manager(
    vault_client,
    integration_client,
    provider: str = "openai",
    params: dict = None,
) -> ModelManager:
    web_issuer = MagicMock()
    web_issuer.issue.return_value = "web-assertion"
    integration_issuer = MagicMock()
    integration_issuer.issue.return_value = "integration-assertion"
    return ModelManager(
        vault_service_client=vault_client,
        integration_service_client=integration_client,
        project_id=100,
        organization_id=200,
        model_provider_name=provider,
        model_parameters=params if params is not None else {"rapida.credential_id": "42", "model": "text-embedding-3-small"},
        references={"knowledge_id": 1, "knowledge_document_id": 2},
        web_service_assertion_issuer=web_issuer,
        integration_service_assertion_issuer=integration_issuer,
    )


def _capture_build_embedding_input(captured: dict):
    """Returns a side_effect that records `contents` and returns a MagicMock.

    We do NOT call the original build_embedding_input because it constructs
    Credential(value=...) where value is google.protobuf.Struct; passing a
    plain Python string from the mock credential would raise AttributeError.
    """
    def _side_effect(**kwargs):
        captured["contents"] = kwargs.get("contents")
        return MagicMock()  # return a stand-in EmbeddingRequest

    return _side_effect


class TestContentsFix:
    """Trailing-comma bug: contents=(OrderedDict(...),) → contents=OrderedDict(...)"""

    @pytest.mark.asyncio
    async def test_contents_is_dict_not_tuple(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        captured = {}

        with patch(
            "app.core.model_runtime.model_manager.build_embedding_input",
            side_effect=_capture_build_embedding_input(captured),
        ):
            await manager.invoke_text_embedding(["hello", "world"])

        contents = captured["contents"]
        assert not isinstance(contents, tuple), (
            "contents must be a dict/OrderedDict. "
            "A trailing comma in model_manager.py would turn it into a 1-tuple."
        )
        assert isinstance(contents, dict)

    @pytest.mark.asyncio
    async def test_contents_maps_indices_to_texts(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        captured = {}

        with patch(
            "app.core.model_runtime.model_manager.build_embedding_input",
            side_effect=_capture_build_embedding_input(captured),
        ):
            await manager.invoke_text_embedding(["hello", "world"])

        assert captured["contents"] == {0: "hello", 1: "world"}

    @pytest.mark.asyncio
    async def test_contents_preserves_text_order(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        texts = ["alpha", "beta", "gamma", "delta"]
        captured = {}

        with patch(
            "app.core.model_runtime.model_manager.build_embedding_input",
            side_effect=_capture_build_embedding_input(captured),
        ):
            await manager.invoke_text_embedding(texts)

        for i, text in enumerate(texts):
            assert captured["contents"][i] == text, f"Index {i} should be '{text}'"

    @pytest.mark.asyncio
    async def test_single_text_contents_is_not_tuple(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        captured = {}

        with patch(
            "app.core.model_runtime.model_manager.build_embedding_input",
            side_effect=_capture_build_embedding_input(captured),
        ):
            await manager.invoke_text_embedding(["only-text"])

        assert not isinstance(captured["contents"], tuple)
        assert captured["contents"] == {0: "only-text"}

    @pytest.mark.asyncio
    async def test_empty_text_list_contents_is_empty_dict(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        captured = {}

        with patch(
            "app.core.model_runtime.model_manager.build_embedding_input",
            side_effect=_capture_build_embedding_input(captured),
        ):
            await manager.invoke_text_embedding([])

        assert not isinstance(captured["contents"], tuple)
        assert captured["contents"] == {}


class TestCredentialFetching:

    @pytest.mark.asyncio
    async def test_credential_fetched_on_first_call(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        await manager.invoke_text_embedding(["text"])

        vault_client.get_credential.assert_called_once()

    @pytest.mark.asyncio
    async def test_outbound_assertions_are_destination_bound(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])
        web_issuer = MagicMock()
        web_issuer.issue.return_value = "web-assertion"
        integration_issuer = MagicMock()
        integration_issuer.issue.return_value = "integration-assertion"
        manager = ModelManager(
            vault_service_client=vault_client,
            integration_service_client=integration_client,
            project_id=100,
            organization_id=200,
            model_provider_name="openai",
            model_parameters={"rapida.credential_id": "42"},
            references={},
            web_service_assertion_issuer=web_issuer,
            integration_service_assertion_issuer=integration_issuer,
        )

        await manager.invoke_text_embedding(["text"])

        web_issuer.issue.assert_called_once()
        integration_issuer.issue.assert_called_once()
        assert "destination" not in web_issuer.issue.call_args.kwargs
        assert "destination" not in integration_issuer.issue.call_args.kwargs
        assert vault_client.get_credential.call_args.kwargs["auth_token"] == "web-assertion"
        assert integration_client.get_embedding.call_args.kwargs["auth_token"] == "integration-assertion"

    @pytest.mark.asyncio
    async def test_credential_cached_across_calls(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client)
        await manager.invoke_text_embedding(["first"])
        await manager.invoke_text_embedding(["second"])
        await manager.invoke_text_embedding(["third"])

        # Vault should only be called once regardless of how many embedding calls
        vault_client.get_credential.assert_called_once()

    @pytest.mark.asyncio
    async def test_missing_credential_id_raises(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()

        manager = _make_manager(
            vault_client, integration_client, params={}  # no rapida.credential_id
        )

        with pytest.raises(Exception, match="credential_id not found"):
            await manager.invoke_text_embedding(["text"])

    @pytest.mark.asyncio
    async def test_empty_credential_value_raises(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()

        cred = _make_credential()
        cred.value = None  # simulate illegal state
        vault_client.get_credential.return_value = cred

        manager = _make_manager(vault_client, integration_client)

        with pytest.raises(Exception):
            await manager.invoke_text_embedding(["text"])


class TestEmbeddingCallForwarding:

    @pytest.mark.asyncio
    async def test_correct_provider_name_forwarded(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client, provider="google")
        await manager.invoke_text_embedding(["text"])

        call_kwargs = integration_client.get_embedding.call_args.kwargs
        assert call_kwargs["provider_name"] == "google"

    @pytest.mark.asyncio
    @pytest.mark.parametrize("provider", ["openai", "cohere", "google", "vertex-ai", "bedrock"])
    async def test_provider_name_passes_through(self, provider):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.return_value = ([], [])

        manager = _make_manager(vault_client, integration_client, provider=provider)
        await manager.invoke_text_embedding(["text"])

        call_kwargs = integration_client.get_embedding.call_args.kwargs
        assert call_kwargs["provider_name"] == provider

    @pytest.mark.asyncio
    async def test_bridge_exception_propagated(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()
        integration_client.get_embedding.side_effect = BridgeException(
            message="embedding failed", bridge_name="integration"
        )

        manager = _make_manager(vault_client, integration_client)

        with pytest.raises(BridgeException):
            await manager.invoke_text_embedding(["text"])

    @pytest.mark.asyncio
    async def test_return_value_is_data_metrics_tuple(self):
        vault_client = AsyncMock()
        integration_client = AsyncMock()
        vault_client.get_credential.return_value = _make_credential()

        mock_embedding = MagicMock()
        mock_metric = MagicMock()
        integration_client.get_embedding.return_value = ([mock_embedding], [mock_metric])

        manager = _make_manager(vault_client, integration_client)
        data, metrics = await manager.invoke_text_embedding(["text"])

        assert data == [mock_embedding]
        assert metrics == [mock_metric]
