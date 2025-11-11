# The above classes define configurations for file extensions and extractors in a Python application.
from typing import Optional, List

from pydantic import BaseModel, Field


# The class `ExtractorOptions` defines two optional attributes, `autodetect_encoding` and
# `extract_type`.
class ExtractorOptions(BaseModel):
    autodetect_encoding: Optional[bool] = None


class EntityExtractorOptions(BaseModel):
    entities: List[str]


class TransformerOptions(BaseModel):
    entity_extractor: str
    options: Optional[EntityExtractorOptions]


class TransformerConfig(BaseModel):
    transformer: str
    stage: str
    options: TransformerOptions


class FileExtensionConfig(BaseModel):
    # The code snippet you provided is defining a Pydantic model called `FileExtensionConfig` with three
    # attributes:
    extension: str  # Change to simple str
    extractor: str
    options: Optional[ExtractorOptions] = None


class EncoderOptions(BaseModel):
    model_name: str
    api_key: str


class ChunkingOptions(BaseModel):
    encoder: str
    options: EncoderOptions
    splitter: str  # Import path as a string
    threshold_adjustment: float
    dynamic_threshold: bool
    window_size: int
    min_split_tokens: int
    max_split_tokens: int
    split_tokens_tolerance: int
    plot_chunks: bool
    enable_statistics: bool


class ChunkingTechniqueConfig(BaseModel):
    chunker: str  # Import path as a string
    options: ChunkingOptions


class ExtractorConfig(BaseModel):
    # The line `file_extensions: List[FileExtensionConfig]` in the `ExtractorsConfig` class is
    # defining an attribute named `file_extensions` that is expected to be a list of
    # `FileExtensionConfig` objects. This means that the `file_extensions` attribute will hold a
    # collection of `FileExtensionConfig` instances, allowing multiple file extension configurations
    # to be stored and accessed within the `ExtractorsConfig` object.
    file_extensions: List[FileExtensionConfig] = Field(
        description="file extensions which is enable for document extraction"
    )

    transformers: List[TransformerConfig] = Field(
        description="transformers for chunked documents"
    )

    chunking_technique: Optional[ChunkingTechniqueConfig] = Field(None,
                                                                  description="chunking technique for your document")
