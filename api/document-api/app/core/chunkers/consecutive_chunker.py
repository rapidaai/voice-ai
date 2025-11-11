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
import logging
from typing import Any, List

import numpy as np
from semantic_router.encoders.base import BaseEncoder
from tqdm.auto import tqdm

from app.core.chunkers import Chunk
from app.core.chunkers.base_chunker import BaseChunker
from app.core.splitter import BaseSplitter, SentenceRegexSplitter

logger = logging.getLogger(__name__)


class ConsecutiveChunker(BaseChunker):
    """
    Called "consecutive sim chunker" because we check the similarities of consecutive document embeddings (compare ith to i+1th document embedding).
    """

    def __init__(
        self,
        encoder: BaseEncoder,
        splitter: BaseSplitter = SentenceRegexSplitter(),
        name: str = "consecutive_chunker",
        score_threshold: float = 0.45,
    ):
        super().__init__(name=name, encoder=encoder, splitter=splitter)
        encoder.score_threshold = score_threshold
        self.score_threshold = score_threshold

    def _chunk(self, splits: List[Any], batch_size: int = 64) -> List[Chunk]:
        """Merge splits into chunks using semantic similarity.

        :param splits: splits to be merged into chunks.

        :return: list of chunks.
        """
        split_embeds = []
        num_splits = len(splits)
        for i in tqdm(range(0, num_splits, batch_size)):
            split_embeds.extend(self.encoder(splits[i : i + batch_size]))
        norm_embeds = split_embeds / np.linalg.norm(split_embeds, axis=1, keepdims=True)
        sim_matrix = np.matmul(norm_embeds, norm_embeds.T)
        chunks = []
        curr_split_start_idx = 0

        for idx in tqdm(range(1, norm_embeds.shape[0])):
            curr_sim_score = sim_matrix[idx - 1][idx]
            if idx < len(sim_matrix) and curr_sim_score < self.score_threshold:
                chunks.append(
                    Chunk(
                        splits=splits[curr_split_start_idx:idx],
                        is_triggered=True,
                        triggered_score=curr_sim_score,
                    )
                )
                curr_split_start_idx = idx
        # append final chunk
        chunks.append(Chunk(splits=splits[curr_split_start_idx:]))
        self.chunks = chunks
        return chunks

    async def _async_chunk(
        self, splits: List[Any], batch_size: int = 64
    ) -> List[Chunk]:
        """Merge splits into chunks using semantic similarity.

        :param splits: splits to be merged into chunks.

        :return: list of chunks.
        """
        split_embeds = []
        num_splits = len(splits)
        for i in tqdm(range(0, num_splits, batch_size)):
            split_embeds.extend(await self.encoder.acall(splits[i : i + batch_size]))
        norm_embeds = split_embeds / np.linalg.norm(split_embeds, axis=1, keepdims=True)
        sim_matrix = np.matmul(norm_embeds, norm_embeds.T)
        chunks = []
        curr_split_start_idx = 0

        for idx in tqdm(range(1, norm_embeds.shape[0])):
            curr_sim_score = sim_matrix[idx - 1][idx]
            if idx < len(sim_matrix) and curr_sim_score < self.score_threshold:
                chunks.append(
                    Chunk(
                        splits=splits[curr_split_start_idx:idx],
                        is_triggered=True,
                        triggered_score=curr_sim_score,
                    )
                )
                curr_split_start_idx = idx
        # append final chunk
        chunks.append(Chunk(splits=splits[curr_split_start_idx:]))
        self.chunks = chunks
        return chunks

    def __call__(self, docs: List[Any]) -> List[List[Chunk]]:
        """Split documents into smaller chunks based on semantic similarity.

        :param docs: list of text documents to be split, if only wanted to
            split a single document, pass it as a list with a single element.

        :return: list of list objects containing the chunks.
        """
        all_chunks = []
        for doc in docs:
            # split the document into sentences (if needed)
            if isinstance(doc, str):
                splits = self._split(doc)
            else:
                splits = doc
            doc_chunks = self._chunk(splits)
            all_chunks.append(doc_chunks)
        return all_chunks

    async def acall(self, docs: List[Any]) -> List[List[Chunk]]:
        all_chunks = []
        for doc in docs:
            # split the document into sentences (if needed)
            if isinstance(doc, str):
                splits = self._split(doc)
            else:
                splits = doc
            doc_chunks = await self._async_chunk(splits)
            all_chunks.append(doc_chunks)
        return all_chunks