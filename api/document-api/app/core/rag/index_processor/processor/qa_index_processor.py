"""Paragraph index processor."""

import logging
import re
import uuid
from typing import Optional

import pandas as pd

from app.core.cleaner.clean_processor import CleanProcessor
from app.core.rag.datasource.vdb.vector_factory import Vector
from app.core.rag.extractor.extract_processor import ExtractProcessor
from app.core.rag.index_processor.index_processor_base import BaseIndexProcessor
from app.core.rag.models.document import Document
from app.models.knowledge_model import KnowledgeDocument
from app.utils.general import generate_text_hash


class QAIndexProcessor(BaseIndexProcessor):
    def extract(
            self, knowledge_document: KnowledgeDocument, **kwargs
    ) -> list[Document]:
        text_docs = ExtractProcessor.extract(knowledge_document=knowledge_document)
        return text_docs

    def transform(
            self,
            knowledge_document: KnowledgeDocument,
            documents: list[Document],
            **kwargs,

    ) -> list[Document]:
        splitter = self._get_splitter(
            processing_rule=kwargs.get("process_rule"), model_manager=model_manager
        )

        # Split the text documents into nodes.
        all_documents = []
        all_qa_documents = []
        for document in documents:
            # document clean
            document_text = CleanProcessor.clean(
                document.page_content, kwargs.get("process_rule")
            )
            document.page_content = document_text

            # parse document to nodes
            document_nodes = splitter.split_documents([document])
            split_documents = []
            for document_node in document_nodes:

                if document_node.page_content.strip():
                    doc_id = str(uuid.uuid4())
                    hash = generate_text_hash(document_node.page_content)
                    document_node.metadata["doc_id"] = doc_id
                    document_node.metadata["doc_hash"] = hash
                    # delete Spliter character
                    page_content = document_node.page_content
                    if page_content.startswith(".") or page_content.startswith("。"):
                        page_content = page_content[1:]
                    else:
                        page_content = page_content
                    document_node.page_content = page_content
                    split_documents.append(document_node)
            all_documents.extend(split_documents)
        for i in range(0, len(all_documents), 10):
            threads = []
            sub_documents = all_documents[i: i + 10]
            for doc in sub_documents:
                self._format_qa_document(knowledge_document, doc, all_qa_documents)
        return all_qa_documents

    def format_by_template(self, file, **kwargs) -> list[Document]:

        # check file type
        if not file.filename.endswith(".csv"):
            raise ValueError("Invalid file type. Only CSV files are allowed")

        try:
            # Skip the first row
            df = pd.read_csv(file)
            text_docs = []
            for index, row in df.iterrows():
                data = Document(page_content=row[0], metadata={"answer": row[1]})
                text_docs.append(data)
            if len(text_docs) == 0:
                raise ValueError("The CSV file is empty.")

        except Exception as e:
            raise ValueError(str(e))
        return text_docs

    def load(self, knowledge_document: KnowledgeDocument, documents: list[Document]):
        vector = Vector(knowledge_document)
        vector.create(documents)

    def clean(
            self, knowledge_document: KnowledgeDocument, node_ids: Optional[list[str]]
    ):
        vector = Vector(knowledge_document)
        if node_ids:
            vector.delete_by_ids(node_ids)
        else:
            vector.delete()

    def _format_qa_document(
            self, knowledge_document: KnowledgeDocument, document_node, all_qa_documents
    ):
        format_documents = []
        if document_node.page_content is None or not document_node.page_content.strip():
            return
        try:
            response = self.generate_qa_document(
                knowledge_document, document_node.page_content
            )
            document_qa_list = self._format_split_text(response)
            qa_documents = []
            for result in document_qa_list:
                qa_document = Document(
                    page_content=result["question"],
                    metadata=document_node.metadata.copy(),
                )
                doc_id = str(uuid.uuid4())
                hash = generate_text_hash(result["question"])
                qa_document.metadata["answer"] = result["answer"]
                qa_document.metadata["doc_id"] = doc_id
                qa_document.metadata["doc_hash"] = hash
                qa_documents.append(qa_document)
            format_documents.extend(qa_documents)
        except Exception as e:
            logging.exception(e)

        all_qa_documents.extend(format_documents)

    def _format_split_text(self, text):
        regex = r"Q\d+:\s*(.*?)\s*A\d+:\s*([\s\S]*?)(?=Q\d+:|$)"
        matches = re.findall(regex, text, re.UNICODE)

        return [
            {"question": q, "answer": re.sub(r"\n\s*", "\n", a.strip())}
            for q, a in matches
            if q and a
        ]

    GENERATOR_QA_PROMPT = (
        "<Task> The user will send a long text. Generate a Question and Answer pairs only using the knowledge in the long text. Please think step by step."
        "Step 1: Understand and summarize the main content of this text.\n"
        "Step 2: What key information or concepts are mentioned in this text?\n"
        "Step 3: Decompose or combine multiple pieces of information and concepts.\n"
        "Step 4: Generate questions and answers based on these key information and concepts.\n"
        "<Constraints> The questions should be clear and detailed, and the answers should be detailed and complete. "
        "You must answer in {language}, in a style that is clear and detailed in {language}. No language other than {language} should be used. \n"
        "<Format> Use the following format: Q1:\nA1:\nQ2:\nA2:...\n"
        "<QA Pairs>"
    )

    def generate_qa_document(self, knowledge_document: KnowledgeDocument, query):
        prompt = self.GENERATOR_QA_PROMPT.format(language=knowledge_document.language)

        # model_manager = ModelManager()
        # model_instance = model_manager.get_default_model_instance(
        #     tenant_id=tenant_id,
        #     model_type=ModelType.LLM,
        # )
        #
        # prompt_messages = [
        #     SystemPromptMessage(content=prompt),
        #     UserPromptMessage(content=query)
        # ]

        # response = model_instance.invoke_llm(
        #     prompt_messages=prompt_messages,
        #     model_parameters={
        #         'temperature': 0.01,
        #         "max_tokens": 2000
        #     },
        #     stream=False
        # )

        # answer = response.message.content
        # return answer.strip()
        return ""
