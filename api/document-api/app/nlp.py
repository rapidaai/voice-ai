import spacy
import nltk
import torch

# Load the SpaCy model
# do not get confuse with faster and accrate
en_core_web_trf = spacy.load("en_core_web_trf")
en_core_web_sm = spacy.load("en_core_web_sm")

torch.set_num_threads(1)


def init_model():
    nltk.download("punkt")
