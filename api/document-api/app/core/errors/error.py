from typing import Optional

from app.exceptions import RapidaException


class LLMError(RapidaException):
    """Base class for all LLM exceptions."""
    description: Optional[str] = None

    def __init__(self, description: Optional[str] = None) -> None:
        self.description = description
        super(self, LLMError).__init__(message=description)


class LLMBadRequestError(LLMError):
    """Raised when the LLM returns bad request."""
    description = "Bad Request"


class ProviderTokenNotInitError(Exception):
    """
    Custom exception raised when the provider token is not initialized.
    """
    description = "Provider Token Not Init"

    def __init__(self, *args, **kwargs):
        self.description = args[0] if args else self.description


class QuotaExceededError(Exception):
    """
    Custom exception raised when the quota for a provider has been exceeded.
    """
    description = "Quota Exceeded"


class ModelCurrentlyNotSupportError(Exception):
    """
    Custom exception raised when the model not support
    """
    description = "Model Currently Not Support"
