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

from typing import Optional, Union

from pydantic import Field

from app.configs import ExternalDatasourceModel
from app.configs.auth.aws_auth import AWSAuth
from app.configs.auth.basic_auth import BasicAuth


class ElasticSearchConfig(ExternalDatasourceModel):
    """
    Elastic search configuration template
    """

    # Host
    host: str

    # Port of running elastic search
    port: Optional[int]

    # authentication of elastic search node
    auth: Optional[Union[BasicAuth, AWSAuth]] = Field(
        default=None, description="auth information for elastic search config"
    )

    # default schema is https can be override from env
    scheme: str = "https"

    # max number of connections
    max_connection: int = 5

    class Config:
        env_nested_delimiter = "__"
        env_file_encoding = "utf-8"
