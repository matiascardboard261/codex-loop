#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys

from loop_common import iter_loop_records, normalize_status


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Inspect global Codex loop state.")
    parser.add_argument("--session-id", default=None, help="Only print loop records for one session.")
    parser.add_argument(
        "--workspace-root",
        default=None,
        help="Only print loop records for one workspace root.",
    )
    parser.add_argument("--all", action="store_true", help="Include non-active loop records.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    workspace_root = Path(args.workspace_root).resolve() if args.workspace_root else None
    records = []
    for path, data in iter_loop_records():
        if args.session_id and data.get("session_id") != args.session_id:
            continue
        if workspace_root is not None:
            candidate = Path(str(data.get("workspace_root") or data.get("cwd") or ".")).resolve()
            if candidate != workspace_root:
                continue
        if not args.all and normalize_status(data.get("status")) != "active":
            continue
        enriched = dict(data)
        enriched["path"] = str(path)
        records.append(enriched)

    json.dump(records, sys.stdout, indent=2, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
