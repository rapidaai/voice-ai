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

from app.configs import ExternalDatasourceModel
from app.configs.auth.basic_auth import BasicAuth


class PostgresConfig(ExternalDatasourceModel):
    """
    Postgres Config Template
    """

    # Host of postgres without protocol and port
    host: str

    # Postgres running port
    port: int

    # Database name
    db: str

    # Support of user and password authentication to postgres
    auth: Optional[BasicAuth]

    # Minimum number of connection to be active at any given point of time
    ideal_connection: int = 1

    # Maximum number of connection to be active at any given point of time
    max_connection: int = 5

    # do we need to place in docker container
    # or place as external service
    dockerize: Optional[bool] = True

    class Config:
        # need to be case-sensitive db-name (ato generated db name problem)
        env_nested_delimiter = "__"
        case_sensitive = True
        env_file_encoding = "utf-8"
