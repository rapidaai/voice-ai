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
import json
import pickle
from typing import Union, Any, Dict

from fastapi.encoders import jsonable_encoder
from sqlalchemy import Column, BigInteger, String, DateTime, func, Float, Text, ForeignKey, JSON, Boolean, text, \
    LargeBinary
from sqlalchemy.orm import declarative_base

from app.models import StringUUID

Base = declarative_base()


class CustomJSONEncoder:
    @staticmethod
    def encode_bigint(obj: Any) -> Union[str, Any]:
        print(f"{obj} => {type(obj)}")
        if isinstance(obj, dict):
            # Loop through dictionary keys and check for BigInteger
            for key, value in obj.items():
                print(f"{key} => {type(value)}")
                if isinstance(value, int):
                    # Convert BigInteger to string
                    obj[key] = str(value)
        return obj

    @classmethod
    def jsonable_encoder(cls, obj: Any, **kwargs: Any) -> Dict[str, Any]:
        encoded_obj = jsonable_encoder(obj, **kwargs)
        encoded_obj = cls.encode_bigint(encoded_obj)
        return encoded_obj


class PostgresModel(Base):
    __abstract__ = True

    def to_dict(self):
        return CustomJSONEncoder.jsonable_encoder(self)


class Audited(PostgresModel):
    """
    A common definition for all the sql model in postgres replicated from go-service
    """
    __abstract__ = True
    created_date = Column(DateTime, nullable=False, default=func.current_timestamp())
    updated_date = Column(DateTime, nullable=True, default=func.current_timestamp(),
                          onupdate=func.current_timestamp())


class Knowledge(Audited):
    """
    Do not change that knowledge
    """
    __tablename__ = 'knowledges'

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    project_id = Column(BigInteger, nullable=False)
    organization_id = Column(BigInteger, nullable=False)

    embedding_model_provider_name = Column(BigInteger, nullable=False)
    embedding_model_provider_id = Column(BigInteger, nullable=False)
    # migrate with following
    # f'vs__{self.organization_id}__{self.project_id}__{self.knowledge_id}__{provider_model_id}'
    storage_namespace = Column(String(400), nullable=False)


class KnowledgeEmbeddingModelOption(Audited):
    __tablename__ = 'knowledge_embedding_model_options'

    id = Column(BigInteger, primary_key=True, autoincrement=True, nullable=False)
    status = Column(String(50), nullable=False, default='ACTIVE')
    created_by = Column(BigInteger, nullable=False)
    updated_by = Column(BigInteger, nullable=True)
    key = Column(String(200), nullable=False)
    value = Column(Text, nullable=False)
    knowledge_id = Column(BigInteger, ForeignKey("knowledges.id"), nullable=False)


class KnowledgeDocumentProcessRule:
    PRE_PROCESSING_RULES = ['remove_stopwords', 'remove_extra_spaces', 'remove_urls_emails']
    AUTOMATIC_RULES = {
        'pre_processing_rules': [
            {'id': 'remove_extra_spaces', 'enabled': True},
            {'id': 'remove_urls_emails', 'enabled': False}
        ],
        'segmentation': {
            'separator': '\n',
            'max_chunk_size': 500,
            'chunk_overlap': 50
        }
    }

    @property
    def is_automatic(self) -> bool:
        """
        if preprocessing is automated
        :return:
        """
        return True

    @property
    def is_semantic(self) -> bool:
        return True

    @property
    def rule_dict(self):
        return self.AUTOMATIC_RULES


