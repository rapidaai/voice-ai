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
import asyncio
import logging
from logging.config import dictConfig

from celery import Celery
from celery.signals import worker_init, worker_shutdown, task_prerun

from app.config import get_settings
from app.configs.log_config import LogConfig
from app.connectors import Connector
from app.connectors.connector_factory import attach_connectors
from app.storage.connector_factory import attach_storage
from app.vars import service_name

# Set the service name
service_name.set(get_settings().service_name)

# Initialize the logger
_log = logging.getLogger("celery_worker")

dictConfig(LogConfig(level=get_settings().log_level).model_dump())

# Initialize Celery app
celery_app = Celery(
    service_name.get(),
    broker=get_settings().celery.broker,
    backend=get_settings().celery.backend,
    include=['app.tasks.index_knowledge_document']
)

APP_STORAGE = {}


def init_async_connectors():
    """
    Run async connector initialization in a synchronous context.
    """

    async def init_connectors():
        cntr: Connector
        for cntr in attach_connectors(get_settings()):
            await cntr.connect()  # Await async connect
            APP_STORAGE[cntr.name] = cntr
        APP_STORAGE['storage'] = attach_storage(get_settings())

    # Run the async function synchronously
    asyncio.run(init_connectors())


def shutdown_async_connectors():
    """
    Run async connector shutdown in a synchronous context.
    """

    async def shutdown_connectors():
        if APP_STORAGE and isinstance(APP_STORAGE, dict):
            for cntr in APP_STORAGE.values():
                await cntr.disconnect()  # Await async disconnect

    # Run the async function synchronously
    asyncio.run(shutdown_connectors())


@worker_init.connect
def on_worker_init(**kwargs):
    """
    Initialize connectors when Celery worker starts.
    Attach connectors to the app state (APP_STORAGE).
    """
    _log.debug("Initializing connectors on worker start")
    init_async_connectors()  # Call the async init function synchronously


@worker_shutdown.connect
def on_worker_shutdown(**kwargs):
    """
    Disconnect connectors when Celery worker shuts down.
    """
    shutdown_async_connectors()


@task_prerun.connect
def task_prerun_handler(task_id, task, args, **kwargs):
    """Create a new database session before task execution."""
    task.request.state = {
        'datasource': APP_STORAGE
    }


if __name__ == "__main__":
    _log.info(f"Starting Celery worker for {get_settings().service_name}")
    celery_app.start()
