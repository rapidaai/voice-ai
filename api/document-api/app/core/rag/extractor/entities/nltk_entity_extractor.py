from app.core.rag.extractor.extractor_base import EntityExtractor
from app.core.rag.models.document import Document
import nltk
from nltk import ne_chunk, pos_tag, word_tokenize
from nltk.tree import Tree

# Download necessary NLTK resources
nltk.download("punkt")
nltk.download("averaged_perceptron_tagger")
nltk.download("maxent_ne_chunker")
nltk.download("words")
# Load the SpaCy model
# do not get confuse with faster and accrate


class NltkEntityExtractor(EntityExtractor):
    label_mapping = {
        "GPE": "Geopolitical Entities",
        "ORGANIZATION": "Organizations",
        "PERSON": "People",
        "DATE": "Dates",
        "EVENT": "Events",
        "MONEY": "Monetary Values",
        "NORP": "Nationalities or Religious/Political Groups",
        "PRODUCT": "Products",
        "WORK_OF_ART": "Artworks",
        "LANGUAGE": "Languages",
        "LAW": "Laws",
        "ORDINAL": "Ordinals",
        "QUANTITY": "Quantities",
        "TIME": "Times",
    }

    def __init__(self, sentence: str):
        """Initialize with file path."""
        self._sentence = sentence

    def extract(self, document: Document) -> Document:
        tokens = word_tokenize(document.page_content)
        tagged = pos_tag(tokens)
        tree = ne_chunk(tagged)

        # Create a mapping of NLTK labels to user-friendly labels

        # Initialize metadata structure
        metadata = {label: [] for label in self.label_mapping.values()}

        # Populate the metadata with entities found in the text
        for subtree in tree:
            if isinstance(subtree, Tree):
                label = subtree.label()
                if label in self.label_mapping:
                    metadata[self.label_mapping[label]].append(
                        " ".join(word for word, _ in subtree.leaves())
                    )

        document.entities.update(metadata)
        return document
