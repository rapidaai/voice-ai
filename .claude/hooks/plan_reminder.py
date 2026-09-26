#!/usr/bin/env python3
"""Remind Claude to establish the lifecycle plan before the first source edit."""

from __future__ import annotations

import hashlib
import json
import os
import sys
from pathlib import Path

from session_state import first_delivery, stamp_directory


SOURCE_SUFFIXES = {
    ".css",
    ".go",
    ".html",
    ".js",
    ".json",
    ".jsx",
    ".proto",
    ".py",
    ".scss",
    ".sh",
    ".sql",
    ".ts",
    ".tsx",
    ".yaml",
    ".yml",
}


def repository_root() -> Path:
    configured = os.environ.get("CLAUDE_PROJECT_DIR")
    return Path(configured).resolve() if configured else Path(__file__).resolve().parents[2]


def source_path(root: Path, value: object) -> str | None:
    if not isinstance(value, str) or not value:
        return None
    target = Path(value)
    if not target.is_absolute():
        target = root / target
    try:
        relative = target.resolve().relative_to(root)
    except ValueError:
        return None
    is_extensionless_script = relative.suffix == "" and relative.parts[:1] in {("bin",), ("githooks",)}
    if relative.suffix.lower() not in SOURCE_SUFFIXES and not is_extensionless_script:
        return None
    return relative.as_posix()


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        return 0
    relative = source_path(repository_root(), (payload.get("tool_input") or {}).get("file_path"))
    if relative is None:
        return 0

    try:
        policy = (repository_root() / "AGENTS.md").read_bytes()
    except OSError as error:
        print(f"plan reminder unavailable: {error}", file=sys.stderr)
        return 0
    digest = hashlib.sha256(policy).hexdigest()[:12]
    directory = stamp_directory("voice-ai-plan-reminder", payload.get("session_id"))
    if not first_delivery(directory, f"source-edit.{digest}"):
        return 0

    context = (
        f"Plan check before editing `{relative}`:\n"
        "- Confirm the Fast, Standard, or Governed tier.\n"
        "- Fast work needs a concise change contract.\n"
        "- Standard work needs agreed acceptance criteria, scope, ownership, risks, and verification.\n"
        "- Governed implementation must not begin before the approved exact-digest gate.\n"
        "If this evidence is absent, pause the edit and establish it first."
    )
    json.dump(
        {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "additionalContext": context,
            },
            "suppressOutput": True,
        },
        sys.stdout,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
