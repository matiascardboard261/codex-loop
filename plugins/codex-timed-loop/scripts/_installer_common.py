from __future__ import annotations

import json
import os
from pathlib import Path
import re
from typing import Any


HOOK_CONFIG_NAME = "hooks.json"
CONFIG_NAME = "config.toml"
PLUGIN_RUNTIME_NAME = "codex-timed-loop"
INLINE_HOOKS_RE = re.compile(r"^\s*\[\[?\s*hooks(?:[.\]])", re.MULTILINE)


def plugin_root() -> Path:
    return Path(__file__).resolve().parents[1]


def template_hooks_path() -> Path:
    return plugin_root() / "templates" / HOOK_CONFIG_NAME


def template_hooks_dir() -> Path:
    return plugin_root() / "templates" / "hooks"


def template_runtime_config_path() -> Path:
    return plugin_root() / "templates" / CONFIG_NAME


def codex_home() -> Path:
    configured = os.environ.get("CODEX_HOME", "").strip()
    if configured:
        return Path(configured).expanduser().resolve()
    return (Path.home() / ".codex").resolve()


def global_hooks_json_path() -> Path:
    return codex_home() / HOOK_CONFIG_NAME


def global_config_path() -> Path:
    return codex_home() / CONFIG_NAME


def runtime_root(create: bool = False) -> Path:
    path = codex_home() / PLUGIN_RUNTIME_NAME
    if create:
        path.mkdir(parents=True, exist_ok=True)
    return path


def runtime_hooks_dir(create: bool = False) -> Path:
    path = runtime_root(create=create) / "hooks"
    if create:
        path.mkdir(parents=True, exist_ok=True)
    return path


def runtime_loops_dir(create: bool = False) -> Path:
    path = runtime_root(create=create) / "loops"
    if create:
        path.mkdir(parents=True, exist_ok=True)
    return path


def runtime_config_path() -> Path:
    return runtime_root(create=False) / CONFIG_NAME


def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"{path} must contain a JSON object")
    return data


def dump_json(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def managed_commands(template_doc: dict[str, Any]) -> set[str]:
    commands: set[str] = set()
    hooks_root = template_doc.get("hooks", {})
    if not isinstance(hooks_root, dict):
        raise ValueError("template hooks.json must contain a top-level hooks object")
    for matcher_groups in hooks_root.values():
        if not isinstance(matcher_groups, list):
            raise ValueError("template hook event entries must be lists")
        for matcher_group in matcher_groups:
            hooks = matcher_group.get("hooks", [])
            if not isinstance(hooks, list):
                raise ValueError("template matcher hooks must be lists")
            for hook in hooks:
                if hook.get("type") == "command" and isinstance(hook.get("command"), str):
                    commands.add(hook["command"])
    return commands


def remove_managed_hooks(existing_doc: dict[str, Any], template_doc: dict[str, Any]) -> tuple[dict[str, Any], bool]:
    if not existing_doc:
        existing_doc = {"hooks": {}}
    hooks_root = existing_doc.setdefault("hooks", {})
    if not isinstance(hooks_root, dict):
        raise ValueError("existing hooks.json must contain a top-level hooks object")

    managed = managed_commands(template_doc)
    changed = False
    empty_events: list[str] = []
    for event_name, matcher_groups in list(hooks_root.items()):
        if not isinstance(matcher_groups, list):
            raise ValueError(f"hooks.{event_name} must be a list")
        filtered_groups = []
        for matcher_group in matcher_groups:
            hooks = matcher_group.get("hooks", [])
            if not isinstance(hooks, list):
                raise ValueError(f"hooks.{event_name}.hooks must be a list")
            remaining_hooks = [
                hook
                for hook in hooks
                if not (hook.get("type") == "command" and hook.get("command") in managed)
            ]
            if len(remaining_hooks) != len(hooks):
                changed = True
            if not remaining_hooks:
                continue
            next_group = dict(matcher_group)
            next_group["hooks"] = remaining_hooks
            filtered_groups.append(next_group)
        if filtered_groups:
            hooks_root[event_name] = filtered_groups
        else:
            empty_events.append(event_name)
    for event_name in empty_events:
        changed = True
        hooks_root.pop(event_name, None)
    return existing_doc, changed


def merge_hooks(existing_doc: dict[str, Any], template_doc: dict[str, Any]) -> dict[str, Any]:
    cleaned_doc, _ = remove_managed_hooks(existing_doc, template_doc)
    hooks_root = cleaned_doc.setdefault("hooks", {})
    for event_name, matcher_groups in template_doc["hooks"].items():
        merged_groups = hooks_root.setdefault(event_name, [])
        if not isinstance(merged_groups, list):
            raise ValueError(f"hooks.{event_name} must be a list")
        merged_groups.extend(matcher_groups)
    return cleaned_doc


def ensure_codex_hooks_enabled(config_path: Path) -> bool:
    if config_path.exists():
        text = config_path.read_text(encoding="utf-8")
    else:
        text = ""

    if INLINE_HOOKS_RE.search(text):
        raise RuntimeError(
            f"{config_path} already defines inline hooks. Remove inline [hooks] tables before running the codex-loop installer."
        )

    updated = False
    lines = text.splitlines()
    section_start = None
    section_end = None
    header_re = re.compile(r"^\s*\[([^\[\]]+)\]\s*$")
    for index, line in enumerate(lines):
        match = header_re.match(line)
        if not match:
            continue
        if match.group(1).strip() == "features":
            section_start = index
            section_end = len(lines)
            for probe in range(index + 1, len(lines)):
                if header_re.match(lines[probe]):
                    section_end = probe
                    break
            break

    if section_start is None:
        append_lines = []
        if lines and lines[-1].strip():
            append_lines.append("")
        append_lines.extend(["[features]", "codex_hooks = true"])
        lines.extend(append_lines)
        updated = True
    else:
        replaced = False
        for index in range(section_start + 1, section_end):
            if re.match(r"^\s*codex_hooks\s*=", lines[index]):
                if lines[index].strip() != "codex_hooks = true":
                    lines[index] = "codex_hooks = true"
                    updated = True
                replaced = True
                break
        if not replaced:
            lines.insert(section_end, "codex_hooks = true")
            updated = True

    output = "\n".join(lines).rstrip() + "\n"
    if output != text:
        config_path.parent.mkdir(parents=True, exist_ok=True)
        config_path.write_text(output, encoding="utf-8")
        return True
    return updated
