import logging
from typing import List, Tuple

from app.core.embedding.embedding import Embedder
from app.core.model_runtime.model_manager import ModelManager
from app.utils.rapida_metric import get_metric_token_count

logger = logging.getLogger(__name__)


class PlainEmbedder(Embedder):
    model_manager: ModelManager

    def __init__(
            self, model_instance: ModelManager
    ) -> None:
        # In the provided code snippet, `self.model_manager = model_instance` and `self.postgres =
        # postgres` are initializing instance variables within the `PostgresCacheEmbedding` class in
        # Python.
        self.model_manager = model_instance

    async def embed_documents(self, texts: List[str]) -> Tuple[List[List[float]], int]:
        """Embed search docs in batches of 10."""
        # use doc embedding cache or store if not exists
        text_embeddings = [None for _ in range(len(texts))]
        token_count = 0
        embedding_queue_indices = []
        for i, text in enumerate(texts):
            embedding_queue_indices.append(i)
        if embedding_queue_indices:
            embedding_queue_texts = [texts[i] for i in embedding_queue_indices]
            try:
                embedding_result, mtr = (
                    await self.model_manager.invoke_text_embedding(
                        texts=embedding_queue_texts
                    )
                )
                token_count += get_metric_token_count(mtr)
                logger.debug(f"metrics === {mtr}  and token count {token_count}")
                for i, embedding in zip(
                        embedding_queue_indices, embedding_result
                ):
                    text_embeddings[i] = list(embedding.embedding)
            except Exception as ex:
                logger.error(
                    f"Error in embedding documents: {ex}",
                    exc_info=True
                )
                raise ex
        return text_embeddings, token_count
