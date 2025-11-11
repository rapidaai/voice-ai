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

from pydantic import BaseModel, SecretStr, Field


class AWSAuth(BaseModel):
    # aws region
    region: str

    # aws access key
    access_key_id: Optional[SecretStr]

    # aws secret key
    secret_access_key: Optional[SecretStr]

    # if sts get used
    assume_role: Optional[str] = Field(default=None)

    class Config:
        case_sensitive = True
        env_file_encoding = "utf-8"
