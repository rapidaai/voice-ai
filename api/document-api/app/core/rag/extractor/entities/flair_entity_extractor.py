from app.core.rag.extractor.extractor_base import EntityExtractor
from app.core.rag.models.document import Document
from flair.data import Sentence
from flair.models import SequenceTagger

# Load the Flair NER tagger
tagger = SequenceTagger.load("ner")
# Load the SpaCy model
# do not get confuse with faster and accrate


class FlairEntityExtractor(EntityExtractor):
    label_mapping = {
        "B-ORG": "Organizations",
        "I-ORG": "Organizations",
        "B-DATE": "Dates",
        "I-DATE": "Dates",
        "B-PRODUCT": "Products",
        "I-PRODUCT": "Products",
        # Add more labels if needed
    }

    def __init__(self, sentence: str):
        """Initialize with file path."""
        self._sentence = sentence

    def extract(self, document: Document) -> Document:
        sentence = Sentence(document.page_content)
        tagger.predict(sentence)

        # Initialize metadata structure
        metadata = {label: [] for label in self.label_mapping.values()}

        # Populate the metadata with entities found in the text
        for entity in sentence.get_spans("ner"):
            metadata[self.label_mapping[entity.tag]].append(entity.text)
        document.entities.update(metadata)
        return document
