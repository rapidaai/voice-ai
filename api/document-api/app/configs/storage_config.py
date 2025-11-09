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
from pydantic_settings import SettingsConfigDict

from app.configs import ExternalDatasourceModel
from app.configs.auth.aws_auth import AWSAuth


class AssetStoreConfig(ExternalDatasourceModel):
    storage_type: Optional[str]
    storage_path_prefix: Optional[str]
    auth: Optional[AWSAuth] = Field(
        default=None, description="auth information for storage config"
    )
    model_config = SettingsConfigDict(env_file_encoding="utf-8", extra="ignore")
