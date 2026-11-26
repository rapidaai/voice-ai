"""
Task-level conftest: stubs app.celery_worker in sys.modules so that
app.tasks.index_knowledge_document can be imported without a live
Celery broker / config (celery_worker.py calls get_settings().celery
at module level, which requires a fully populated config.yaml).
"""
import sys
from unittest.mock import MagicMock

if "app.celery_worker" not in sys.modules:
    _stub = MagicMock()
    _stub.celery_app = MagicMock()
    _stub.web_service_assertion_issuer = MagicMock()
    _stub.integration_service_assertion_issuer = MagicMock()
    sys.modules["app.celery_worker"] = _stub
