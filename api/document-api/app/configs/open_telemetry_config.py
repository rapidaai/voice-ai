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
from typing import List

from pydantic import BaseModel


class OpenTelemetryConfig(BaseModel):
    """
    OpenTelemetry configuration template
    """

    enable: bool

    # Debug
    debug: bool = False

    # flag for ssl/tls
    insecure: bool = True

    # ignore url
    ignore_urls: List[str] = ["/readiness/", "/healthz/"]

    # for setting log attribute
    enable_log_tracing: bool = False

    class Config:

        # For secret key
        case_sensitive = True
        env_file_encoding = "utf-8"
