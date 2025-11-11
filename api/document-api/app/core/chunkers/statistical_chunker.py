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
import tiktoken
from semantic_router.encoders.base import BaseEncoder
from tqdm.auto import tqdm

from app.core.chunkers import Chunk
from app.core.chunkers.base_chunker import BaseChunker
from app.core.splitter import SentenceRegexSplitter, BaseSplitter

logger = logging.getLogger(__name__)


def tiktoken_length(text: str) -> int:
    tokenizer = tiktoken.get_encoding("cl100k_base")
    tokens = tokenizer.encode(text, disallowed_special=())
    return len(tokens)


class StatisticalChunker(BaseChunker):
    def __init__(
        self,
        encoder: BaseEncoder,
        splitter: BaseSplitter = SentenceRegexSplitter(),
        name="statistical_chunker",
        threshold_adjustment=0.01,
        dynamic_threshold: bool = True,
        window_size=5,
        min_split_tokens=100,
        max_split_tokens=300,
        split_tokens_tolerance=10,
        plot_chunks=False,
        enable_statistics=False,
    ):
        super().__init__(name=name, encoder=encoder, splitter=splitter)
        self.encoder = encoder
        self.threshold_adjustment = threshold_adjustment
        self.dynamic_threshold = dynamic_threshold
        self.window_size = window_size
        self.plot_chunks = plot_chunks
        self.min_split_tokens = min_split_tokens
        self.max_split_tokens = max_split_tokens
        self.split_tokens_tolerance = split_tokens_tolerance
        self.enable_statistics = enable_statistics
        self.DEFAULT_THRESHOLD = 0.5

    def _chunk(
        self, splits: List[Any], batch_size: int = 64, enforce_max_tokens: bool = False
    ) -> List[Chunk]:
        """Merge splits into chunks using semantic similarity, with optional enforcement
        of maximum token limits per chunk.

        :param splits: Splits to be merged into chunks.
        :param batch_size: Number of splits to process in one batch.
        :param enforce_max_tokens: If True, further split chunks that exceed the maximum
        token limit.

        :return: List of chunks.
        """
        # Split the docs that already exceed max_split_tokens to smaller chunks
        if enforce_max_tokens:
            new_splits = []
            for split in splits:
                token_count = tiktoken_length(split)
                if token_count > self.max_split_tokens:
                    logger.info(
                        f"Single document exceeds the maximum token limit "
                        f"of {self.max_split_tokens}. "
                        "Splitting to sentences before semantically merging."
                    )
                    _splits = self._split(split)
                    new_splits.extend(_splits)
                else:
                    new_splits.append(split)

            splits = [split for split in new_splits if split and split.strip()]

        chunks = []
        last_chunk: Chunk | None = None
        for i in tqdm(range(0, len(splits), batch_size)):
            batch_splits = splits[i : i + batch_size]
            if last_chunk is not None:
                batch_splits = last_chunk.splits + batch_splits

            encoded_splits = self._encode_documents(batch_splits)
            similarities = self._calculate_similarity_scores(encoded_splits)

            if self.dynamic_threshold:
                calculated_threshold = self._find_optimal_threshold(
                    batch_splits, similarities
                )
            else:
                calculated_threshold = (
                    self.encoder.score_threshold
                    if self.encoder.score_threshold
                    else self.DEFAULT_THRESHOLD
                )
            split_indices = self._find_split_indices(
                similarities=similarities, calculated_threshold=calculated_threshold
            )

            doc_chunks = self._split_documents(
                docs=batch_splits,
                split_indices=split_indices,
                similarities=similarities,
            )

            if len(doc_chunks) > 1:
                chunks.extend(doc_chunks[:-1])
                last_chunk = doc_chunks[-1]
            else:
                last_chunk = doc_chunks[0]

            if self.plot_chunks:
                self.plot_similarity_scores(
                    similarities=similarities,
                    split_indices=split_indices,
                    chunks=doc_chunks,
                    calculated_threshold=calculated_threshold,
                )

        if last_chunk:
            chunks.append(last_chunk)

        return chunks

    def __call__(self, docs: List[str], batch_size: int = 64) -> List[List[Chunk]]:
        """Split documents into smaller chunks based on semantic similarity.

        :param docs: list of text documents to be split, if only wanted to
            split a single document, pass it as a list with a single element.

        :return: list of Chunk objects containing the split documents.
        """
        if not docs:
            raise ValueError("At least one document is required for splitting.")

        all_chunks = []
        for doc in docs:
            token_count = tiktoken_length(doc)
            if token_count > self.max_split_tokens:
                logger.info(
                    f"Single document exceeds the maximum token limit "
                    f"of {self.max_split_tokens}. "
                    "Splitting to sentences before semantically merging."
                )
            if isinstance(doc, str):
                splits = self._split(doc)
                doc_chunks = self._chunk(splits, batch_size=batch_size)
                all_chunks.append(doc_chunks)
            else:
                raise ValueError("The document must be a string.")
        return all_chunks

    def _encode_documents(self, docs: List[str]) -> np.ndarray:
        """
        Encodes a list of documents into embeddings. If the number of documents
        exceeds 2000, the documents are split into batches to avoid overloading
        the encoder. OpenAI has a limit of len(array) < 2048.

        :param docs: List of text documents to be encoded.
        :return: A numpy array of embeddings for the given documents.
        """
        max_docs_per_batch = 2000
        embeddings = []

        for i in range(0, len(docs), max_docs_per_batch):
            batch_docs = docs[i : i + max_docs_per_batch]
            try:
                batch_embeddings = self.encoder(batch_docs)
                embeddings.extend(batch_embeddings)
            except Exception as e:
                logger.error(f"Error encoding documents {batch_docs}: {e}")
                raise

        return np.array(embeddings)

    def _calculate_similarity_scores(self, encoded_docs: np.ndarray) -> List[float]:
        raw_similarities = []
        for idx in range(1, len(encoded_docs)):
            window_start = max(0, idx - self.window_size)
            cumulative_context = np.mean(encoded_docs[window_start:idx], axis=0)
            curr_sim_score = np.dot(cumulative_context, encoded_docs[idx]) / (
                np.linalg.norm(cumulative_context) * np.linalg.norm(encoded_docs[idx])
                + 1e-10
            )
            raw_similarities.append(curr_sim_score)
        return raw_similarities

    def _find_split_indices(
        self, similarities: List[float], calculated_threshold: float
    ) -> List[int]:
        split_indices = []
        for idx, score in enumerate(similarities):
            logger.debug(f"Similarity score at index {idx}: {score}")
            if score < calculated_threshold:
                logger.debug(
                    f"Adding to split_indices due to score < threshold: "
                    f"{score} < {calculated_threshold}"
                )
                # Chunk after the document at idx
                split_indices.append(idx + 1)
        return split_indices

    def _find_optimal_threshold(self, docs: List[str], similarity_scores: List[float]):
        token_counts = [tiktoken_length(doc) for doc in docs]
        cumulative_token_counts = np.cumsum([0] + token_counts)

        # Analyze the distribution of similarity scores to set initial bounds
        median_score = np.median(similarity_scores)
        std_dev = np.std(similarity_scores)

        # Set initial bounds based on median and standard deviation
        low = max(0.0, float(median_score - std_dev))
        high = min(1.0, float(median_score + std_dev))

        iteration = 0
        median_tokens = 0
        calculated_threshold = 0.0
        while low <= high:
            calculated_threshold = (low + high) / 2
            split_indices = self._find_split_indices(
                similarity_scores, calculated_threshold
            )
            logger.debug(
                f"Iteration {iteration}: Trying threshold: {calculated_threshold}"
            )

            # Calculate the token counts for each split using the cumulative sums
            split_token_counts = [
                cumulative_token_counts[end] - cumulative_token_counts[start]
                for start, end in zip(
                    [0] + split_indices, split_indices + [len(token_counts)]
                )
            ]

            # Calculate the median token count for the chunks
            median_tokens = np.median(split_token_counts)
            logger.debug(
                f"Iteration {iteration}: Median tokens per split: {median_tokens}"
            )
            if (
                self.min_split_tokens - self.split_tokens_tolerance
                <= median_tokens
                <= self.max_split_tokens + self.split_tokens_tolerance
            ):
                logger.debug("Median tokens in target range. Stopping iteration.")
                break
            elif median_tokens < self.min_split_tokens:
                high = calculated_threshold - self.threshold_adjustment
                logger.debug(f"Iteration {iteration}: Adjusting high to {high}")
            else:
                low = calculated_threshold + self.threshold_adjustment
                logger.debug(f"Iteration {iteration}: Adjusting low to {low}")
            iteration += 1

        logger.debug(
            f"Optimal threshold {calculated_threshold} found "
            f"with median tokens ({median_tokens}) in target range "
            f"({self.min_split_tokens}-{self.max_split_tokens})."
        )

        return calculated_threshold

    def _split_documents(
        self, docs: List[str], split_indices: List[int], similarities: List[float]
    ) -> List[Chunk]:
        """
        This method iterates through each document, appending it to the current split
        until it either reaches a split point (determined by split_indices) or exceeds
        the maximum token limit for a split (self.max_split_tokens).
        When a document causes the current token count to exceed this limit,
        or when a split point is reached and the minimum token requirement is met,
        the current split is finalized and added to the List of chunks.
        """
        token_counts = [tiktoken_length(doc) for doc in docs]
        chunks, current_split = [], []
        current_tokens_count = 0

        # Statistics
        chunks_by_threshold = 0
        chunks_by_max_chunk_size = 0
        chunks_by_last_split = 0

        for doc_idx, doc in enumerate(docs):
            doc_token_count = token_counts[doc_idx]
            logger.debug(f"Accumulative token count: {current_tokens_count} tokens")
            logger.debug(f"Document token count: {doc_token_count} tokens")
            # Check if current index is a split point based on similarity
            if doc_idx + 1 in split_indices:
                if (
                    self.min_split_tokens
                    <= current_tokens_count + doc_token_count
                    < self.max_split_tokens
                ):
                    # Include the current document before splitting
                    # if it doesn't exceed the max limit
                    current_split.append(doc)
                    current_tokens_count += doc_token_count

                    triggered_score = (
                        similarities[doc_idx] if doc_idx < len(similarities) else None
                    )
                    chunks.append(
                        Chunk(
                            splits=current_split.copy(),
                            is_triggered=True,
                            triggered_score=triggered_score,
                            token_count=current_tokens_count,
                        )
                    )
                    logger.debug(
                        f"Chunk finalized with {current_tokens_count} tokens due to "
                        f"threshold {triggered_score}."
                    )
                    current_split, current_tokens_count = [], 0
                    chunks_by_threshold += 1
                    continue  # Move to the next document after splitting

            # Check if adding the current document exceeds the max token limit
            if current_tokens_count + doc_token_count > self.max_split_tokens:
                if current_tokens_count >= self.min_split_tokens:
                    chunks.append(
                        Chunk(
                            splits=current_split.copy(),
                            is_triggered=False,
                            triggered_score=None,
                            token_count=current_tokens_count,
                        )
                    )
                    chunks_by_max_chunk_size += 1
                    logger.debug(
                        f"Chink finalized with {current_tokens_count} tokens due to "
                        f"exceeding token limit of {self.max_split_tokens}."
                    )
                    current_split, current_tokens_count = [], 0

            current_split.append(doc)
            current_tokens_count += doc_token_count

        # Handle the last split
        if current_split:
            chunks.append(
                Chunk(
                    splits=current_split.copy(),
                    is_triggered=False,
                    triggered_score=None,
                    token_count=current_tokens_count,
                )
            )
            chunks_by_last_split += 1
            logger.debug(
                f"Final split added with {current_tokens_count} "
                "tokens due to remaining documents."
            )

        # Validation to ensure no tokens are lost during the split
        original_token_count = sum(token_counts)
        split_token_count = sum(
            [tiktoken_length(doc) for split in chunks for doc in split.splits]
        )
        if original_token_count != split_token_count:
            logger.error(
                f"Token count mismatch: {original_token_count} != {split_token_count}"
            )
            raise ValueError(
                f"Token count mismatch: {original_token_count} != {split_token_count}"
            )

        # Statistics
        total_chunks = len(chunks)
        chunks_by_similarity_ratio = (
            chunks_by_threshold / total_chunks if total_chunks else 0
        )
        min_token_size = max_token_size = 0
        if chunks:
            token_counts = [
                split.token_count for split in chunks if split.token_count is not None
            ]
            min_token_size, max_token_size = min(token_counts, default=0), max(
                token_counts, default=0
            )

        return chunks

    def plot_similarity_scores(
        self,
        similarities: List[float],
        split_indices: List[int],
        chunks: list[Chunk],
        calculated_threshold: float,
    ):
        try:
            from matplotlib import pyplot as plt
        except ImportError:
            logger.warning(
                "Plotting is disabled. Please `pip install "
                "semantic-router[processing]`."
            )
            return

        _, axs = plt.subplots(2, 1, figsize=(12, 12))  # Adjust for two plots

        # Plot 1: Similarity Scores
        axs[0].plot(similarities, label="Similarity Scores", marker="o")
        for split_index in split_indices:
            axs[0].axvline(
                x=split_index - 1,
                color="r",
                linestyle="--",
                label="Chunk" if split_index == split_indices[0] else "",
            )
        axs[0].axhline(
            y=calculated_threshold,
            color="g",
            linestyle="-.",
            label="Threshold Similarity Score",
        )

        # Annotating each similarity score
        for i, score in enumerate(similarities):
            axs[0].annotate(
                f"{score:.2f}",  # Formatting to two decimal places
                (i, score),
                textcoords="offset points",
                xytext=(0, 10),  # Positioning the text above the point
                ha="center",
            )  # Center-align the text

        axs[0].set_xlabel("Document Segment Index")
        axs[0].set_ylabel("Similarity Score")
        axs[0].set_title(
            f"Threshold: {calculated_threshold} |" f" Window Size: {self.window_size}",
            loc="right",
            fontsize=10,
        )
        axs[0].legend()

        # Plot 2: Chunk Token Size Distribution
        token_counts = [split.token_count for split in chunks]
        axs[1].bar(range(len(token_counts)), token_counts, color="lightblue")
        axs[1].set_title("Chunk Token Sizes")
        axs[1].set_xlabel("Chunk Index")
        axs[1].set_ylabel("Token Count")
        axs[1].set_xticks(range(len(token_counts)))
        axs[1].set_xticklabels([str(i) for i in range(len(token_counts))])
        axs[1].grid(True)

        # Annotate each bar with the token size
        for idx, token_count in enumerate(token_counts):
            if not token_count:
                continue
            axs[1].text(
                idx, token_count + 0.01, str(token_count), ha="center", va="bottom"
            )

        plt.tight_layout()
        plt.show()

    def plot_sentence_similarity_scores(
        self, docs: List[str], threshold: float, window_size: int
    ):
        try:
            from matplotlib import pyplot as plt
        except ImportError:
            logger.warning("Plotting is disabled. Please `pip install matplotlib`.")
            return
        """
        Computes similarity scores between the average of the last
        'window_size' sentences and the next one,
        plots a graph of these similarity scores, and prints the first
        sentence after a similarity score below
        a specified threshold.
        """
        sentences = [sentence for doc in docs for sentence in self._split(doc)]
        encoded_sentences = self._encode_documents(sentences)
        similarity_scores = []

        for i in range(window_size, len(encoded_sentences)):
            window_avg_encoding = np.mean(
                encoded_sentences[i - window_size : i], axis=0
            )
            sim_score = np.dot(window_avg_encoding, encoded_sentences[i]) / (
                np.linalg.norm(window_avg_encoding)
                * np.linalg.norm(encoded_sentences[i])
                + 1e-10
            )
            similarity_scores.append(sim_score)

        plt.figure(figsize=(10, 8))
        plt.plot(similarity_scores, marker="o", linestyle="-", color="b")
        plt.title("Sliding Window Sentence Similarity Scores")
        plt.xlabel("Sentence Index")
        plt.ylabel("Similarity Score")
        plt.grid(True)
        plt.axhline(y=threshold, color="r", linestyle="--", label="Threshold")
        plt.show()

        for i, score in enumerate(similarity_scores):
            if score < threshold:
                print(
                    f"First sentence after similarity score "
                    f"below {threshold}: {sentences[i + window_size]}"
                )
