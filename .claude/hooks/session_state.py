"""Secure per-session delivery stamps for advisory Claude hooks."""

from __future__ import annotations

import os
import re
import stat
import tempfile
import hashlib
from pathlib import Path


def stamp_directory(namespace: str, session_id: object) -> Path | None:
    if not session_id:
        return None
    safe_namespace = re.sub(r"[^A-Za-z0-9_-]", "", namespace)
    safe_session = re.sub(r"[^A-Za-z0-9_-]", "", str(session_id))
    if not safe_namespace or not safe_session:
        return None
    base = Path(tempfile.gettempdir()) / f"{safe_namespace}-{os.getuid()}"
    session = base / safe_session
    try:
        for directory in (base, session):
            directory.mkdir(mode=0o700, exist_ok=True)
            details = directory.lstat()
            if not stat.S_ISDIR(details.st_mode) or details.st_uid != os.getuid() or details.st_mode & 0o077:
                return None
    except OSError:
        return None
    return session


def first_delivery(directory: Path | None, key: str) -> bool:
    if directory is None:
        return True
    safe_key = hashlib.sha256(key.encode("utf-8")).hexdigest()
    flags = os.O_CREAT | os.O_EXCL | os.O_WRONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        descriptor = os.open(directory / safe_key, flags, 0o600)
    except FileExistsError:
        return False
    except OSError:
        return True
    os.close(descriptor)
    return True
