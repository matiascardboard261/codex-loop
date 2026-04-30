#!/usr/bin/env python3

from __future__ import annotations

import ast
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import re
import tempfile
from typing import Any

try:
    import tomllib
except ModuleNotFoundError:  # pragma: no cover - Python 3.11+ should provide tomllib
    tomllib = None


VERSION = 2
RAPID_STOP_THRESHOLD_SECONDS = 120
RAPID_STOP_LIMIT = 3
ACTIVATION_PREFIX = "[[CODEX_LOOP"
ACTIVATION_RE = re.compile(r'^\[\[CODEX_LOOP(?P<body>[^\]]+)\]\]\s*$')
ATTR_RE = re.compile(r'([A-Za-z_][A-Za-z0-9_-]*)="([^"]*)"')
DURATION_TOKEN_RE = re.compile(
    r"(?i)(\d+)\s*(seconds?|secs?|sec|s|minutes?|mins?|min|m|hours?|hrs?|hr|h|days?|day|d)\b"
)
ROUND_RE = re.compile(r"^[1-9][0-9]*$")
SUPPORTED_STATUSES = {"active", "completed", "superseded", "cut_short"}
PLUGIN_RUNTIME_NAME = "codex-timed-loop"
DEFAULT_RUNTIME_CONFIG = {
    "optional_skill_name": "",
    "optional_skill_path": "",
    "extra_continuation_guidance": "",
}


@dataclass(frozen=True)
class Activation:
    name: str
    slug: str
    limit_mode: str
    task_prompt: str
    activation_prompt: str
    duration_text: str | None = None
    min_duration_seconds: int | None = None
    rounds_text: str | None = None
    target_rounds: int | None = None


