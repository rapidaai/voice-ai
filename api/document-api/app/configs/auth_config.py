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

from typing import Set, Optional, Union

from pydantic import BaseModel, SecretStr, Field


class TokenAuthenticationConfig(BaseModel):
    # strict authentication
    strict: bool = False

    header_key: str = "Authorization"

    auth_host: Optional[str] = Field(
        default=None, description="auth host that will get called when authentication"
    )


class JwtAuthenticationConfig(BaseModel):
    """
    JWT configuration Template
    """

    strict: bool = False
    # secret key
    # print safe
    secret_key: SecretStr
    # algorithms for jwt
    algorithms: Set = ["HS256"]

    # header keys
    header_key: str = "Authorization"

    class Config:
        case_sensitive = True
        env_file_encoding = "utf-8"


class AuthenticationConfig(BaseModel):
    #
    type: Optional[str]

    config: Optional[Union[TokenAuthenticationConfig, JwtAuthenticationConfig]]
