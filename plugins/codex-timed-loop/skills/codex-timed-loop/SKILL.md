---
name: codex-timed-loop
description: Install or refresh the global Codex loop runtime in ~/.codex for structured [[CODEX_LOOP ...]] activation with either min="..." or rounds="...". Do not use for unrelated hook authoring, generic plugin packaging, or repository-local bootstrap flows.
---

# Codex Timed Loop

## Procedures

**Step 1: Inspect the current global Codex state**
1. Read `~/.codex/config.toml` only if it already exists so you can explain whether `features.codex_hooks` was added or was already enabled.
2. Read `~/.codex/hooks.json` only if it already exists so you can describe whether non-managed hooks were preserved.

**Step 2: Install or refresh the managed files**
1. Execute `python3 ../../scripts/install.py`.
2. Read the command output and report which files were created or updated under `~/.codex/`.
3. Tell the user to restart Codex so the global hook configuration is reloaded.

**Step 3: Explain activation**
1. Tell the user that loop activation must be the first line of the prompt.
2. Tell the user that the header must contain exactly one limiter:
   - `[[CODEX_LOOP name="release-stress-qa" min="6h"]]`
   - `[[CODEX_LOOP name="release-stress-qa" rounds="3"]]`
3. Tell the user that the task prompt starts on the next line and remains the source task for every continuation.
4. Tell the user that loop state is persisted under `~/.codex/codex-timed-loop/loops/`.

## Error Handling

- If `../../scripts/install.py` fails because `~/.codex/config.toml` already defines inline `[hooks]`, remove or migrate the inline hook tables before retrying.
- If `../../scripts/install.py` fails because `python3` is unavailable, install Python 3 or run the installer from an environment where `python3` exists on `PATH`.
