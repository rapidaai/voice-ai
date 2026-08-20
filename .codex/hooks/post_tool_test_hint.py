#!/usr/bin/env python3
import json
import sys


def main() -> int:
    _ = sys.stdin.read()
    msg = {
        "hook": "PostToolUse",
        "advice": "Stay within the approved ownership scope, update corresponding tests, run required commands and strict skill validation, then route the verified diff to an independent code reviewer.",
    }
    print(json.dumps(msg))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
