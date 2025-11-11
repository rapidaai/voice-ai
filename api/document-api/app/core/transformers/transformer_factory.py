from typing import List
from app.configs.extractor_config import TransformerConfig
from app.core.rag.extractor.extractor_base import EntityExtractor
from app.core.transformers.document_transformer import BaseDocumentTransformer
from app.utils.general import dynamic_class_import


def initialize_transformers(
    config: List[TransformerConfig], stage: str = None
) -> List[BaseDocumentTransformer]:
    initialized_transformers: List[BaseDocumentTransformer] = []

    # Filter transformers based on the provided stage ("pre" or "post")
    transformers_to_initialize = [
        t for t in config if stage is None or t.stage == stage
    ]

    for transformer in transformers_to_initialize:
        # Dynamically import and initialize the transformer class
        transformer_class = dynamic_class_import(transformer.transformer)

        # Dynamically import and initialize the extractor class
        extractor_class: EntityExtractor = dynamic_class_import(
            transformer.options.entity_extractor
        )
        extractor_instance = extractor_class(**transformer.options.options.model_dump())

        # Initialize the transformer class with the extractor instance
        transformer_instance = transformer_class(entity_extractor=extractor_instance)

        # Append initialized transformer to the list
        initialized_transformers.append(transformer_instance)

    return initialized_transformers
