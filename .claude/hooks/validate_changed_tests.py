#!/usr/bin/env python3
import json
import os
import re
import subprocess
import sys
from pathlib import Path

def _run(cmd: list[str]) -> str:
    out = subprocess.run(cmd, check=False, capture_output=True, text=True)
    return out.stdout.strip()


def _repo_root() -> str:
    root = _run(["git", "rev-parse", "--show-toplevel"]).strip()
    return root or str(Path.cwd())


def _looks_like_repo_file(path: str) -> bool:
    return bool(re.match(r"^(ui|api|pkg|cmd|\.claude|\.codex)/", path)) or path == "AGENTS.md"


def _normalize_path(value: str, repo_root: str) -> str:
    s = value.strip().strip("\"'")
    if not s:
        return ""
    if s.startswith("./"):
        s = s[2:]
    if s.startswith(repo_root):
        s = str(Path(s).relative_to(repo_root))
    if _looks_like_repo_file(s):
        return s
    return ""


def _extract_paths_from_obj(obj, repo_root: str, out: set[str]) -> None:
    if isinstance(obj, dict):
        for _, v in obj.items():
            _extract_paths_from_obj(v, repo_root, out)
        return
    if isinstance(obj, list):
        for v in obj:
            _extract_paths_from_obj(v, repo_root, out)
        return
    if isinstance(obj, str):
        p = _normalize_path(obj, repo_root)
        if p:
            out.add(p)


def _paths_from_stdin_payload(raw: str, repo_root: str) -> list[str]:
    if not raw.strip():
        return []
    try:
        payload = json.loads(raw)
    except Exception:
        return []
    found: set[str] = set()
    _extract_paths_from_obj(payload, repo_root, found)
    return sorted(found)


def _paths_from_env(repo_root: str) -> list[str]:
    raw = os.getenv("HOOK_CHANGED_FILES", "").strip()
    if not raw:
        return []
    items = re.split(r"[\n,]", raw)
    out = set()
    for i in items:
        p = _normalize_path(i, repo_root)
        if p:
            out.add(p)
    return sorted(out)


def _changed_files(stdin_raw: str) -> list[str]:
    """Resolve only explicitly scoped files.

    Never fall back to repository-wide git diff because the worktree may contain
    unrelated user or agent changes.
    """
    repo_root = _repo_root()

    env_files = _paths_from_env(repo_root)
    if env_files:
        return env_files

    payload_files = _paths_from_stdin_payload(stdin_raw, repo_root)
    if payload_files:
        return payload_files
    return []


def _is_ui_code(path: str) -> bool:
    if not path.startswith("ui/src/"):
        return False
    p = Path(path)
    if p.suffix not in {".ts", ".tsx", ".js", ".jsx"}:
        return False
    low = path.lower()
    return ".test." not in low and ".spec." not in low and "__tests__" not in low


def _is_ui_test(path: str) -> bool:
    low = path.lower()
    return path.startswith("ui/src/") and (".test." in low or ".spec." in low or "__tests__" in low)


def _is_backend_code(path: str) -> bool:
    if not (path.startswith("api/") or path.startswith("pkg/") or path.startswith("cmd/")):
        return False
    return path.endswith(".go") and not path.endswith("_test.go")


def _is_backend_test(path: str) -> bool:
    return (path.startswith("api/") or path.startswith("pkg/") or path.startswith("cmd/")) and path.endswith("_test.go")


def main() -> int:
    raw = "" if sys.stdin.isatty() else sys.stdin.read()
    changed = _changed_files(raw)

    ui_code_changed = [f for f in changed if _is_ui_code(f)]
    ui_tests_changed = [f for f in changed if _is_ui_test(f)]
    backend_code_changed = [f for f in changed if _is_backend_code(f)]
    backend_tests_changed = [f for f in changed if _is_backend_test(f)]

    errors = []
    if ui_code_changed and not ui_tests_changed:
        errors.append(
            "UI code changed but no UI unit test changed. Add/update tests under ui/src with .test/.spec naming."
        )
    backend_test_dirs = {str(Path(path).parent) for path in backend_tests_changed}
    missing_backend_test_dirs = sorted(
        {str(Path(path).parent) for path in backend_code_changed} - backend_test_dirs
    )
    if missing_backend_test_dirs:
        errors.append(
            "Backend Go code changed without a *_test.go change in: "
            + ", ".join(missing_backend_test_dirs)
        )

    result = {
        "hook": "validate_changed_tests",
        "ui_code_changed": ui_code_changed,
        "ui_tests_changed": ui_tests_changed,
        "backend_code_changed": backend_code_changed,
        "backend_tests_changed": backend_tests_changed,
        "missing_backend_test_dirs": missing_backend_test_dirs,
        "errors": errors,
    }
    print(json.dumps(result, indent=2))

    if errors:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
