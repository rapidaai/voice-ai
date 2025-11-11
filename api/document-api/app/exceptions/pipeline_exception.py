from app.exceptions.rapida_exception import RapidaException


class IllegalDocumentExtractorConfigException(RapidaException):
    """
    Exception raised when no suitable extractor configuration is found for a given document type or extension.
    """

    def __init__(
            self,
            message: str = "Illegal or missing document extractor configuration.",
            status_code: int = 400,
            error_code: int = 1001,
            error_prefix: str = "RAPIDA",
            service_code: str = "KN_PIPELINE",
    ):
        super().__init__(
            status_code=status_code,
            message=message,
            error_code=error_code,
            error_prefix=error_prefix,
            service_code=service_code,
        )


class VectorDatabaseIndexingException(RapidaException):
    pass
