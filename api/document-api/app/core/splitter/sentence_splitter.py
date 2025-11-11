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

from typing import List
import regex
import nltk
from app.core.splitter.base_splitter import BaseSplitter
from app.nlp import en_core_web_sm


class SentenceRegexSplitter(BaseSplitter):
    """
    Enhanced regex pattern to split a given text into sentences more accurately.

    The enhanced regex pattern includes handling for:
    - Direct speech and quotations.
    - Abbreviations, initials, and acronyms.
    - Decimal numbers and dates.
    - Ellipses and other punctuation marks used in informal text.
    - Removing control characters and format characters.
    """

    regex_pattern = r"""
        # Negative lookbehind for word boundary, word char, dot, word char
        (?<!\b\w\.\w.)
        # Negative lookbehind for single uppercase initials like "A."
        (?<!\b[A-Z][a-z]\.)
        # Negative lookbehind for abbreviations like "U.S."
        (?<!\b[A-Z]\.)
        # Negative lookbehind for abbreviations with uppercase letters and dots
        (?<!\b\p{Lu}\.\p{Lu}.)
        # Negative lookbehind for numbers, to avoid splitting decimals
        (?<!\b\p{N}\.)
        # Positive lookbehind for punctuation followed by whitespace
        (?<=\.|\?|!|:|\.\.\.)\s+
        # Positive lookahead for uppercase letter or opening quote at word boundary
        (?="?(?=[A-Z])|"\b)
        # OR
        |
        # Splits after punctuation that follows closing punctuation, followed by
        # whitespace
        (?<=[\"\'\]\)\}][\.!?])\s+(?=[\"\'\(A-Z])
        # OR
        |
        # Splits after punctuation if not preceded by a period
        (?<=[^\.][\.!?])\s+(?=[A-Z])
        # OR
        |
        # Handles splitting after ellipses
        (?<=\.\.\.)\s+(?=[A-Z])
        # OR
        |
        # Matches and removes control characters and format characters
        [\p{Cc}\p{Cf}]+
    """

    def __call__(self, doc: str) -> List[str]:
        sentences = regex.split(self.regex_pattern, doc, flags=regex.VERBOSE)
        sentences = [sentence.strip() for sentence in sentences if sentence.strip()]
        return sentences


class SpacySentenceSplitter(BaseSplitter):
    def __call__(self, doc: str) -> List[str]:
        # Pass the document to the base class's method for potential preprocessing
        # Note: This may not be necessary if the base class does not alter the input
        doc = en_core_web_sm(doc)
        # Extract and return the text of each identified sentence from the processed document
        return [sent.text for sent in doc.sents]
        # In the current implementation, this method does not leverage spaCy's capabilities,
        # but could be modified to integrate spaCy's NLP features if needed.


class NLTKSetneceSplitter(BaseSplitter):
    def __call__(self, doc: str) -> List[str]:
        # Process the input document with the NLP model from NLTK
        # The `nlp` function tokenizes the text and identifies sentence boundaries
        return nltk.sent_tokenize(doc)

        # Note: Ensure that `nlp` is defined (e.g., through `nltk.data.load`) before calling this method.
