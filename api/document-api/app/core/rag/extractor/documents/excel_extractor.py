"""Abstract interface for document loader implementations."""
from typing import Optional

import pandas as pd
import xlrd

from app.core.rag.extractor.extractor_base import BaseExtractor
from app.core.rag.models.document import Document


class ExcelExtractor(BaseExtractor):
    """Load Excel files.


    Args:
        file_path: Path to the file to load.
    """

    def __init__(
            self,
            file_path: str,
            encoding: Optional[str] = None,
            autodetect_encoding: bool = False
    ):
        """Initialize with file path."""
        self._file_path = file_path
        self._encoding = encoding
        self._autodetect_encoding = autodetect_encoding

    def extract(self) -> list[Document]:
        """ parse excel file"""
        if self._file_path.endswith('.xls'):
            return self._extract4xls()
        elif self._file_path.endswith('.xlsx'):
            return self._extract4xlsx()

    def _extract4xls(self) -> list[Document]:
        wb = xlrd.open_workbook(filename=self._file_path)
        documents = []
        # loop over all sheets
        for sheet in wb.sheets():
            row_header = None
            for row_index, row in enumerate(sheet.get_rows()):
                if self.is_blank_row(row):
                    continue
                if row_header is None:
                    row_header = row
                    continue
                    
                item_arr = []
                for index, cell in enumerate(row):
                    if index < len(row_header):  # Ensure header index exists
                        txt_value = str(cell.value).strip()
                        header_value = str(row_header[index].value).strip()
                        if txt_value and header_value:  # Only add non-empty values
                            item_arr.append(f'{header_value}:{txt_value}')
                
                if item_arr:  # Only create document if we have content
                    item_str = "\n".join(item_arr)
                    document = Document(page_content=item_str, metadata={'source': self._file_path})
                    documents.append(document)
        
        return documents

    def _extract4xlsx(self) -> list[Document]:
        """Load from file path using Pandas."""
        data = []
        # Read each worksheet of an Excel file using Pandas
        xls = pd.ExcelFile(self._file_path)
        for sheet_name in xls.sheet_names:
            df = pd.read_excel(xls, sheet_name=sheet_name)

            # filter out rows with all NaN values
            df.dropna(how='all', inplace=True)

            # transform each row into a Document
            for _, row in df.iterrows():
                item = ';'.join(f'{k}:{v}' for k, v in row.items() if pd.notna(v))
                document = Document(page_content=item, metadata={'source': self._file_path})
                data.append(document)
        return data

    @staticmethod
    def is_blank_row(row):
        """

        Determine whether the specified line is a blank line.
        :param row: row object。
        :return: Returns True if the row is blank, False otherwise.
        """
        # Iterates through the cells and returns False if a non-empty cell is found
        for cell in row:
            if cell.value is not None and cell.value != '':
                return False
        return True
