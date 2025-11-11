# from difflib import SequenceMatcher
# from app.core.rag.extractor.extractor_base import BaseExtractor
# from unstructured.partition.pdf import partition_pdf
# from typing import List
# from textractor.data.constants import TextractFeatures
# from textractor import Textractor
# from textractor.entities.table_cell import TableCell, IS_MERGED_CELL
# from textractor.entities.table import Table as TextractTable
# from unstructured.chunking.title import chunk_by_title
# from unstructured.documents.elements import Element as UnstructuredElement

# from app.core.rag.models.document import Document as RapidaDocument
# import re 

# import config
# cfg = config.Config()
# import os
# import logging
# logger = logging.getLogger(__name__)
# import time
# import os

# class HybridPDFExtractor(BaseExtractor):
#     """
#     The HybridPDFExtractor class combines the capabilities of unstructured PDF extraction and AWS Textract 
#     to extract and analyze data from PDF documents. This class can process both simple text elements and 
#     table structures from PDFs, and employs text similarity measures to compare and update elements based 
#     on their contents. It supports initialization with a file path, partitioning of PDF content, starting 
#     Textract analysis jobs, waiting for job completion, retrieving cell content, cleaning text, comparing 
#     text similarity, and creating documents from processed chunks.
#     """

#     def __init__(
#             self,
#             file_path: str,
#     ):
#         """Initialize with file path."""
#         self._file_path = file_path

#     def start_document_analysis(self, extractor, _file_path):
#         """
#         Starts AWS Textract analysis on a document in an S3 bucket.

#         Parameters:
#             filename (str): The name of the file.

#         Returns:
#             str: The job ID for the analysis or None if an error occurred.
#         """
#         try:
#             response = extractor.start_document_analysis(
#                 file_source=_file_path,
#                 features=[TextractFeatures.TABLES],
#                 s3_upload_path=cfg.S3_TEXTRACT_BUCKET,
#             )
#             return response.job_id, response._api
#         except Exception as e:
#             logging.error(f"Error starting document analysis: {e}")


#     def wait_for_analysis(self, extractor, job_id, api):
#         """
#         Waits for AWS Textract analysis to complete.

#         Parameters:
#             job_id (str): The job ID for the analysis.

#         Returns:
#             dict: The job status or None if an error occurred.
#         """
#         while True:
#             try:
#                 job_status = extractor.get_result(job_id=job_id, api=api)
#                 status = job_status.response['JobStatus']

#                 if status in ['SUCCEEDED', 'FAILED']:
#                     break
                
#                 print('sleeping ..', status)
#                 time.sleep(5)
#             except Exception as e:
#                 logging.error(f"Error getting document analysis: {e}")

#         return job_status
    
#     # def get_cell_content(self, cell: TableCell):
#     #     """
#     #     Retrieve the content of a Textract table cell. If the cell is a merged cell, its content 
#     #     is compiled from its sibling cells. Otherwise, the content is derived directly from the 
#     #     cell's individual words.

#     #     Parameters:
#     #         cell (TableCell): The Textract table cell whose content needs to be extracted.

#     #     Returns:
#     #         str: A string representation of the cell's content.
#     #     """

#     #     # Check if the cell is a merged cell.
#     #     if cell.metadata.get(IS_MERGED_CELL, False):
#     #         # Create a dictionary with the row and column indices as keys and their associated words 
#     #         # sorted by x and y coordinates.
#     #         entities = {
#     #                     (cell.row_index, cell.col_index): sorted(
#     #                         cell.words, key=lambda x: (x.bbox.x, x.bbox.y)
#     #                     )
#     #                     for cell in sorted(
#     #                         cell.siblings, key=lambda x: (x.row_index, x.col_index)
#     #                     )
#     #                 }

#     #         # Get unique row and column indices for the merged cells.
#     #         rows = set(sorted([cell_key[0] for cell_key in entities.keys()]))
#     #         cols = set(sorted([cell_key[1] for cell_key in entities.keys()]))

