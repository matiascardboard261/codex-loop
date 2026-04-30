#!/usr/bin/env python3

from __future__ import annotations

import json
import os
from pathlib import Path
import sys
from typing import Any

from loop_common import (
    block_with_reason,
    build_loop_record,
    create_loop_path,
    extract_activation,
    find_workspace_root,
    looks_like_activation,
    replace_loop_file,
    supersede_active_loops,
    utc_now,
)


def handle_user_prompt_submit(payload: dict[str, Any], now=None) -> dict[str, Any] | None:
    prompt = payload.get("prompt")
    if not isinstance(prompt, str) or not prompt:
        return None
    if not looks_like_activation(prompt):
        return None

    try:
        activation = extract_activation(prompt)
        if activation is None:
            return None

        session_id = payload.get("session_id")
        if not isinstance(session_id, str) or not session_id.strip():
            raise ValueError("codex-loop activation requires a session_id from Codex")

        working_dir = payload.get("cwd") or os.getcwd()
        workspace_root = find_workspace_root(working_dir)
        resolved_cwd = str(Path(working_dir).resolve())
        now = now or utc_now()

        supersede_active_loops(session_id)
        loop_path = create_loop_path(activation.slug, now)
        loop_record = build_loop_record(
            session_id=session_id,
            cwd=resolved_cwd,
            workspace_root=workspace_root,
            activation=activation,
            now=now,
        )
        replace_loop_file(loop_path, loop_record)
    except Exception as exc:
        return block_with_reason(f"Codex loop activation failed: {exc}")

    return None


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(f"invalid JSON input: {exc}", file=sys.stderr)
        return 1

    result = handle_user_prompt_submit(payload)
    if result is not None:
        json.dump(result, sys.stdout)
        sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
