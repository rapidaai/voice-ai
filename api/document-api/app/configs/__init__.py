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

from pydantic import BaseModel


class ExternalDatasourceModel(BaseModel):
    """
    Datasource model provide scope to config which are more related to datasource
    """

    # do we need to place in docker container
    # or place as external service
    # default false
    dockerize: Optional[bool] = False
