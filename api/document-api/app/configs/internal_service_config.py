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

from pydantic import BaseModel


class InternalServiceConfig(BaseModel):
    web_host: str
    integration_host: str
    endpoint_host: str
    assistant_host: str

    class Config:
        # For secret key
        env_file_encoding = "utf-8"
