from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
from pathlib import Path
from types import ModuleType
from importlib.machinery import SourceFileLoader


ROOT = Path(__file__).resolve().parents[2]


def load_script(name: str, relative_path: str) -> ModuleType:
    loader = SourceFileLoader(name, str(ROOT / relative_path))
    spec = importlib.util.spec_from_loader(name, loader)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {relative_path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


def run_hook(relative_path: str, command: str, **extra: object) -> subprocess.CompletedProcess[str]:
    payload = {"tool_input": {"command": command, "cwd": str(ROOT)}, **extra}
    environment = os.environ.copy()
    environment["CLAUDE_PROJECT_DIR"] = str(ROOT)
    return subprocess.run(
        [sys.executable, str(ROOT / relative_path)],
        input=json.dumps(payload),
        capture_output=True,
        check=False,
        text=True,
        env=environment,
    )
