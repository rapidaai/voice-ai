from hashlib import sha256
import importlib
import re
from typing import Union

import jwt
import tiktoken


def generate_text_hash(text: str) -> str:
    """

    Args:
        text:

    Returns:

    """
    hash_text = str(text) + "None"
    return sha256(hash_text.encode()).hexdigest()


def generate_jwt_token(
    project_id: Union[int, None],
    organization_id: int,
    user_id: Union[int, None] = None,
    key: str = "rpd_pks",
    algo: str = "HS256",
) -> str:
    """

    Returns:

    """
    payload = {"organizationId": str(organization_id)}
    if user_id:
        payload["userId"] = str(user_id)

    if project_id:
        payload["projectId"] = str(project_id)

    return jwt.encode(
        payload=payload,
        key=key,
        algorithm=algo,
    )


def count_string_token(prompt: str, model: str = None) -> int:
    """
    Returns the number of tokens in a (prompt or completion) text string.

    Args:
        prompt (str): The text string
        model_name (str): The name of the encoding to use. (e.g., "gpt-3.5-turbo")

    Returns:
        int: The number of tokens in the text string.
    """
    try:
        if model:
            model = model.lower()
            return len(tiktoken.encoding_for_model(model).encode(prompt))
    except KeyError:
        pass

    return len(tiktoken.get_encoding("cl100k_base").encode(prompt))


def count_string_word(text: str):
    # Regular expression to match words, allowing apostrophes for contractions and hyphens in words
    words = re.findall(
        r"\b[\w\']+\b", text.lower()
    )  # Convert to lower case for uniformity
    return len(words)


def dynamic_class_import(class_path: str):
    # This code snippet is a function called `dynamic_class_import` that dynamically imports a class
    # based on the provided `class_path`. Here's a breakdown of what each line is doing:
    module_path, class_name = class_path.rsplit(".", 1)
    module = importlib.import_module(module_path)
    return getattr(module, class_name)
