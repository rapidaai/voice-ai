#!/usr/bin/env python3
import json
import os
import re
import subprocess
import sys
from pathlib import Path

from hook_state import read_tracked_files


def _run(cmd: list[str], cwd: str | None = None) -> tuple[int, str]:
    out = subprocess.run(cmd, cwd=cwd, check=False, capture_output=True, text=True)
    text = (out.stdout or "") + (out.stderr or "")
    return out.returncode, text


def _repo_root() -> str:
    rc, out = _run(["git", "rev-parse", "--show-toplevel"])
    _ = rc
    return out.strip() or str(Path.cwd())


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
    """Resolve explicitly scoped files, then session-scoped edit records.

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
    return read_tracked_files(stdin_raw, repo_root)


def _backend_dirs(changed: list[str]) -> list[str]:
    dirs = set()
    for f in changed:
        if (f.startswith("api/") or f.startswith("pkg/") or f.startswith("cmd/")) and f.endswith(".go"):
            dirs.add(str(Path(f).parent))
    return sorted(dirs)


def main() -> int:
    raw = sys.stdin.read()
    changed = _changed_files(raw)
    results = []

    ui_changed = any(f.startswith("ui/src/") for f in changed)
    backend_dirs = _backend_dirs(changed)

    if ui_changed:
        rc, output = _run(["yarn", "test", "providers"], cwd="ui")
        results.append({"cmd": "cd ui && yarn test providers", "exit_code": rc, "output_tail": output[-2000:]})

    for d in backend_dirs:
        rc, output = _run(["go", "test", f"./{d}"])
        results.append({"cmd": f"go test ./{d}", "exit_code": rc, "output_tail": output[-2000:]})

    failed = [r for r in results if r["exit_code"] != 0]
    print(json.dumps({"hook": "run_required_tests", "results": results, "failed_count": len(failed)}, indent=2))
    if failed:
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
