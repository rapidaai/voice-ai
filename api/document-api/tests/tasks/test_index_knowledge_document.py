"""
Tests for index_knowledge_document.py Celery task

Covers:
- P0: get_me_provider_client is NOT imported (would cause ImportError on worker start)
- P0: bridge clients receive URL strings, not ApplicationSettings objects
- P0: IndexingRunner constructed without provider_client kwarg
- Task returns error dict on missing required fields
- Task returns error dict when document not found
"""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

import app.tasks.index_knowledge_document as task_module
import app.bridges.bridge_factory as factory_module


def _get_index_fn():
    """Retrieve the private __index_document function from the module."""
    return task_module.__dict__["__index_document"]


def _make_settings(integration_host="integration:9004", web_host="web:9001"):
    settings = MagicMock()
    settings.internal_service.integration_host = integration_host
    settings.internal_service.web_host = web_host
    return settings


def _base_data(**overrides):
    data = {
        "organization_id": "org-1",
        "project_id": "proj-2",
        "knowledge_id": "know-3",
        "knowledge_document_id": "doc-4",
        "actor_type": "service",
        "actor_id": 41,
    }
    data.update(overrides)
    return data


class TestImports:
    """P0: broken import must be removed so the Celery worker can start."""

    def test_get_me_provider_client_not_in_task_module(self):
        assert not hasattr(task_module, "get_me_provider_client"), (
            "get_me_provider_client does not exist in bridge_factory; "
            "importing it would crash the Celery worker at startup"
        )

    def test_get_me_integration_client_imported(self):
        assert hasattr(task_module, "get_me_integration_client")

    def test_get_me_vault_service_client_imported(self):
        assert hasattr(task_module, "get_me_vault_service_client")

    def test_index_document_celery_task_exists(self):
        assert hasattr(task_module, "index_document")


class TestBridgeClientInitialization:
    """P0: bridge clients must be initialized with URL strings, not settings objects."""

    @pytest.mark.asyncio
    async def test_integration_client_receives_url_string(self):
        settings = _make_settings(integration_host="int-host:9004")
        captured = {}

        def fake_integration(url):
            captured["integration_url"] = url
            return MagicMock()

        def fake_vault(url):
            captured["vault_url"] = url
            return MagicMock()

        mock_ks = MagicMock()
        mock_ks.return_value.get_knowledge_document.return_value = None

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", side_effect=fake_integration),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", side_effect=fake_vault),
            patch("app.tasks.index_knowledge_document.KnowledgeService", mock_ks),
        ):
            await _get_index_fn()(MagicMock(), _base_data())

        assert isinstance(captured["integration_url"], str), (
            f"get_me_integration_client expected str, got {type(captured['integration_url'])}"
        )
        assert captured["integration_url"] == "int-host:9004"

    @pytest.mark.asyncio
    async def test_vault_client_receives_url_string(self):
        settings = _make_settings(web_host="web-host:9001")
        captured = {}

        def fake_vault(url):
            captured["vault_url"] = url
            return MagicMock()

        mock_ks = MagicMock()
        mock_ks.return_value.get_knowledge_document.return_value = None

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", side_effect=fake_vault),
            patch("app.tasks.index_knowledge_document.KnowledgeService", mock_ks),
        ):
            await _get_index_fn()(MagicMock(), _base_data())

        assert isinstance(captured["vault_url"], str)
        assert captured["vault_url"] == "web-host:9001"

    @pytest.mark.asyncio
    async def test_integration_client_not_called_with_settings_object(self):
        """Regression: passing the full settings object caused a gRPC TypeError."""
        settings = MagicMock()
        settings.internal_service.integration_host = "localhost:9004"
        settings.internal_service.web_host = "localhost:9001"

        integration_args = []

        def capture_integration(arg):
            integration_args.append(arg)
            return MagicMock()

        mock_ks = MagicMock()
        mock_ks.return_value.get_knowledge_document.return_value = None

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", side_effect=capture_integration),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.KnowledgeService", mock_ks),
        ):
            await _get_index_fn()(MagicMock(), _base_data())

        assert len(integration_args) == 1
        assert isinstance(integration_args[0], str), (
            "get_me_integration_client must receive a string URL, "
            f"got {type(integration_args[0])}"
        )
        assert not isinstance(integration_args[0], MagicMock)


