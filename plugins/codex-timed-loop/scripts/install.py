#!/usr/bin/env python3

from __future__ import annotations

import shutil
import sys

from _installer_common import (
    codex_home,
    dump_json,
    ensure_codex_hooks_enabled,
    global_config_path,
    global_hooks_json_path,
    load_json,
    merge_hooks,
    runtime_config_path,
    runtime_hooks_dir,
    runtime_loops_dir,
    runtime_root,
    template_hooks_dir,
    template_hooks_path,
    template_runtime_config_path,
)


def install_hook_scripts() -> list[str]:
    hooks_dir = runtime_hooks_dir(create=True)
    copied: list[str] = []
    for template in sorted(template_hooks_dir().glob("*.py")):
        destination = hooks_dir / template.name
        shutil.copy2(template, destination)
        copied.append(str(destination))
    return copied


def install_runtime_config() -> bool:
    destination = runtime_config_path()
    if destination.exists():
        return False
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(template_runtime_config_path(), destination)
    return True


def install() -> list[str]:
    codex_root = codex_home()
    runtime_root(create=True)
    runtime_loops_dir(create=True)

    copied_hooks = install_hook_scripts()
    runtime_config_created = install_runtime_config()

    template_doc = load_json(template_hooks_path())
    hooks_json_path = global_hooks_json_path()
    existing_doc = load_json(hooks_json_path) if hooks_json_path.exists() else {"hooks": {}}
    merged = merge_hooks(existing_doc, template_doc)
    dump_json(hooks_json_path, merged)

    config_path = global_config_path()
    config_updated = ensure_codex_hooks_enabled(config_path)

    messages = [
        f"Installed {len(copied_hooks)} hook script(s) into {runtime_hooks_dir(create=False)}",
        f"Updated managed hook config at {hooks_json_path}",
        f"Ensured loop state directory exists at {runtime_loops_dir(create=False)}",
    ]
    if runtime_config_created:
        messages.append(f"Created optional runtime config at {runtime_config_path()}")
    else:
        messages.append(f"Preserved existing runtime config at {runtime_config_path()}")
    if config_updated:
        messages.append(f"Enabled features.codex_hooks in {config_path}")
    else:
        messages.append(f"features.codex_hooks was already enabled in {config_path}")
    messages.append(f"Global Codex home: {codex_root}")
    return messages


def main() -> int:
    try:
        messages = install()
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        return 1

    for message in messages:
        print(message)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
