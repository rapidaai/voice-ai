from typing import Optional
from app.configs.extractor_config import ChunkingTechniqueConfig, EncoderOptions
from app.core.chunkers.base_chunker import BaseChunker
from semantic_router.encoders import BaseEncoder
from app.core.splitter.base_splitter import BaseSplitter
from app.utils.general import dynamic_class_import


class ChunkerFactory:

    # The line `config: Optional[ChunkingTechniqueConfig]` in the `ChunkerFactory` class is defining a
    # class variable `config` with type hinting.
    config: Optional[ChunkingTechniqueConfig]

    def __init__(self, cfg: ChunkingTechniqueConfig):
        """
        The function `__init__` initializes an object with a `ChunkingTechniqueConfig` configuration.
        :param cfg: The `cfg` parameter is an instance of the `ChunkingTechniqueConfig` class, which is
        used to configure the chunking technique for the class or method where it is being passed as an
        argument
        :type cfg: ChunkingTechniqueConfig
        """
        self.config = cfg

    def __create_encoder(
        self, encoder: str, encoder_config: EncoderOptions
    ) -> BaseEncoder:
        # Dynamically import the encoder class based on the type
        encoder_class = dynamic_class_import(encoder)
        # Create and return an instance of the encoder
        encoder_instance = encoder_class(**encoder_config.model_dump())
        return encoder_instance

    def __create_splitter(self, splitter: str) -> BaseSplitter:
        splitter_class = dynamic_class_import(splitter)
        splitter_instance = splitter_class()
        return splitter_instance

    # Function to create a chunker instance
    def __create_chunker(self, chunking_config: ChunkingTechniqueConfig) -> BaseChunker:
        # Dynamically import the chunker class based on the type
        chunker_class = dynamic_class_import(chunking_config.chunker)
        # Create and return an instance of the chunker
        chunker_instance = chunker_class(
            encoder=self.__create_encoder(
                chunking_config.options.encoder, chunking_config.options.options
            ),
            splitter=self.__create_splitter(chunking_config.options.splitter),
            **chunking_config.options.model_dump(
                exclude={"encoder", "options", "splitter"}
            )
        )
        return chunker_instance

    def get(self) -> "BaseChunker":
        return self.__create_chunker(self.config)
