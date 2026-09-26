#!/usr/bin/env python3
"""Reject destructive or release-oriented shell commands before Claude executes them."""

from __future__ import annotations

import json
import os
import re
import shlex
import subprocess
import sys


PROTECTED_BRANCHES = {"main", "master"}
WRAPPERS = {"builtin", "caffeinate", "command", "env", "exec", "nice", "nohup", "time"}
SHELLS = {"bash", "dash", "sh", "zsh"}
DELETE_PROGRAMS = {"rm", "rmdir", "shred", "srm", "truncate", "unlink"}
EGRESS_PROGRAMS = {"ftp", "nc", "ncat", "netcat", "scp", "sftp", "socat"}


def reject(reason: str, command: str) -> None:
    preview = command.strip().replace("\n", " ")[:180]
    print(f"Blocked by repository command guard: {reason}: {preview}", file=sys.stderr)
    print("Ask the user to run the command directly when this operation is intentional.", file=sys.stderr)
    raise SystemExit(2)


def tokens(command: str) -> list[str]:
    lexer = shlex.shlex(command.replace("\n", ";"), posix=True, punctuation_chars=";&|<>()")
    lexer.whitespace_split = True
    return list(lexer)


def command_segments(command: str) -> list[tuple[list[str], list[str] | None]]:
    parsed = tokens(command)
    result: list[tuple[list[str], list[str] | None]] = []
    segment: list[str] = []
    pipe_source: list[str] | None = None
    for item in parsed + [";"]:
        if item and set(item) <= set(";&|<>()"):
            if segment:
                result.append((segment, pipe_source))
            pipe_source = segment if item == "|" else None
            segment = []
            continue
        segment.append(item)
    return result


def strip_prefix(words: list[str]) -> list[str]:
    index = 0
    while index < len(words):
        word = words[index]
        if re.match(r"^[A-Za-z_][A-Za-z0-9_]*=", word):
            index += 1
            continue
        if os.path.basename(word) in WRAPPERS:
            index += 1
            while index < len(words) and words[index].startswith("-"):
                index += 1
            continue
        break
    return words[index:]


def current_branch(cwd: str) -> str:
    try:
        result = subprocess.run(
            ["git", "symbolic-ref", "--short", "-q", "HEAD"],
            cwd=cwd or None,
            capture_output=True,
            check=False,
            text=True,
            timeout=5,
        )
    except (OSError, subprocess.TimeoutExpired):
        return ""
    return result.stdout.strip()


def ref_target(spec: str) -> str:
    target = spec.lstrip("+").split(":", 1)[-1]
    return re.sub(r"^refs/heads/", "", target)


def is_existing_tag(cwd: str, spec: str) -> bool:
    if ":" in spec or spec.startswith("-"):
        return False
    try:
        result = subprocess.run(
            ["git", "show-ref", "--verify", "--quiet", f"refs/tags/{spec}"],
            cwd=cwd or None,
            check=False,
            timeout=5,
        )
    except (OSError, subprocess.TimeoutExpired):
        return False
    return result.returncode == 0


def check_git(arguments: list[str], cwd: str, command: str) -> None:
    index = 0
    while index < len(arguments) and arguments[index].startswith("-"):
        option = arguments[index]
        if option == "-C" and index + 1 < len(arguments):
            selected = arguments[index + 1]
            cwd = selected if os.path.isabs(selected) else os.path.join(cwd, selected)
        if option == "-c" and index + 1 < len(arguments):
            if arguments[index + 1].lower().startswith("core.hookspath="):
                reject("Git hook configuration may not be bypassed", command)
        index += 2 if option in {"-C", "-c", "--git-dir", "--namespace", "--work-tree"} else 1
    if index >= len(arguments):
        return

    action = arguments[index]
    rest = arguments[index + 1 :]
    flags = [item for item in rest if item.startswith("-")]
    positional = [item for item in rest if not item.startswith("-")]

    if "--no-verify" in rest or (action == "commit" and "-n" in flags):
        reject("repository hooks may not be bypassed", command)

    if action == "push":
        if any(
            item in {"--force", "-f", "--force-if-includes", "--force-with-lease"}
            or item.startswith("--force-with-lease=")
            for item in flags
        ):
            reject("force pushes are not agent actions", command)
        if any(
            item in {"--all", "--delete", "-d", "--follow-tags", "--mirror", "--prune", "--tags"}
            for item in flags
        ):
            reject("bulk, tag, or deletion pushes are not agent actions", command)
        refspecs = positional[1:]
        if "tag" in refspecs or any(is_existing_tag(cwd, spec) for spec in refspecs):
            reject("tag pushes are not agent actions", command)
        for spec in refspecs:
            if spec.startswith(("+", ":")):
                reject("force or deletion refspec", command)
            if spec.startswith("refs/tags/") or ref_target(spec) in PROTECTED_BRANCHES:
                reject("protected branch or tag push", command)
            if ref_target(spec) == "HEAD" and current_branch(cwd) in PROTECTED_BRANCHES:
                reject("implicit push from a protected branch", command)
        if not refspecs and current_branch(cwd) in PROTECTED_BRANCHES:
            reject("implicit push from a protected branch", command)
        return

    if action == "reset" and "--hard" in flags:
        reject("git reset --hard discards work", command)
    if action == "clean" and any(item == "--force" or re.match(r"^-[A-Za-z]*f", item) for item in flags):
        reject("git clean deletes untracked work", command)
    if action == "checkout" and "--" in rest:
        reject("git checkout with a path discards work", command)
    if action == "restore" and "--staged" not in flags and "-S" not in flags:
        reject("git restore may discard work", command)
    if action == "branch" and any(item in {"-d", "-D", "--delete"} for item in flags):
        reject("branch deletion is not an agent action", command)
    if action == "stash":
        stash_action = positional[0] if positional else ""
        if stash_action in {"", "save", "pop", "drop", "clear"}:
            reject("shared stash mutation is unsafe across worktrees", command)
    if action == "tag" and positional:
        reject("tag mutation is not an agent action", command)
    if action == "config" and "--global" in flags:
        reject("global Git configuration is outside repository scope", command)


