"""
author: prashant.srivastav
"""

from app.exceptions import RapidaException


class DocumentIsPausedException(RapidaException):
    pass


class DocumentIsDeletedPausedException(RapidaException):
    pass


class DocumentNotFoundException(RapidaException):
    pass
