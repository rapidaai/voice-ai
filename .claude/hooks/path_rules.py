#!/usr/bin/env python3
"""Inject repository path rules before Claude edits a matching file."""

from __future__ import annotations

import fnmatch
import hashlib
import json
import os
import sys
from pathlib import Path

from session_state import first_delivery, stamp_directory


def repository_root() -> Path:
    configured = os.environ.get("CLAUDE_PROJECT_DIR")
    return Path(configured).resolve() if configured else Path(__file__).resolve().parents[2]


def matched_rules(root: Path, file_path: str) -> tuple[list[dict[str, object]], str]:
    target = Path(file_path)
    if not target.is_absolute():
        target = root / target
    try:
        relative = target.resolve().relative_to(root).as_posix()
    except ValueError:
        return [], ""

    registry = root / "agent-rules.json"
    document = json.loads(registry.read_text(encoding="utf-8"))
    rules = document.get("rules")
    if document.get("version") != 1 or not isinstance(rules, list):
        raise ValueError("agent-rules.json has an unsupported shape")

    matched = []
    for rule in rules:
        patterns = rule.get("paths") if isinstance(rule, dict) else None
        if not isinstance(patterns, list):
            raise ValueError("agent-rules.json contains a rule without paths")
        if any(isinstance(pattern, str) and fnmatch.fnmatchcase(relative, pattern) for pattern in patterns):
            matched.append(rule)
    return matched, relative


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        return 0
    tool_input = payload.get("tool_input") or {}
    file_path = tool_input.get("file_path") or ""
    if not isinstance(file_path, str) or not file_path:
        return 0

    root = repository_root()
    try:
        rules, relative = matched_rules(root, file_path)
        registry_bytes = (root / "agent-rules.json").read_bytes()
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"path rules unavailable: {error}", file=sys.stderr)
        return 0
    if not rules:
        return 0

    digest = hashlib.sha256(registry_bytes).hexdigest()[:12]
    session_id = payload.get("session_id")
    directory = stamp_directory("voice-ai-path-rules", session_id)
    fresh = [rule for rule in rules if first_delivery(directory, f"{rule.get('id')}.{digest}")]
    if not fresh:
        return 0

    sections = []
    for rule in fresh:
        instructions = rule.get("instructions")
        if not isinstance(instructions, list):
            continue
        body = "\n".join(f"- {instruction}" for instruction in instructions)
        sections.append(f"### {rule.get('title')}\n{body}")
    if not sections:
        return 0

    context = (
        f"Repository path rules for `{relative}` from `agent-rules.json`. "
        "Follow these rules for this edit.\n\n" + "\n\n".join(sections)
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
