#!/usr/bin/env python3
"""Gate pull request publication on body policy and repository checks."""

from __future__ import annotations

import json
import os
import re
import shlex
import subprocess
import sys
import tempfile
from pathlib import Path


GATED = {"create", "edit", "ready"}
INLINE_BODY = {"--body", "-b", "--fill", "-f", "--fill-first", "--fill-verbose", "--web", "-w"}
GATED_COMMAND = re.compile(r"(?:^|[\s\"'])(?:\S*/)?gh\s+pr\s+(?:create|edit|ready)\b")


def repository_root() -> Path:
    configured = os.environ.get("CLAUDE_PROJECT_DIR")
    return Path(configured).resolve() if configured else Path(__file__).resolve().parents[2]


def block(message: str) -> None:
    print(f"PR gate: {message}", file=sys.stderr)
    raise SystemExit(2)


def invocations(command: str) -> list[tuple[str, list[str]]]:
    lexer = shlex.shlex(command.replace("\n", ";"), posix=True, punctuation_chars=";&|")
    lexer.whitespace_split = True
    try:
        words = list(lexer)
    except ValueError:
        if re.search(r"\bgh\s+pr\s+(create|edit|ready)\b", command):
            block("cannot safely parse a pull request command")
        return []

    calls: list[tuple[str, list[str]]] = []
    segment: list[str] = []
    for word in words + [";"]:
        if word and set(word) <= set(";&|"):
            for index in range(len(segment) - 2):
                if os.path.basename(segment[index]) == "gh" and segment[index + 1] == "pr":
                    action = segment[index + 2]
                    if action in GATED:
                        calls.append((action, segment[index + 3 :]))
                    break
            segment = []
            continue
        segment.append(word)
    return calls


def flag_value(arguments: list[str], *names: str) -> str | None:
    for index, argument in enumerate(arguments):
        for name in names:
            if argument == name and index + 1 < len(arguments):
                return arguments[index + 1]
            if argument.startswith(name + "="):
                return argument.split("=", 1)[1]
    return None


def validate_call(root: Path, cwd: Path, action: str, arguments: list[str]) -> str | None:
    if action in {"create", "edit"}:
        inline = next((item for item in arguments if item.split("=", 1)[0] in INLINE_BODY), None)
        if inline is not None:
            return f"gh pr {action} must use --body-file, not {inline}"

    body_file = flag_value(arguments, "--body-file", "-F")
    title = flag_value(arguments, "--title", "-t")
    if action == "create" and body_file is None:
        return "gh pr create requires --body-file based on .github/PULL_REQUEST_TEMPLATE.md"
    if action == "create" and title is None:
        return "gh pr create requires an explicit Conventional Commit title"
    if body_file == "-":
        return "pull request bodies from stdin cannot be validated before publication"
    if body_file is not None:
        body_path = Path(body_file).expanduser()
        if not body_path.is_absolute():
            body_path = cwd / body_path
        try:
            body_path = body_path.resolve()
            body_path.relative_to(root)
        except (OSError, ValueError):
            return "the pull request body file must be inside the repository"
        command = [sys.executable, str(root / "bin" / "check-pr-body"), "--body-file", str(body_path)]
        command.extend(["--title", title or "chore: retain current pull request title"])
        result = subprocess.run(command, cwd=root, capture_output=True, check=False, text=True, timeout=30)
        if result.returncode != 0:
            return (result.stdout + result.stderr).strip()
        with tempfile.TemporaryDirectory(prefix="voice-ai-pr-gate-") as directory:
            filtered = Path(directory) / "body.md"
            result = subprocess.run(
                [str(root / "bin" / "agent-egress"), "text", str(body_path), str(filtered)],
                cwd=root,
                capture_output=True,
                check=False,
                text=True,
                timeout=30,
            )
            if result.returncode != 0:
                return (result.stdout + result.stderr).strip()
            manifest = json.loads(filtered.with_name(filtered.name + ".manifest.json").read_text(encoding="utf-8"))
            if manifest.get("redacted_lines"):
                return "pull request body contains secret-looking content"
    elif title is not None:
        result = subprocess.run(
            [sys.executable, str(root / "bin" / "check-pr-body"), "--title-only", "--title", title],
            cwd=root,
            capture_output=True,
            check=False,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            return (result.stdout + result.stderr).strip()
    return None


def main() -> int:
    raw = sys.stdin.read()
    try:
        payload = json.loads(raw)
        command = (payload.get("tool_input") or {}).get("command") or ""
    except (json.JSONDecodeError, AttributeError):
        if re.search(r"gh\s+pr\s+(create|edit|ready)", raw):
            block("cannot parse a payload containing a pull request command")
        return 0
    if not isinstance(command, str):
        return 0

    calls = invocations(command)
    if not calls:
        if GATED_COMMAND.search(command):
            block("cannot safely inspect a nested or indirect pull request command")
        return 0
    root = repository_root()
    cwd = Path((payload.get("tool_input") or {}).get("cwd") or root).resolve()
    base = "main"
    for action, arguments in calls:
        error = validate_call(root, cwd, action, arguments)
        if error:
            block(error)
        selected_base = flag_value(arguments, "--base", "-B")
        if selected_base:
            base = selected_base

    result = subprocess.run(
        [str(root / "bin" / "agent-pr-ready"), base],
        cwd=root,
        capture_output=True,
        check=False,
        text=True,
        timeout=900,
    )
    if result.returncode != 0:
        output = (result.stdout + result.stderr).strip().splitlines()
        block("repository checks failed\n" + "\n".join(output[-40:]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
