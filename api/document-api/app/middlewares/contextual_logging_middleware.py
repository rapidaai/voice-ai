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
from dataclasses import dataclass
from uuid import uuid4

from elasticapm.traces import execution_context
from fastapi import FastAPI
from opentelemetry.trace import (
    INVALID_SPAN,
    Span,
    format_span_id,
    format_trace_id,
    get_current_span,
)
from starlette.types import Message, Receive, Scope, Send

from app.config import ApplicationSettings
from app.vars import ObservabilityType, observability_type, span_id, trace_id

_log = logging.getLogger("app.common.middlewares.contextual_logging_middleware")


@dataclass
class ContextualLoggingMiddleware:
    """
    Contextual Logger
    - Making log contextual to trace_id and span_id
    - Every console log or file appender log format can use span_id and trace_id to update the log records
    - Can also be identified in the service anywhere to find the span and trace ids as it's using contextual variable
    """

    # the app where the middleware will be added.
    app: FastAPI

    # app settings
    # primarily used for identifying if elastic apm or open-telemetry is there
    settings: ApplicationSettings

    async def __call__(self, scope: "Scope", receive: "Receive", send: "Send") -> None:
        def elastic_apm_attributes_setup():
            """
            Only valid for elastic apm when it's setup
            - transaction id is the trace_id
            - trace_id is considered as span_id in elastic apm case.

            Getting the data from elastic apm client and setting to the context variable.
            """

            transaction = execution_context.get_transaction()
            trace_id.set(transaction.id) if transaction else None
            span_id.set(
                transaction.trace_parent.trace_id
            ) if transaction and transaction.trace_parent else None

        def open_telemetry_attribute_setup():
            """
            For open telemetry.
            - Getting current context span from open telemetry and setting up the value.
            - Span id is open-telemetry span id
            - trace_id is open-telemetry trace id
            """
            current_span: Span = get_current_span()
            if current_span != INVALID_SPAN:
                span_context = get_current_span().get_span_context()
                span_id.set(
                    str(
                        format_span_id(
                            span_context.span_id
                            if span_context.span_id
                            else uuid4().hex
                        )
                    )
                )
                trace_id.set(
                    str(
                        format_trace_id(
                            span_context.trace_id
                            if span_context.trace_id
                            else uuid4().hex
                        )
                    )
                )

        # if there is no telemetry and apm then set the uuid as span and tra
        def fallback_attribute():
            """
            if nothing is enabled open telemetry and elastic apm use uuid for transaction and span
            """
            span_id.set(uuid4().hex)
            trace_id.set(uuid4().hex)

        # set the attributes depends on apm and otel
        if observability_type.get() == ObservabilityType.ELASTICAPM:
            _log.debug("Adding elastic apm attributes to the log.")
            elastic_apm_attributes_setup()
        elif observability_type.get() == ObservabilityType.OPENTELEMETRY:
            _log.debug("Adding open telemetry attributes to the log.")
            open_telemetry_attribute_setup()
        else:
            _log.debug("Adding default attributes to the log.")
            # fallback_attribute()

        # check the scope of request
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        async def handle_outgoing_request(message: "Message") -> None:
            # @TODO when will need pass trace header to other services
            # if message["type"] == "http.response.start" and trace_id.get():
            #     headers = MutableHeaders(scope=message)
            #     _log.debug(headers)
            await send(message)

        await self.app(scope, receive, handle_outgoing_request)
        return
