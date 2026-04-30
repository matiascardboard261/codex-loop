#!/usr/bin/env python3

from __future__ import annotations

import shutil
import sys

from _installer_common import (
    dump_json,
    global_hooks_json_path,
    load_json,
    remove_managed_hooks,
    runtime_root,
    template_hooks_path,
)


def uninstall() -> list[str]:
    messages: list[str] = []
    template_doc = load_json(template_hooks_path())
    hooks_json_path = global_hooks_json_path()

    if hooks_json_path.exists():
        existing_doc = load_json(hooks_json_path)
        cleaned_doc, removed = remove_managed_hooks(existing_doc, template_doc)
        if removed:
            dump_json(hooks_json_path, cleaned_doc)
            messages.append(f"Removed managed hook registrations from {hooks_json_path}")
        else:
            messages.append(f"No managed hook registrations were present in {hooks_json_path}")
    else:
        messages.append(f"No global hooks.json found at {hooks_json_path}")

    runtime_dir = runtime_root(create=False)
    if runtime_dir.exists():
        shutil.rmtree(runtime_dir)
        messages.append(f"Removed managed runtime directory {runtime_dir}")
    else:
        messages.append(f"No managed runtime directory found at {runtime_dir}")

    messages.append("Left ~/.codex/config.toml unchanged, including features.codex_hooks.")
    return messages


def main() -> int:
    try:
        messages = uninstall()
    except Exception as exc:
        print(str(exc), file=sys.stderr)
        return 1

    for message in messages:
        print(message)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
