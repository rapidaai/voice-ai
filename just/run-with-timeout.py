#!/usr/bin/env python3

import subprocess
import sys


def main() -> int:
    if len(sys.argv) < 3:
        print("usage: run-with-timeout.py SECONDS COMMAND [ARG ...]", file=sys.stderr)
        return 2

    try:
        timeout_seconds = float(sys.argv[1])
    except ValueError:
        print(f"invalid timeout: {sys.argv[1]}", file=sys.stderr)
        return 2

    try:
        completed = subprocess.run(sys.argv[2:], timeout=timeout_seconds, check=False)
    except subprocess.TimeoutExpired:
        print(f"command timed out after {timeout_seconds:g}s: {' '.join(sys.argv[2:])}", file=sys.stderr)
        return 124
    except OSError as error:
        print(f"unable to run {sys.argv[2]}: {error}", file=sys.stderr)
        return 127

    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
