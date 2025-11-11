"""Paragraph index processor."""

import logging
from typing import Optional, List

from app.commons import constants
from app.config import get_settings
from app.configs.extractor_config import ExtractorConfig
from app.core.chunkers.base_chunker import BaseChunker
from app.core.chunkers.chunker_factory import ChunkerFactory
from app.core.embedding.embedding import Embedder
from app.core.rag.datasource.vdb import constants
from app.core.rag.datasource.vdb.vector_base import BaseVector
from app.core.rag.datasource.vdb.vector_factory import Vector
from app.core.rag.extractor.extract_processor import ExtractProcessor
from app.core.rag.index_processor.index_processor_base import BaseIndexProcessor
from app.core.rag.models.document import Document
from app.core.transformers.document_transformer import BaseDocumentTransformer
from app.core.transformers.transformer_factory import initialize_transformers
from app.models.knowledge_model import KnowledgeDocument
from app.storage.storage import Storage
from app.utils.general import generate_text_hash

logger = logging.getLogger(__name__)


class ParagraphIndexProcessor(BaseIndexProcessor):
    # The line `storage: Storage` is defining a class attribute `storage` of type `Storage` in the
    # `ParagraphIndexProcessor` class. This attribute is declared at the class level and is shared
    # among all instances of the class. It indicates that an instance of `ParagraphIndexProcessor`
    # should have a `storage` attribute that is expected to be an object of type `Storage`. This
    # attribute can be accessed and modified by any method within the class.
    storage: Storage

    # The line `cfg: ExtractorConfig` is defining a class attribute `cfg` of type `ExtractorConfig` in
    # the `ParagraphIndexProcessor` class. This attribute is declared at the class level and is shared
    # among all instances of the class. It indicates that an instance of `ParagraphIndexProcessor`
    # should have a `cfg` attribute that is expected to be an object of type `ExtractorConfig`. This
    # attribute can be accessed and modified by any method within the class.
    cfg: ExtractorConfig

    def __init__(
            self,
            storage: Storage,
    ) -> None:
        super().__init__()
        self.storage = storage
        self.cfg = get_settings().knowledge_extractor_config

    async def extract(
            self, knowledge_document: KnowledgeDocument, **kwargs
    ) -> List[Document]:
        """

        :param knowledge_document:
        :param kwargs:
        :return:
        """
        logger.info(
            "ParagraphIndexProcessor.Extract knowledge_id: {} and knowledge_document_id: {}".format(
                knowledge_document.knowledge_id, knowledge_document.knowledge_id
            )
        )
        text_docs = ExtractProcessor(self.storage, self.cfg).extract(knowledge_document)
        return text_docs

    async def transform(
            self, knowledge_document: KnowledgeDocument, documents: List[Document], **kwargs
    ) -> List[Document]:
        """
        Args:
            knowledge_document:
            documents:
            **kwargs:

        Returns:
        :param knowledge_document:
        :param documents:

        """
        # Split the text documents into nodes.
        logger.info(
            "ParagraphIndexProcessor.transform knowledge_id: {} and knowledge_document_id: {}".format(
                knowledge_document.knowledge_id, knowledge_document.knowledge_id
            )
        )
        # This block of code is responsible for splitting the text documents into smaller nodes or
        # chunks. Here's a breakdown of what each step is doing:
        all_documents: List[Document] = []

        if self.cfg.chunking_technique is not None:
            chunker: BaseChunker = ChunkerFactory(self.cfg.chunking_technique).get()
            chunked_document = chunker(docs=[cntnt.page_content for cntnt in documents])
            # This block of code is iterating over each document in the `chunked_document` list. For each
            # document, it further iterates over the chunks within that document.
            for document in chunked_document:
                split_documents = []
                for chunk in document:
                    if chunk.content.strip():

                        page_content = chunk.content
                        if page_content.startswith(".") or page_content.startswith("。"):
                            page_content = page_content[1:].strip()
                        # The code block `if len(page_content) > 0:` is checking if the `page_content` of a
                        # document chunk is not empty. If the length of the `page_content` is greater than
                        # 0 (i.e., it is not an empty string), then the following actions are taken:
                        if len(page_content) == 0:
                            continue

                        split_documents.append(Document(page_content=page_content))
                all_documents.extend(split_documents)

            # transformer
        else:
            # If no chunking technique is specified, the code simply extends the `all_documents` list
            # with the original documents.
            all_documents.extend(documents)

        parsed_document: List[Document] = []
        for document_node in all_documents:
            doc_id = generate_text_hash(document_node.page_content)
            document_node.metadata[constants.Field.DOCUMENT_HASH_KEY.value] = (
                doc_id
            )
            document_node.metadata[constants.Field.DOCUMENT_ID_KEY.value] = str(
                doc_id
            )
            document_node.metadata[
                constants.Field.KNOWLEDGE_DOCUMENT_ID_KEY.value
            ] = knowledge_document.id
            document_node.metadata[constants.Field.KNOWLEDGE_ID_KEY.value] = (
                knowledge_document.knowledge_id
            )
            document_node.metadata[constants.Field.PROJECT_ID_KEY.value] = (
                knowledge_document.project_id
            )
            document_node.metadata[
                constants.Field.ORGANIZATION_ID_KEY.value
            ] = knowledge_document.organization_id
            parsed_document.append(document_node)

        post_transformers: List[BaseDocumentTransformer] = initialize_transformers(
            self.cfg.transformers, stage="post"
        )
        if not post_transformers or len(post_transformers) == 0:
            return parsed_document

        for trf in post_transformers:
            parsed_document = trf.transform_documents(parsed_document)

        return parsed_document

    async def load(
            self,
            knowledge_document: KnowledgeDocument,
            documents: List[Document],
            embedder: Embedder,
            vector_processor: BaseVector,
    ) -> int:
        logger.info(
            "ParagraphIndexProcessor.load knowledge_id: {} and knowledge_document_id: {}".format(
                knowledge_document.knowledge_id, knowledge_document.id
            )
        )
        vector = Vector(
            knowledge_document=knowledge_document,
            embedder=embedder,
            vector_processor=vector_processor,
        )
        return await vector.create(documents)

    async def clean(self, dataset: KnowledgeDocument, node_ids: Optional[list[str]]):
        pass
