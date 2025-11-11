from app.core.rag.extractor.extractor_base import EntityExtractor
from app.core.rag.models.document import Document
from app.nlp import en_core_web_trf


class SpacyEntityExtractor(EntityExtractor):
    label_mapping = {}
    allowed = {
        "ORG": "organizations",
        "DATE": "dates",
        "PRODUCT": "products",
        "GPE": "geopolitical_entities",
        "EVENT": "events",
        "MONEY": "monetary_values",
        "PERSON": "people",
        "WORK_OF_ART": "artworks",
        "LANGUAGE": "languages",
        "LAW": "laws",
        "ORDINAL": "ordinals",
        "QUANTITY": "quantities",
        "TIME": "times",
    }

    def __init__(
            self,
            entities=[
                "organizations",
                "dates",
                "products",
                "events",
                "people",
                "times",
                "quantities",
            ],
    ):
        """Initialize with a list of allowed entity values."""
        self.entities = entities
        self.map_entities()

    def map_entities(self):
        """Map the provided entity values to their corresponding keys."""
        for value in self.entities:
            for key, mapped_value in self.allowed.items():
                if mapped_value == value:
                    self.label_mapping[key] = mapped_value

    def get_mapped_labels(self):
        """Return the mapped labels."""
        return self.label_mapping

    def extract(self, document: Document) -> Document:
        doc = en_core_web_trf(document.page_content)
        metadata = {label: set() for label in self.label_mapping.values()}
        # Populate the metadata with entities found in the text
        for ent in doc.ents:
            if ent.label_ in self.label_mapping:
                # Convert to lowercase and add to the set
                metadata[self.label_mapping[ent.label_]].add(ent.text.lower())

        # Convert sets back to lists for the final metadata
        document.entities.update({key: list(value) for key, value in metadata.items()})
        return document
