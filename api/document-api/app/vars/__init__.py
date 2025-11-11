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
from contextvars import ContextVar
from enum import Enum
from typing import Optional


class ObservabilityType(Enum):
    """
    open-telemetry type
    """

    OPENTELEMETRY = 0

    ELASTICAPM = 1


# using for otel span and trace id
observability_type: ContextVar[Optional[ObservabilityType]] = ContextVar(
    "observability_type", default=None
)
trace_name: ContextVar[Optional[str]] = ContextVar(
    "trace_name", default="app.middlewares.open_telemetry_middleware"
)
meter_name: ContextVar[Optional[str]] = ContextVar(
    "meter_name", default="app.middlewares.open_telemetry_middleware"
)
trace_id: ContextVar[Optional[str]] = ContextVar("trace_id", default=None)
span_id: ContextVar[Optional[str]] = ContextVar("span_id", default=None)
service_name: ContextVar[Optional[str]] = ContextVar(
    "service_name", default="python_service"
)
