#!/usr/bin/env python3

from __future__ import annotations

import json
import sys
from typing import Any

from loop_common import (
    RAPID_STOP_LIMIT,
    RAPID_STOP_THRESHOLD_SECONDS,
    continuation_reason,
    int_value,
    isoformat_z,
    parse_iso8601,
    replace_loop_file,
    resolve_active_loop,
    resolve_limit_mode,
    stop_warning,
    utc_now,
)


def handle_stop(payload: dict[str, Any], now=None) -> dict[str, Any] | None:
    session_id = payload.get("session_id")
    if not isinstance(session_id, str) or not session_id.strip():
        return None

    try:
        loop_path, loop_data = resolve_active_loop(session_id)
        if loop_path is None or loop_data is None:
            return None

        now = now or utc_now()
        previous_stop = parse_iso8601(loop_data.get("last_stop_at"))
        loop_data["last_assistant_message"] = payload.get("last_assistant_message")
        loop_data["last_stop_at"] = isoformat_z(now)

        limit_mode = resolve_limit_mode(loop_data)
        if limit_mode == "time":
            deadline_at = parse_iso8601(loop_data.get("deadline_at"))
            if deadline_at is None:
                raise ValueError("active codex-loop is missing deadline_at")
            if now >= deadline_at:
                loop_data["status"] = "completed"
                replace_loop_file(loop_path, loop_data)
                return None
            remaining_seconds = max(0, int((deadline_at - now).total_seconds()))
        else:
            remaining_seconds = None
            completed_rounds = int_value(loop_data.get("completed_rounds"), 0) + 1
            target_rounds = int_value(loop_data.get("target_rounds"), 0)
            if target_rounds <= 0:
                raise ValueError("active codex-loop is missing a positive target_rounds")
            loop_data["completed_rounds"] = completed_rounds
            if completed_rounds >= target_rounds:
                loop_data["status"] = "completed"
                replace_loop_file(loop_path, loop_data)
                return None

        rapid_count = 0
        if previous_stop is not None:
            delta = (now - previous_stop).total_seconds()
            if delta <= RAPID_STOP_THRESHOLD_SECONDS:
                rapid_count = int_value(loop_data.get("rapid_stop_count"), 0) + 1
        loop_data["rapid_stop_count"] = rapid_count

        if rapid_count >= RAPID_STOP_LIMIT and bool(loop_data.get("escalation_used")):
            loop_data["status"] = "cut_short"
            replace_loop_file(loop_path, loop_data)
            return stop_warning(
                "Codex loop stopped after repeated rapid completions. Review the latest result manually before reactivating the loop."
            )

        aggressive = rapid_count >= RAPID_STOP_LIMIT and not bool(loop_data.get("escalation_used"))
        if aggressive:
            loop_data["escalation_used"] = True

        loop_data["continue_count"] = int_value(loop_data.get("continue_count"), 0) + 1
        loop_data["last_continue_at"] = isoformat_z(now)
        replace_loop_file(loop_path, loop_data)

        return {
            "decision": "block",
            "reason": continuation_reason(
                loop_data,
                remaining_seconds=remaining_seconds,
                aggressive=aggressive,
            ),
        }
    except Exception as exc:
        return stop_warning(f"Codex loop stop hook failed: {exc}")


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        print(json.dumps(stop_warning(f"Codex loop stop hook received invalid JSON: {exc}")))
        return 0

    result = handle_stop(payload)
    if result is not None:
        json.dump(result, sys.stdout)
        sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