#     #         entity_repr = []
#     #         # Iterate through each row.
#     #         for row in rows:
#     #             # Iterate through each column in the current row.
#     #             for col in cols:
#     #                 # Append a string representation of the words in the current cell.
#     #                 entity_repr.append(
#     #                     " ".join([entity.__repr__() for entity in entities[(row, col)]])
#     #                 )
#     #                 entity_repr.append(" ")
#     #             entity_repr.append("\n")
#     #         # Join all parts into a single string to form the cell content.
#     #         entity_repr = "".join(entity_repr)
        
#     #     # If the cell is not a merged cell.
#     #     else:
#     #         # Retrieve the words directly from the cell.
#     #         entities = cell.words
#     #         # Create a string representation of the words in the cell.
#     #         entity_repr = " ".join([entity.__repr__() for entity in entities])

#     #         # Form a detailed string representation of the cell including its position, span,
#     #         # if it's a column header, and if it's a merged cell.
#     #         entity_string = f"<Cell: ({cell.row_index},{cell.col_index}), Span: ({cell.row_span}, {cell.col_span}), Column Header: { cell.is_column_header}, "
#     #         entity_string += (
#     #             f"MergedCell: {cell.metadata.get(IS_MERGED_CELL, False)}>  " + entity_repr
#     #         )
        
#     #     # Return the string representation of the cell content.
#     #     return entity_repr

#     def get_cell_content(self, cell:TableCell):
#         if cell.metadata.get(IS_MERGED_CELL, False):
#             entities = {
#                         (cell.row_index, cell.col_index): sorted(
#                             cell.words, key=lambda x: (x.bbox.x, x.bbox.y)
#                         )
#                         for cell in sorted(
#                             cell.siblings, key=lambda x: (x.row_index, x.col_index)
#                         )
#                     }

#             rows = set(sorted([cell_key[0] for cell_key in entities.keys()]))
#             cols = set(sorted([cell_key[1] for cell_key in entities.keys()]))

#             entity_repr = []
#             for row in rows:
#                 for col in cols:
#                     entity_repr.append(
#                         " ".join([entity.__repr__() for entity in entities[(row, col)]])
#                     )
#                 #     entity_repr.append(" ")
#                 # entity_repr.append("\n")
#             entity_repr = "".join(entity_repr)
#         else:
#             entities = cell.words
#             entity_repr = " ".join([entity.__repr__() for entity in entities])

#             entity_string = f"<Cell: ({cell.row_index},{cell.col_index}), Span: ({cell.row_span}, {cell.col_span}), Column Header: { cell.is_column_header}, "
#             entity_string += (
#                 f"MergedCell: {cell.metadata.get(IS_MERGED_CELL, False)}>  " + entity_repr
#             )
#         # print("*****")
#         # print(entity_repr)
#         return entity_repr

#     def clean_text(self, text):
#         # Remove URLs
#         url_pattern = re.compile(r'http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\\(\\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+')
#         text = re.sub(url_pattern, ' ', text)

#         # Remove special characters, new lines, tabs, and extra spaces except parentheses
#         text = re.sub(r'[^\w\s()]', '', text)
#         # Remove extra whitespace
#         text = ' '.join(text.split())
#         return text.lower()


#     def text_similarity(self, text1, text2):
#         # Use SequenceMatcher to calculate similarity ratio
#         text1 = self.clean_text(text1)
#         text2 = self.clean_text(text2)

#         text1 = ' '.join(text1.split()).lower()
#         text2 = ' '.join(text2.split()).lower()

#         return round(SequenceMatcher(None, text1, text2).ratio(), 1)

#     def compare_and_update(self, unstructured_elements: List[UnstructuredElement], 
#                         textract_tables: List[TextractTable], 
#                         num_pages: int) -> List[UnstructuredElement]:
#         """
#         Compare and update unstructured elements with Textract table data based on text similarity.

#         This method iterates through the unstructured elements and Textract tables, comparing their 
#         text content using a similarity threshold. If the texts are similar enough, the text content 
#         of the unstructured elements is updated to match the Textract tables.

#         Parameters:
#             unstructured_elements (List[UnstructuredElement]): List of unstructured elements extracted 
#                                                                from the PDF.
#             textract_tables (List[TextractTable]): List of tables extracted using AWS Textract.
#             num_pages (int): Number of pages in the document.

