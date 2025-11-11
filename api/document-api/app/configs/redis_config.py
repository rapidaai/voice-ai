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

from typing import Optional

from pydantic import Field

from app.configs import ExternalDatasourceModel
from app.configs.auth.basic_auth import BasicAuth


class RedisConfig(ExternalDatasourceModel):
    """
    Redis config template
    - defined all required or optional parameter which will be needed to create redis connection
    - organize with pydantic lib to get loaded from .env
    """

    # Host of redis instance
    host: str

    # redis port
    port: int

    # db of redis
    db: int = 0

    # maximum number of connection at given point of time.
    max_connection: int = 5

    # charset
    charset: str = "utf-8"

    # decode_responses
    decode_responses: bool = True

    # support only basic auth
    auth: Optional[BasicAuth] = Field(
        default=None, description="authentication information for redis"
    )

    # do we need to place in docker container
    # or place as external service
    dockerize: Optional[bool] = True

    class Config:
        env_nested_delimiter = "__"
        env_file_encoding = "utf-8"