class KnowledgeDocument(Audited):
    """
    Do not change this model as same model is getting used in workflow-service to create knowledge documents
    Only read and update / as select query is not transactional can be locked for update.

    """
    __tablename__ = 'knowledge_documents'

    id: int = Column(BigInteger, primary_key=True, autoincrement=True)
    knowledge_id = Column(BigInteger, ForeignKey("knowledges.id"), nullable=False)
    project_id = Column(BigInteger, nullable=False)
    organization_id = Column(BigInteger, nullable=False)
    language = Column(String(50), default='english')
    name = Column(String)
    description = Column(String)
    document_size = Column(BigInteger, nullable=False)
    index_status = Column(String(50), nullable=False, default='pending')
    status = Column(String(50), nullable=False, default='active')
    retrieval_count = Column(BigInteger, default=0)
    token_count = Column(BigInteger, default=0)
    word_count = Column(BigInteger, default=0)
    created_by = Column(BigInteger, nullable=False)
    updated_by = Column(BigInteger)

    indexing_latency = Column(Float, nullable=True)
    completed_at = Column(DateTime, nullable=True)

    error = Column(Text, nullable=True)
    parsing_completed_at = Column(DateTime, nullable=True)
    processing_started_at = Column(DateTime, nullable=True)
    cleaning_completed_at = Column(DateTime, nullable=True)
    splitting_completed_at = Column(DateTime, nullable=True)

    #
    index_struct = Column(Text, nullable=True)
    document_source = Column(String, nullable=False)

    class Config:
        orm_mode = True

    # Getter methods
    def _get_json_data(self):
        # Parse JSON string into a dictionary
        return json.loads(self.document_source)

    @property
    def complete_path(self):
        data = self._get_json_data()
        return data.get("completePath")

    @property
    def document_url(self):
        data = self._get_json_data()
        return data.get("documentUrl")

    @property
    def mime_type(self):
        data = self._get_json_data()
        return data.get("mimeType")

    @property
    def source(self):
        data = self._get_json_data()
        return data.get("source")

    @property
    def storage(self):
        data = self._get_json_data()
        return data.get("storage")

    @property
    def type(self):
        data = self._get_json_data()
        return data.get("type")

    @property
    def index_struct_dict(self):
        return json.loads(self.index_struct) if self.index_struct else None

    @property
    def process_rule(self) -> KnowledgeDocumentProcessRule:
        return KnowledgeDocumentProcessRule()

    # def collection_name(self, provider_model_id) -> str:
    #     """
    #     Generate a unique vector store for given organization and knowledge id
    #     this should also have model id
    #     :return:
    #     """
    #     return f'vs__{self.organization_id}__{self.project_id}__{self.knowledge_id}__{provider_model_id}'


class KnowledgeDocumentSegment(Audited):
    __tablename__ = 'knowledge_document_segments'

    # initial fields
    id = Column(BigInteger, primary_key=True, autoincrement=True)
    knowledge_id = Column(BigInteger, nullable=False)
    knowledge_document_id: int = Column(BigInteger, nullable=False)
    position = Column(BigInteger, nullable=False)
    content = Column(Text, nullable=False)
    answer = Column(Text, nullable=True)
    word_count = Column(BigInteger, nullable=False)
    token_count = Column(BigInteger, nullable=False)
    hit_count = Column(BigInteger, nullable=False, default=0)

    # indexing fields
    keywords = Column(JSON, nullable=True)
    index_node_id = Column(String(255), nullable=True)
    index_node_hash = Column(String(255), nullable=True)

    # basic fields
    enabled = Column(Boolean, nullable=False,
                     server_default=text('true'))
    disabled_at = Column(DateTime, nullable=True)
    disabled_by = Column(StringUUID, nullable=True)
    status = Column(String(255), nullable=False,
                    server_default=text("'waiting'::character varying"))

    indexing_at = Column(DateTime, nullable=True)
    completed_at = Column(DateTime, nullable=True)
    error = Column(Text, nullable=True)
    stopped_at = Column(DateTime, nullable=True)


class KnowledgeDocumentEmbedding(Audited):
    __tablename__ = 'knowledge_document_embeddings'

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    hash = Column(String(64), nullable=False)
    base64 = Column(Text, nullable=False)
    embedding = Column(LargeBinary, nullable=False)
    embedding_model_name = Column(String(100), nullable=False)
    embedding_model_provider_name = Column(String(100), nullable=False)

    def set_embedding(self, embedding_data: list[float]):
        self.embedding = pickle.dumps(embedding_data, protocol=pickle.HIGHEST_PROTOCOL)

    def get_embedding(self) -> list[float]:
        return pickle.loads(self.embedding)
