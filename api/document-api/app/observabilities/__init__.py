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
from contextlib import contextmanager
from enum import Enum
from typing import Iterator, Optional, Union

import elasticapm
from elasticapm.context import init_execution_context
from elasticapm.traces import Span as ESpan
from opentelemetry import trace
from opentelemetry.trace import Span as OSpan
from opentelemetry.trace import SpanKind, Status, StatusCode

from app.vars import ObservabilityType, observability_type, trace_name

# only for apm
"""

Provide enough cross functional feature support for
migration for elastic apm to open-telemetry specially made for python service template can be converted to other python projects.

- Api/ Method Signature conversion for elastic apm to open telemetry.
- Fail safe apis do not raise any exception.
- impure function interact with elastic apm and opentelemtry both.

"""

_log = logging.getLogger("app.observabilities.within_span")


def span_type_to_kind(span_type: str) -> SpanKind:
    """
    Span type to SpanKind
    // elastic apm has span_kind as string but open-telemetry need ENUM as span kind
    :param span_type:
    :return:
    """
    if span_type == "server":
        return SpanKind.SERVER
    elif span_type == "client":
        return SpanKind.CLIENT
    else:
        return SpanKind.INTERNAL


def kind() -> ObservabilityType:
    """
    Kind of observability enabled for service
    :return:
    """
    return observability_type.get()


class SpanOutcome(Enum):
    """
    Generalize outcome  for open-telemetry and elastic apm
    """

    # predefined outcome verify boolean either successes or failed.
    SUCCESS = "SUCCESS"
    FAILURE = "FAILURE"


def span_outcome_to_status(outcome: SpanOutcome):
    """
    converting elasticapm status to open telemetry apm
    :param outcome:
    :return:
    """
    if outcome == SpanOutcome.SUCCESS:
        return StatusCode.OK
    elif outcome == SpanOutcome.FAILURE:
        return StatusCode.ERROR
    else:
        return StatusCode.UNSET


def default_span_type() -> str:
    """
    Default span type for elastic span.
    :return:
    """

    return "internal"


class GlobalSpan:
    """
    Adapter pattern for open-telemetry and elastic apm span
    """

    span: Optional[Union[ESpan, OSpan]] = None

    def __init__(self, span: Optional[Union[ESpan, OSpan]] = None):
        self.span = span

    def set_status(self, status: SpanOutcome, description: Optional[str] = None):
        try:
            if kind() == ObservabilityType.OPENTELEMETRY:
                self.span.set_status(
                    Status(span_outcome_to_status(status), description=description),
                    description=description,
                )

            elif kind() == ObservabilityType.ELASTICAPM:
                elasticapm.set_transaction_result(status)
            else:
                pass
        except Exception as e:
            _log.error(f"got error while setting status {e}")

    def set_attribute(self, attr: str, val):
        """
        Setting attribute for spans
        Adapting open telemetry and elastic apm behaviour for setting additional information to spans

        :param attr: always takes string
        :param val:
        :return:
        """
        try:
            if kind() == ObservabilityType.OPENTELEMETRY:
                self.span.set_attribute(attr, str(val))

            elif kind() == ObservabilityType.ELASTICAPM:
                elasticapm.set_custom_context({attr: str(val)})
            else:
                pass
        except Exception as e:
            _log.error(f"got error while setting attribute {attr}, {val} {e}")


@contextmanager
def within_span(
    name: str,
    span_type: Optional[str] = "internal",
    span_subtype: Optional[str] = None,
    span_action: Optional[str] = None,
) -> Iterator[GlobalSpan]:
    """
    Global spanner apis can be used for elastic apm and open-telemetry both
    As some part of lomotif services uses open-telemetry and some elastic apm
    it will be cross-functional and easy migrating senselessly

    :param name:
    :param span_type:
    :param span_subtype:
    :param span_action:
    :return:
    """
    try:

        if observability_type.get() == ObservabilityType.OPENTELEMETRY:

            _log.debug("observability_type is defined as open telemetry.")
            with trace.get_tracer(trace_name.get()).start_as_current_span(
                name, kind=span_type_to_kind(span_type)
            ) as span:
                """
                Persist the action and types depends on all elastic apm semantic value as attribute.
                """
                if span_type is not None:
                    span.set_attribute("type", span_type)

                if span_subtype is not None:
                    span.set_attribute("subtype", span_subtype)

                if span_action is not None:
                    span.set_attribute("action", span_action)

                # make sure to follow same pattern as current elastic apm implementation return iterator
                with use_global_span(GlobalSpan(span)) as gSpan:
                    yield gSpan
        #
        elif observability_type.get() == ObservabilityType.ELASTICAPM:
            _log.debug("observability_type is defined as elastic apm.")
            with elasticapm.capture_span(
                name,
                span_type=span_type,
                span_subtype=span_subtype,
                span_action=span_action,
            ):

                execution_context = init_execution_context()
                with use_global_span(GlobalSpan(execution_context.get_span())) as span:
                    yield span

        else:
            # don't fail
            with use_global_span(GlobalSpan(None)) as span:
                yield span
    except Exception as e:
        """
        Fail safe do not raise any exception for anything.
        """
        print(e.__traceback__)
        _log.error(f"got error while creating span. {e}")
        raise


@contextmanager
def use_global_span(span: GlobalSpan) -> None:
    """
    Global span
    // @Todo context change implementation requires for thread local <APM specific>
    :param span:
    :return:
    """
    yield span
