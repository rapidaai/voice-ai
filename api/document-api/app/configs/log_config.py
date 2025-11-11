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
from typing import Any, Dict

from pydantic import BaseModel

from app.vars import service_name, span_id, trace_id


class TracingIdFilter(logging.Filter):
    """Logging filter to attached tracing and span id to log records"""

    def __init__(self):
        super().__init__(name="tracing_id_filter")

    def filter(self, record: logging.LogRecord) -> bool:
        """
        Attach a tracing id and span id to the log record.
        Since the ids are defined in the middleware layer, any
        log generated from a request after this point can easily be searched
        for, if the ids is added to the message, or included as
        metadata.
        """
        record.spanning_id = span_id.get()
        record.tracing_id = trace_id.get()
        record.service_name = service_name.get()
        return True


class TracingLogFormatter(logging.Formatter):
    """Overriding log format. Do not format span, trace if not in otel or apm"""

    default_fmt: str = "%(asctime)s.%(msecs)03d %(levelname)s %(pathname)s %(lineno)d %(funcName)s %(name)s %(message)s"

    def __init__(
            self,
            fmt=default_fmt,
            datefmt: str = "%Y-%m-%d %H:%M:%S",
            style="%",
            validate=True,
    ):
        super().__init__(fmt, datefmt, style, validate)

    def format(self, record: logging.LogRecord):
        # copy already existing format given
        e_style: logging.PercentStyle = self._style

        if trace_id.get() is not None:
            self._style = logging.PercentStyle(
                f"{self.default_fmt} [trace_id:%(tracing_id)s span_id:%(spanning_id)s "
                f"service_name:%(service_name)s]"
            )
        result = super().format(record)

        # update the format back
        self._style = e_style
        return result


class LogConfig(BaseModel):
    version: int = 1

    # level
    level: str = "INFO"
    #
    filters: Dict[Any, Any] = {
        "tracing_id_filter": {
            "()": "app.configs.log_config.TracingIdFilter",
        },
    }
    # handlers
    disable_existing_loggers: bool = (False,)
    # "Name of formatter" : {Formatter Config Dict}
    formatters: Dict[Any, Any] = {
        # Formatter Name
        "standard": {
            "()": "app.configs.log_config.TracingLogFormatter",
        }
    }

    # Handlers use the formatter names declared above
    handlers: Dict[Any, Any] = {
        # Name of handler
        "console": {
            # The class of logger. A mixture of logging.config.dictConfig() and
            # logger class-specific keyword arguments (kwargs) are passed in here.
            "class": "logging.StreamHandler",
            # This is the formatter name declared above
            "formatter": "standard",
            "level": level,
            "filters": ["tracing_id_filter"],
            # The default is stderr
            "stream": "ext://sys.stdout",
        }
    }

    root: Dict[Any, Any] = {
        # The default is not set
        "level": level,
        # The default is the handlers declared above
        "handlers": ["console"],
    }
    # Loggers use the handler names declared above
    loggers: Dict[Any, Any] = {
        "app": {
            # Use a list even if one handler is used
            "handlers": ["console"],
            "level": level,
            "propagate": False,
        },
        # deprecating uvicorn logger
        "uvicorn": {
            "level": logging.CRITICAL,
        },
    }

    def __init__(self, **data: Any):
        super().__init__(**data)
        self.root["level"] = data["level"].upper()
        self.loggers["app"]["level"] = data["level"].upper()
        self.loggers["uvicorn"]["level"] = "ERROR"
        self.handlers["console"]["level"] = data["level"].upper()
