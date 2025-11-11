from enum import Enum


class Field(str, Enum):
    VECTOR_KEY = "vector"
    SOURCE_KEY = "source"
    TEXT_KEY = "text"
    DOCUMENT_ID_KEY = "document_id"
    DOCUMENT_HASH_KEY = "document_hash"
    KNOWLEDGE_DOCUMENT_ID_KEY = "knowledge_document_id"
    KNOWLEDGE_ID_KEY = "knowledge_id"
    PROJECT_ID_KEY = "project_id"
    ORGANIZATION_ID_KEY = "organization_id"
    METADATA_KEY = "metadata"
    ENTITIES_KEY = "entities"


class VectorType(str, Enum):
    WEAVIATE = "weaviate"
    OPENSEARCH = "opensearch"
