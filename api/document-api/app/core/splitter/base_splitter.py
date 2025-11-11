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

from pydantic.v1 import BaseModel, Extra


class BaseSplitter(BaseModel):
    class Config:
        extra = Extra.allow

    def __call__(self, doc: str) -> List[str]:
        raise NotImplementedError("Subclasses must implement this method")