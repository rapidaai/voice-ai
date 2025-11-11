from enum import Enum


class IndexType(Enum):
    PARAGRAPH_INDEX = "paragraph-index"
    QA_INDEX = "qa-index"
    PARENT_CHILD_INDEX = "parent-child-index"
    SUMMARY_INDEX = "summary-index"