class TestIndexingRunnerConstruction:
    """P0: IndexingRunner must not receive provider_client kwarg."""

    @pytest.mark.asyncio
    async def test_indexing_runner_no_provider_client_kwarg(self):
        settings = _make_settings()
        runner_kwargs = {}

        mock_doc = MagicMock()
        mock_knowledge = MagicMock()
        mock_ks_instance = MagicMock()
        mock_ks_instance.get_knowledge_document.return_value = mock_doc
        mock_ks_instance.get_knowledge.return_value = mock_knowledge

        mock_runner = AsyncMock()

        def capture_runner(**kwargs):
            runner_kwargs.update(kwargs)
            return mock_runner

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.KnowledgeService", return_value=mock_ks_instance),
            patch("app.tasks.index_knowledge_document.IndexingRunner", side_effect=capture_runner),
        ):
            await _get_index_fn()(MagicMock(), _base_data())

        assert "provider_client" not in runner_kwargs, (
            "IndexingRunner was called with provider_client kwarg, "
            "but that parameter does not exist in its __init__"
        )

    @pytest.mark.asyncio
    async def test_indexing_runner_has_integration_and_vault_clients(self):
        settings = _make_settings()
        runner_kwargs = {}

        mock_ks_instance = MagicMock()
        mock_ks_instance.get_knowledge_document.return_value = MagicMock()
        mock_ks_instance.get_knowledge.return_value = MagicMock()

        mock_runner = AsyncMock()

        def capture_runner(**kwargs):
            runner_kwargs.update(kwargs)
            return mock_runner

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.KnowledgeService", return_value=mock_ks_instance),
            patch("app.tasks.index_knowledge_document.IndexingRunner", side_effect=capture_runner),
        ):
            await _get_index_fn()(MagicMock(), _base_data())

        assert "integration_client" in runner_kwargs
        assert "vault_client" in runner_kwargs
        assert runner_kwargs["actor"] == {"type": "service", "id": 41}


class TestTaskValidation:
    """Task must return error dicts for invalid input instead of crashing."""

    @pytest.mark.asyncio
    async def test_missing_organization_id_returns_error(self):
        settings = _make_settings()
        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
        ):
            result = await _get_index_fn()(MagicMock(), {"project_id": "1"})

        assert result["status"] == "error"
        assert "org" in result["msg"].lower() or "organization" in result["msg"].lower()

    @pytest.mark.asyncio
    async def test_invalid_actor_returns_error(self):
        settings = _make_settings()
        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
        ):
            result = await _get_index_fn()(MagicMock(), _base_data(actor_id=0))

        assert result["status"] == "error"
        assert "actor" in result["msg"].lower()

    @pytest.mark.asyncio
    async def test_missing_project_id_returns_error(self):
        settings = _make_settings()
        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
        ):
            result = await _get_index_fn()(MagicMock(), {"organization_id": "1"})

        assert result["status"] == "error"

    @pytest.mark.asyncio
    async def test_missing_knowledge_id_returns_error(self):
        settings = _make_settings()
        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
        ):
            result = await _get_index_fn()(
                MagicMock(), {"organization_id": "1", "project_id": "2"}
            )

        assert result["status"] == "error"

    @pytest.mark.asyncio
    async def test_document_not_found_returns_error(self):
        settings = _make_settings()
        mock_ks = MagicMock()
        mock_ks.return_value.get_knowledge_document.return_value = None

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.KnowledgeService", mock_ks),
        ):
            result = await _get_index_fn()(MagicMock(), _base_data())

        assert result["status"] == "error"
        assert "not found" in result["msg"]

    @pytest.mark.asyncio
    async def test_successful_run_returns_processed_data(self):
        settings = _make_settings()

        mock_ks_instance = MagicMock()
        mock_ks_instance.get_knowledge_document.return_value = MagicMock()
        mock_ks_instance.get_knowledge.return_value = MagicMock()
        mock_runner = AsyncMock()

        with (
            patch("app.tasks.index_knowledge_document.get_settings", return_value=settings),
            patch("app.tasks.index_knowledge_document.get_me_postgres", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_elastic_search", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_storage", new_callable=AsyncMock),
            patch("app.tasks.index_knowledge_document.get_me_integration_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.get_me_vault_service_client", return_value=MagicMock()),
            patch("app.tasks.index_knowledge_document.KnowledgeService", return_value=mock_ks_instance),
            patch("app.tasks.index_knowledge_document.IndexingRunner", return_value=mock_runner),
        ):
            result = await _get_index_fn()(MagicMock(), _base_data())

        assert result == {"processed_data": "done"}
