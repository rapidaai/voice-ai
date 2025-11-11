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

from abc import ABC, abstractmethod
from typing import Any, List

from pydantic.v1 import Extra
from semantic_router.encoders.base import BaseEncoder

from app.core.chunkers import Chunk
from app.core.splitter import BaseSplitter


class BaseChunker(ABC):
    name: str
    encoder: BaseEncoder
    splitter: BaseSplitter

    class Config:
        extra = Extra.allow

    def __init__(self, name, encoder, splitter):
        self.name = name
        self.encoder = encoder
        self.splitter = splitter

    def __call__(self, docs: List[str]) -> List[List[Chunk]]:
        raise NotImplementedError("Subclasses must implement this method")

    def _split(self, doc: str) -> List[str]:
        return self.splitter(doc)

    @abstractmethod
    def _chunk(self, splits: List[Any]) -> List[Chunk]:
        raise NotImplementedError("Subclasses must implement this method")