@dataclass(frozen=True)
class OptionalContinuationConfig:
    skill_name: str = ""
    skill_path: str = ""
    extra_guidance: str = ""


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def isoformat_z(value: datetime) -> str:
    return value.astimezone(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def parse_iso8601(value: str | None) -> datetime | None:
    if not value:
        return None
    normalized = value.replace("Z", "+00:00")
    return datetime.fromisoformat(normalized)


def slugify(value: str) -> str:
    lowered = value.strip().lower()
    lowered = re.sub(r"[^a-z0-9]+", "-", lowered)
    lowered = re.sub(r"-{2,}", "-", lowered).strip("-")
    return lowered or "loop"


def codex_home() -> Path:
    configured = os.environ.get("CODEX_HOME", "").strip()
    if configured:
        return Path(configured).expanduser().resolve()
    return (Path.home() / ".codex").resolve()


def runtime_root(create: bool = False) -> Path:
    path = codex_home() / PLUGIN_RUNTIME_NAME
    if create:
        path.mkdir(parents=True, exist_ok=True)
    return path


def runtime_loops_dir(create: bool = False) -> Path:
    path = runtime_root(create=create) / "loops"
    if create:
        path.mkdir(parents=True, exist_ok=True)
    return path


def runtime_config_path() -> Path:
    return runtime_root(create=False) / "config.toml"


def _normalize_duration_unit(unit: str) -> int:
    lowered = unit.lower()
    if lowered.startswith("s"):
        return 1
    if lowered.startswith("m"):
        return 60
    if lowered.startswith("h"):
        return 3600
    if lowered.startswith("d"):
        return 86400
    raise ValueError(f"unsupported duration unit {unit!r}")


def parse_duration(value: str) -> int:
    normalized = value.strip()
    if not normalized:
        raise ValueError(f"invalid duration {value!r}")

    offset = 0
    total = 0
    matched = False
    for match in DURATION_TOKEN_RE.finditer(normalized):
        if normalized[offset:match.start()].strip():
            raise ValueError(f"invalid duration {value!r}")
        total += int(match.group(1)) * _normalize_duration_unit(match.group(2))
        offset = match.end()
        matched = True

    if normalized[offset:].strip() or not matched or total <= 0:
        raise ValueError(f"invalid duration {value!r}")
    return total


def parse_rounds(value: str) -> int:
    normalized = value.strip()
    if not ROUND_RE.fullmatch(normalized):
        raise ValueError(f"invalid rounds {value!r}")
    rounds = int(normalized)
    if rounds <= 0:
        raise ValueError(f"invalid rounds {value!r}")
    return rounds


def extract_activation(prompt: str) -> Activation | None:
    lines = prompt.splitlines()
    if not lines:
        return None

    first_line = lines[0].strip()
    if not first_line.startswith(ACTIVATION_PREFIX):
        return None

    match = ACTIVATION_RE.match(first_line)
    if not match:
        raise ValueError("invalid CODEX_LOOP header syntax")

    attributes = {key: value for key, value in ATTR_RE.findall(match.group("body"))}
    name = attributes.get("name", "").strip()
    duration_text = attributes.get("min", "").strip()
    rounds_text = attributes.get("rounds", "").strip()
    if not name:
        raise ValueError('CODEX_LOOP header requires name="..."')
    if bool(duration_text) == bool(rounds_text):
        raise ValueError('CODEX_LOOP header requires exactly one of min="..." or rounds="..."')

    task_prompt = "\n".join(lines[1:]).lstrip("\n")
    if duration_text:
        return Activation(
            name=name,
            slug=slugify(name),
            limit_mode="time",
            task_prompt=task_prompt,
            activation_prompt=prompt,
            duration_text=duration_text,
            min_duration_seconds=parse_duration(duration_text),
        )

    return Activation(
        name=name,
        slug=slugify(name),
        limit_mode="rounds",
        task_prompt=task_prompt,
        activation_prompt=prompt,
        rounds_text=rounds_text,
        target_rounds=parse_rounds(rounds_text),
    )


def looks_like_activation(prompt: str) -> bool:
    first_line = prompt.splitlines()[0].strip() if prompt.splitlines() else ""
    return first_line.startswith(ACTIVATION_PREFIX)


def find_workspace_root(start: str | Path) -> Path:
    current = Path(start).resolve()
    for candidate in [current, *current.parents]:
        if (candidate / ".codex").is_dir():
            return candidate
        if (candidate / ".git").exists():
            return candidate
    return current


def atomic_write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", delete=False, dir=path.parent, encoding="utf-8") as handle:
        json.dump(payload, handle, indent=2, sort_keys=True)
        handle.write("\n")
        temp_path = Path(handle.name)
    temp_path.replace(path)


def load_loop(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"loop file {path} must contain a JSON object")
    return data


def iter_loop_records() -> list[tuple[Path, dict[str, Any]]]:
    records: list[tuple[Path, dict[str, Any]]] = []
    loops_root = runtime_loops_dir(create=False)
    if not loops_root.is_dir():
        return records
    for path in sorted(loops_root.glob("*.json")):
        try:
            data = load_loop(path)
        except Exception:
            continue
        records.append((path, data))
    return records


def normalize_status(value: Any) -> str:
    if isinstance(value, str) and value in SUPPORTED_STATUSES:
        return value
    return "active"


def replace_loop_file(path: Path, loop_data: dict[str, Any]) -> None:
    loop_data["status"] = normalize_status(loop_data.get("status"))
    atomic_write_json(path, loop_data)


def supersede_active_loops(session_id: str, keep_path: Path | None = None) -> None:
    for path, data in iter_loop_records():
        if path == keep_path:
            continue
        if data.get("session_id") != session_id:
            continue
        if normalize_status(data.get("status")) != "active":
            continue
        data["status"] = "superseded"
        replace_loop_file(path, data)


def resolve_active_loop(session_id: str) -> tuple[Path | None, dict[str, Any] | None]:
    active: list[tuple[Path, dict[str, Any]]] = []
    for path, data in iter_loop_records():
        if data.get("session_id") != session_id:
            continue
        if normalize_status(data.get("status")) != "active":
            continue
        active.append((path, data))

    if not active:
        return None, None

    active.sort(key=lambda item: (item[1].get("started_at", ""), item[0].name))
    keep_path, keep_data = active[-1]
    for path, data in active[:-1]:
        data["status"] = "superseded"
        replace_loop_file(path, data)
    return keep_path, keep_data


def create_loop_path(slug: str, now: datetime) -> Path:
    stamp = now.strftime("%Y%m%dT%H%M%SZ")
    return runtime_loops_dir(create=True) / f"{stamp}_{slug}.json"


def build_loop_record(
    session_id: str,
    cwd: str,
    workspace_root: Path,
    activation: Activation,
    now: datetime,
) -> dict[str, Any]:
    record: dict[str, Any] = {
        "version": VERSION,
        "session_id": session_id,
        "name": activation.name,
        "slug": activation.slug,
        "cwd": cwd,
        "workspace_root": str(workspace_root),
        "started_at": isoformat_z(now),
        "task_prompt": activation.task_prompt,
        "activation_prompt": activation.activation_prompt,
        "status": "active",
        "limit_mode": activation.limit_mode,
        "continue_count": 0,
        "rapid_stop_count": 0,
        "escalation_used": False,
        "last_stop_at": None,
        "last_continue_at": None,
        "last_assistant_message": None,
        "duration_text": None,
        "min_duration_seconds": None,
        "deadline_at": None,
        "rounds_text": None,
        "target_rounds": None,
        "completed_rounds": 0,
    }
    if activation.limit_mode == "time":
        record["duration_text"] = activation.duration_text
        record["min_duration_seconds"] = activation.min_duration_seconds
        record["deadline_at"] = isoformat_z(now + timedelta(seconds=activation.min_duration_seconds or 0))
    else:
        record["rounds_text"] = activation.rounds_text
        record["target_rounds"] = activation.target_rounds
    return record


def format_seconds(seconds: int) -> str:
    if seconds <= 0:
        return "0s"
    parts: list[str] = []
    remainder = seconds
    for suffix, unit in (("d", 86400), ("h", 3600), ("m", 60), ("s", 1)):
        amount, remainder = divmod(remainder, unit)
        if amount:
            parts.append(f"{amount}{suffix}")
    return " ".join(parts)


def int_value(value: Any, default: int = 0) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def resolve_limit_mode(loop_data: dict[str, Any]) -> str:
    mode = loop_data.get("limit_mode")
    if mode in {"time", "rounds"}:
        return mode
    if loop_data.get("target_rounds") is not None:
        return "rounds"
    return "time"


def _load_runtime_config_with_tomllib(text: str) -> dict[str, Any]:
    if tomllib is None:
        raise ModuleNotFoundError("tomllib is unavailable")
    return tomllib.loads(text)


def _load_runtime_config_fallback(text: str) -> dict[str, Any]:
    data: dict[str, Any] = {}
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        key, raw_value = line.split("=", 1)
        key = key.strip()
        try:
            value = ast.literal_eval(raw_value.strip())
        except Exception:
            continue
        if isinstance(value, str):
            data[key] = value
    return data


def load_runtime_config() -> dict[str, str]:
    config = dict(DEFAULT_RUNTIME_CONFIG)
    path = runtime_config_path()
    if not path.is_file():
        return config

    text = path.read_text(encoding="utf-8")
    try:
        raw = _load_runtime_config_with_tomllib(text)
    except Exception:
        raw = _load_runtime_config_fallback(text)

    if not isinstance(raw, dict):
        return config
    for key in DEFAULT_RUNTIME_CONFIG:
        value = raw.get(key)
        if isinstance(value, str):
            config[key] = value
    return config


def resolve_optional_continuation_config(workspace_root: Path) -> OptionalContinuationConfig:
    config = load_runtime_config()
    skill_name = config.get("optional_skill_name", "").strip()
    skill_path_text = config.get("optional_skill_path", "").strip()
    extra_guidance = config.get("extra_continuation_guidance", "").strip()

    resolved_skill_path = ""
    if skill_name and skill_path_text:
        candidate = Path(skill_path_text).expanduser()
        if not candidate.is_absolute():
            candidate = workspace_root / candidate
        candidate = candidate.resolve()
        if candidate.is_dir():
            candidate = candidate / "SKILL.md"
        if candidate.is_file():
            resolved_skill_path = str(candidate)
        else:
            skill_name = ""

    return OptionalContinuationConfig(
        skill_name=skill_name,
        skill_path=resolved_skill_path,
        extra_guidance=extra_guidance,
    )


def continuation_reason(
    loop_data: dict[str, Any],
    remaining_seconds: int | None = None,
    aggressive: bool = False,
) -> str:
    workspace_root = Path(str(loop_data.get("workspace_root") or loop_data.get("cwd") or ".")).resolve()
    continuation_config = resolve_optional_continuation_config(workspace_root)
    original_task = (loop_data.get("task_prompt") or loop_data.get("activation_prompt") or "").strip()
    latest_message = (loop_data.get("last_assistant_message") or "").strip()
    limit_mode = resolve_limit_mode(loop_data)

    lines = ["Continue the active codex-loop task."]
    if limit_mode == "time":
        lines.extend(
            [
                f"The minimum work duration has not elapsed yet. Remaining time: {format_seconds(remaining_seconds or 0)}.",
                "Do not stop just because the primary request appears complete.",
            ]
        )
    else:
        completed_rounds = int_value(loop_data.get("completed_rounds"), 0)
        target_rounds = int_value(loop_data.get("target_rounds"), 0)
        next_round = min(target_rounds, completed_rounds + 1) if target_rounds > 0 else completed_rounds + 1
        lines.extend(
            [
                f"Round {next_round} of {target_rounds} begins now.",
                f"You have completed {completed_rounds} of {target_rounds} required rounds.",
                "Treat this as a deliberate fresh pass. Do not just restate the previous conclusion.",
            ]
        )

    if aggressive:
        lines.append("Several turns have ended too quickly. Broaden the scope materially before stopping again.")

    lines.extend(
        [
            "Expand the work with:",
            "- hardening and cleanup of the current solution",
            "- edge cases and larger scenarios",
            "- adjacent project areas that may share the same weakness",
            "- stronger validation with real commands, tests, or QA evidence",
            "- additional regression coverage where the same failure mode could recur",
        ]
    )
    if limit_mode == "rounds":
        lines.append("- a fresh challenge to any earlier conclusion before you stop again")
    if continuation_config.skill_name and continuation_config.skill_path:
        lines.append(
            f"- explicit use of the {continuation_config.skill_name} skill at {continuation_config.skill_path}"
        )
    if continuation_config.extra_guidance:
        lines.extend(["", "Additional configured guidance:", continuation_config.extra_guidance])

    if original_task:
        lines.extend(["", "Original task:", original_task])
    if latest_message:
        lines.extend(["", "Latest assistant message before this continuation:", latest_message])

    return "\n".join(lines).strip()


def stop_warning(message: str) -> dict[str, Any]:
    return {
        "continue": False,
        "stopReason": "codex-loop-cut-short",
        "systemMessage": message,
    }


def block_with_reason(reason: str) -> dict[str, Any]:
    return {
        "decision": "block",
        "reason": reason,
    }