#         Returns:
#             List[UnstructuredElement]: A list of tuples containing the original and updated text 
#                                        for elements that were changed.
#         """
        
#         num_changes = []
#         text_similarity_threshold = 0.7

#         for page in range(1, num_pages + 1):
#             # Filter unstructured elements and Textract tables by the current page
#             page_unstructured = [elem for elem in unstructured_elements if elem.metadata.page_number == page]
#             page_textract = [table for table in textract_tables if table.page == page]
            
#             for unstructured_elem in page_unstructured:
#                 # Iterate through each Textract table on the current page
#                 for textract_table in page_textract:
#                     # Check if the text similarity between the unstructured element and Textract table 
#                     # meets or exceeds the defined threshold
#                     if self.text_similarity(unstructured_elem.text, textract_table.text) >= text_similarity_threshold:
#                         # If the texts are similar enough, log this change by adding a tuple of the texts
#                         # to the list of changes made
#                         num_changes.append((unstructured_elem.text, textract_table.text))
                        
#                         # Extract and concatenate the text content of the Textract table's cells
#                         updated_text = self._extract_table_text(textract_table)
                        
#                         # Update the text of the unstructured element to match that of the Textract table
#                         unstructured_elem.text = updated_text  
#                         break  # Move to the next unstructured element
#         return num_changes

#     def _extract_table_text(self, textract_table: TextractTable) -> str:
#         """
#         Helper method to extract concatenated cell content from a Textract table.

#         Parameters:
#             textract_table (TextractTable): The Textract table to process.

#         Returns:
#             str: The concatenated cell content of the table.
#         """
#         rows_data = []
#         for _, rows in textract_table._get_table_cells(row_wise=True).items():
#             row_data = [self.get_cell_content(cell) for cell in rows]
#             rows_data.append(", ".join(row_data))
#         return "\n".join(rows_data)

#     def extract(self) -> list[RapidaDocument]:

        

#         os.environ["AWS_ACCESS_KEY_ID"] = cfg.S3_ACCESS_KEY
#         os.environ["AWS_SECRET_ACCESS_KEY"] = cfg.S3_SECRET_KEY

#         extractor = Textractor(region_name=cfg.S3_REGION)

#         # Partition the PDF located at the specified file path.
#         # The partition_pdf function extracts elements from the PDF, inferring table structures.
#         # The strategy parameter 'hi_res' is used for high-resolution processing.
#         # The languages parameter specifies that the text extraction should be in English.
#         elements: List[UnstructuredElement] = partition_pdf(
#                 filename=self._file_path,
#                 infer_table_structure=True,
#                 strategy='hi_res',
#                 languages=["eng"]
#         )

#         # Start the document analysis process using AWS Textract and get the job ID and API instance.
#         job_id, api = self.start_document_analysis(extractor, self._file_path)
        
#         # Wait for the document analysis to complete and get the job status.
#         job_status = self.wait_for_analysis(extractor, job_id, api)

#         if job_status:
#             # If the job status is not None, store the job status in the 'document' variable.
#             # The job_status is effectively the result of the document analysis.
#             document = job_status

#         unstructured_table_elements = []
#         unstructured_image_elements = []
#         unstructured_elements = []

#         for idx, element in enumerate(elements):
#             if element.category == "Table":
#                 unstructured_table_elements.append(element)
#                 unstructured_elements.append(element)
#             elif element.category == "Image":
#                 unstructured_image_elements.append(element)
#                 unstructured_elements.append(element)


#         # Compare and update the unstructured elements extracted from the PDF with
#         # the tables extracted by Textract using text similarity.
#         # The compare_and_update function aligns the content of the unstructured 
#         # elements and Textract tables if their text similarity is above a threshold.
#         _ = self.compare_and_update(unstructured_elements, document.tables, document.num_pages)
                
        
#         chunks = chunk_by_title(elements, max_characters=2000, combine_text_under_n_chars=2000)
        
#         documents = []
        
#         # Iterate through each chunk of text, strip any leading/trailing whitespace,
#         # and create a RapidaDocument instance with the chunk's text content.
#         for chunk in chunks:
#             text = chunk.text.strip()
#             documents.append(RapidaDocument(page_content=text))

#         return documents
