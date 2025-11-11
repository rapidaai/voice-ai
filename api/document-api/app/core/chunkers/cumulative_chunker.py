import logging
from typing import Any, List

import numpy as np
from semantic_router.encoders import BaseEncoder
from tqdm.auto import tqdm

from app.core.chunkers import Chunk
from app.core.chunkers.base_chunker import BaseChunker
from app.core.splitter import BaseSplitter, SentenceRegexSplitter

logger = logging.getLogger(__name__)


class CumulativeChunker(BaseChunker):
    """
    Called "cumulative sim" because we check the similarities of the
    embeddings of cumulative concatenated documents with the next document.
    """

    def __init__(
        self,
        encoder: BaseEncoder,
        splitter: BaseSplitter = SentenceRegexSplitter(),
        name: str = "cumulative_chunker",
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
        chunks = []
        curr_chunk_start_idx = 0
        num_splits = len(splits)

        for idx in tqdm(range(num_splits)):
            if idx + 1 < num_splits:  # Ensure there is a next document to compare with.
                if idx == 0:
                    # On the first iteration, compare the
                    # first document directly to the second.
                    curr_chunk_docs = splits[idx]
                else:
                    # For subsequent iterations, compare cumulative
                    # documents up to the current one with the next.
                    curr_chunk_docs = "\n".join(splits[curr_chunk_start_idx : idx + 1])
                next_doc = splits[idx + 1]

                # Embedding and similarity calculation remains the same.
                curr_chunk_docs_embed = self.encoder([curr_chunk_docs])[0]
                next_doc_embed = self.encoder([next_doc])[0]
                curr_sim_score = np.dot(curr_chunk_docs_embed, next_doc_embed) / (
                    np.linalg.norm(curr_chunk_docs_embed)
                    * np.linalg.norm(next_doc_embed)
                )
                # Decision to chunk based on similarity score.
                if curr_sim_score < self.score_threshold:
                    chunks.append(
                        Chunk(
                            splits=list(splits[curr_chunk_start_idx : idx + 1]),
                            is_triggered=True,
                            triggered_score=curr_sim_score,
                        )
                    )
                    curr_chunk_start_idx = (
                        idx + 1
                    )  # Update the start index for the next segment.

        # Add the last segment after the loop.
        if curr_chunk_start_idx < num_splits:
            chunks.append(Chunk(splits=list(splits[curr_chunk_start_idx:])))

        return chunks

    async def _async_chunk(
        self, splits: List[Any], batch_size: int = 64
    ) -> List[Chunk]:
        """Merge splits into chunks using semantic similarity.

        :param splits: splits to be merged into chunks.

        :return: list of chunks.
        """
        chunks = []
        curr_chunk_start_idx = 0
        num_splits = len(splits)

        for idx in tqdm(range(num_splits)):
            if idx + 1 < num_splits:  # Ensure there is a next document to compare with.
                if idx == 0:
                    # On the first iteration, compare the
                    # first document directly to the second.
                    curr_chunk_docs = splits[idx]
                else:
                    # For subsequent iterations, compare cumulative
                    # documents up to the current one with the next.
                    curr_chunk_docs = "\n".join(splits[curr_chunk_start_idx : idx + 1])
                next_doc = splits[idx + 1]

                # Embedding and similarity calculation remains the same.
                curr_chunk_docs_embed_result = await self.encoder.acall(
                    [curr_chunk_docs]
                )
                next_doc_embed_result = await self.encoder.acall([next_doc])
                curr_chunk_docs_embed = curr_chunk_docs_embed_result[0]
                next_doc_embed = next_doc_embed_result[0]

                curr_sim_score = np.dot(curr_chunk_docs_embed, next_doc_embed) / (
                    np.linalg.norm(curr_chunk_docs_embed)
                    * np.linalg.norm(next_doc_embed)
                )
                # Decision to chunk based on similarity score.
                if curr_sim_score < self.score_threshold:
                    chunks.append(
                        Chunk(
                            splits=list(splits[curr_chunk_start_idx : idx + 1]),
                            is_triggered=True,
                            triggered_score=curr_sim_score,
                        )
                    )
                    curr_chunk_start_idx = (
                        idx + 1
                    )  # Update the start index for the next segment.

        # Add the last segment after the loop.
        if curr_chunk_start_idx < num_splits:
            chunks.append(Chunk(splits=list(splits[curr_chunk_start_idx:])))

        return chunks

    def __call__(self, docs: List[str]) -> List[List[Chunk]]:
        """Split documents into smaller chunks based on semantic similarity.

        :param docs: list of text documents to be chunk, if only wanted to
            chunk a single document, pass it as a list with a single element.

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