def check_segment(words: list[str], pipe_source: list[str] | None, cwd: str, command: str, depth: int) -> None:
    if any(
        word.startswith(("SKIP=", "HUSKY=", "PRE_COMMIT_ALLOW_NO_CONFIG="))
        for word in words
    ):
        reject("required checks may not be bypassed", command)
    words = strip_prefix(words)
    if not words:
        return

    program = os.path.basename(words[0])
    arguments = words[1:]

    if program in {"sudo", "doas"}:
        reject("privilege escalation", command)
    if program in SHELLS and "-c" in arguments:
        index = arguments.index("-c")
        inner = arguments[index + 1] if index + 1 < len(arguments) else ""
        if depth >= 3:
            reject("nested shell depth cannot be inspected safely", command)
        scan(inner, cwd, command, depth + 1)
        return
    if program in SHELLS and pipe_source is not None:
        reject("piping generated or remote input into a shell", command)
    if program == "eval":
        if depth >= 3:
            reject("nested eval depth cannot be inspected safely", command)
        scan(" ".join(arguments), cwd, command, depth + 1)
        return
    if program == "xargs":
        if any(os.path.basename(item) in DELETE_PROGRAMS | SHELLS for item in arguments):
            reject("xargs may not invoke deletion or a shell", command)
        nested = [item for item in arguments if not item.startswith("-")]
        if nested:
            check_segment(nested, None, cwd, command, depth + 1)
        return

    if program in DELETE_PROGRAMS:
        reject(f"file deletion via {program}", command)
    if program in EGRESS_PROGRAMS:
        reject(f"unfiltered network transfer via {program}", command)
    if program == "curl" and any(
        item in {"-d", "--data", "--data-ascii", "--data-binary", "--data-raw", "--form", "-F", "--json", "-T", "--upload-file"}
        or item.startswith(("--data=", "--data-ascii=", "--data-binary=", "--data-raw=", "--form=", "--json=", "--upload-file="))
        or re.match(r"^-(?:d|F|T).+", item)
        for item in arguments
    ):
        reject("unfiltered outbound data via curl", command)
    if program == "wget" and any(
        item.startswith(("--post-data", "--post-file", "--body-data", "--body-file", "--method=POST", "--method=PUT", "--method=PATCH"))
        for item in arguments
    ):
        reject("unfiltered outbound data via wget", command)
    if program == "rsync" and any(":" in item and not item.startswith(("./", "../")) for item in arguments):
        reject("unfiltered remote transfer via rsync", command)
    if program == "find" and (
        "-delete" in arguments
        or (
            any(item in {"-exec", "-execdir", "-ok"} for item in arguments)
            and any(os.path.basename(item) in DELETE_PROGRAMS | SHELLS for item in arguments)
        )
    ):
        reject("file deletion through find", command)
    if program in {"chmod", "chown", "chgrp"} and (program != "chmod" or "-R" in arguments):
        reject("recursive permission or ownership change", command)
    if program in {"dd", "diskutil", "mkfs"}:
        reject("disk-level mutation", command)
    if program == "docker" and arguments[:2] == ["system", "prune"]:
        reject("global Docker cleanup", command)
    if program in {"terraform", "tofu"} and arguments[:1] in (["apply"], ["destroy"]):
        reject("infrastructure mutation", command)
    if program == "kubectl" and arguments[:1] in (["apply"], ["delete"]):
        reject("cluster mutation", command)
    if program == "helm" and arguments[:1] in (["install"], ["upgrade"], ["uninstall"]):
        reject("cluster release mutation", command)
    if program == "gh" and len(arguments) >= 2:
        if arguments[:2] == ["pr", "merge"] or arguments[0] == "release" or arguments[:2] == ["repo", "delete"]:
            reject("merge, release, or repository deletion", command)
        if arguments[:2] in (["gist", "create"], ["issue", "create"], ["issue", "comment"], ["pr", "comment"], ["pr", "review"]):
            reject("unfiltered outbound content via GitHub CLI", command)
        if arguments[0] == "api" and any(item in {"--method", "-X"} or item.startswith(("--method=", "-X")) for item in arguments):
            reject("unfiltered mutating GitHub API call", command)
    if program == "git":
        check_git(arguments, cwd, command)


def scan(command: str, cwd: str, original: str, depth: int = 0) -> None:
    try:
        segments = command_segments(command)
    except ValueError as error:
        reject(f"command could not be inspected safely ({error})", original)
    for words, pipe_source in segments:
        check_segment(words, pipe_source, cwd, original, depth)


def main() -> None:
    try:
        payload = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        return
    tool_input = payload.get("tool_input") or {}
    command = tool_input.get("command") or ""
    if not isinstance(command, str) or not command.strip():
        return
    cwd = tool_input.get("cwd") or os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd()
    scan(command, cwd, command)


if __name__ == "__main__":
    main()
